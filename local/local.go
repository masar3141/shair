// this file provides an implementation of shairer for a local
// data transfer. Using Mdns as the service broadcaster and discoverer
// and a local tcp server for the data transfer
package local

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/masar3141/shair"
)

type tcpInfo struct {
	ip   net.IP
	port int
}

// LocalShairer implements shair.Shairer
type LocalShairer struct {
	logger *slog.Logger

	port int // port on which the tcp server will be listening

	// serviceToDevice maps service names (shair.Device.Name) to discovered devices.
	// Usage:
	// - Add when a peer is discovered via mDNS.
	// - Retrieve device on peer removal to delete to send along peerCh
	// - Remove when peer is lost
	serviceToDevice map[string]*shair.Device
	smu             sync.Mutex // protects the map

	// deviceToTCP maps devices to their TCP connection details.
	// Usage:
	// - Store when a peer is discovered.
	// - Retrieve tcp info with device from ui when want to send file
	// - Remove using serviceToDevice when the peer is lost.
	deviceToTCP map[*shair.Device]tcpInfo
	dmu         sync.Mutex // protects the map
}

func NewLocalShairer(logger *slog.Logger, port int) *LocalShairer {
	return &LocalShairer{
		logger: logger,
		port:   port,

		serviceToDevice: make(map[string]*shair.Device),
		smu:             sync.Mutex{},

		deviceToTCP: make(map[*shair.Device]tcpInfo),
		dmu:         sync.Mutex{},
	}
}

// Discover continuously listens for mDNS service announcements from other nodes on the local network.
// It filters services matching "_shair._tcp" and sends notifications through a channel
// indicating whether a service was added or removed.
// This function runs indefinitely until the provided context is canceled.
func (l *LocalShairer) Discover(ctx context.Context, peerCh chan<- shair.PeerUpdate) {
	l.discover(ctx, peerCh)
}

func (l *LocalShairer) Announce(
	ctx context.Context,
	localDeviceName string,
	saveDir string,
	transferRequestCh chan<- shair.TransferRequest,
) {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.broadcast(ctx, localDeviceName)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.listen(ctx, saveDir, transferRequestCh)
	}()

	wg.Wait()
}

func (l *LocalShairer) SendFiles(ctx context.Context, target *shair.Device, updloadProgressCh chan<- int, filepaths ...string) error {
	// try to open and stat the files and store them in arrays
	numFiles := len(filepaths)
	files := make([]*os.File, numFiles)
	fileInfos := make([]os.FileInfo, numFiles)

	for idx, fp := range filepaths {
		file, err := os.Open(fp)
		if err != nil {
			return shair.NewError(shair.StatFileError, fmt.Sprintf("cannot open file %s", file.Name()), err)
		}

		info, err := file.Stat()
		if err != nil {
			return shair.NewError(shair.StatFileError, fmt.Sprintf("cannot stat file %s", file.Name()), err)
		}

		files[idx] = file
		fileInfos[idx] = info
	}

	// connect to the server
	targetTcpInfo := l.deviceToTCP[target]
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", targetTcpInfo.ip, targetTcpInfo.port))
	if err != nil {
		return shair.NewError(shair.UnexpectedError, fmt.Sprintf("cannot dial with server %s:%d", targetTcpInfo.ip, targetTcpInfo.port), err)
	}

	s := newSender(ctx, conn, files)
	// TODO: Check the connection state in a separate goroutine and report to ErrCh if the destination has closed the connection.
	// See: https://github.com/golang/go/issues/15735#issuecomment-266574151 for feasability
	//
	// Implications:
	// - If the sender exits after writing the file header (before the receiver has explicitly accepted the connection),
	//   the TCP stack may still deliver buffered data to the receiver due to OS-level socket buffering.
	// -> we should drop the accept mechanism receiver's side as soon as the sender quits

	// prepare the header and send it immediately
	hdr := newHeader(fileInfos...)
	w, err := s.writeHeaderToConn(hdr)
	if err != nil && w != int(hdr.headerSize) {
		// TODO: better error handling, maybe switch on the error or create another shair.WriteHeader error
		return shair.NewError(shair.UnexpectedError, "failed to write header on conn", err)
	}

	// rend confirmation bit sent by dest on conn
	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	if err != nil || n != 1 {
		return shair.NewError(shair.UnexpectedError, "failed to read confirmation bit", err)
	}

	// 0 means rejected
	if buf[0] == 0 {
		return shair.NewError(shair.TransferRejected, "cannot send file", err)
	}

	err = s.sendFiles(updloadProgressCh)
	if err != nil {
		return shair.NewError(shair.SendFileError, "cannot send file", err)
	}

	return nil
}
