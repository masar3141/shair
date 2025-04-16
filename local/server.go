package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"encoding/binary"

	"github.com/masar3141/shair"
)

func (s *LocalShairer) listen(
	ctx context.Context,
	saveDir string,
	transferRequestCh chan<- shair.TransferRequest,
) {
	var lnc net.ListenConfig
	ln, err := lnc.Listen(ctx, "tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		panic(fmt.Errorf("unable to start server: %w", err))
	}

	// Close the listener when context is cancelled
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return // listener closed due to ctx cancellation
			default:
				continue
			}
		}

		//TODO: error handling
		_ = s.handleRequest(ctx, saveDir, conn, transferRequestCh)
	}
}

func (s *LocalShairer) handleRequest(
	ctx context.Context,
	saveDir string,
	conn net.Conn,
	transferRequestCh chan<- shair.TransferRequest,
) error {
	defer conn.Close()

	// read the header to send transferRequest to ui
	hdr := s.readHeader(conn)

	// get the sender's hostname with a dns lookup
	// TODO: remove and let client send its name within the header
	remoteAddr := conn.RemoteAddr() // ← this is the client's address
	remoteIP, _, err := net.SplitHostPort(remoteAddr.String())
	names, err := net.LookupAddr(remoteIP)
	if err != nil {
		// in my local setup virtual machines are not register in the router's dns table
		// if the lookup fails, we set the name to unknown.
		names = append(names, "unknown")
	}

	// send preview of requested file transfer to ui
	fp := make([]shair.FilePreview, hdr.numFiles)
	for i := 0; i < int(hdr.numFiles); i++ {
		fp[i] = shair.FilePreview{
			Name: hdr.names[i],
			Size: uint64(hdr.fileSize[i]),
		}
	}

	downloadProgressCh := make(chan int)
	defer close(downloadProgressCh)

	acceptCh := make(chan bool)

	// notify ui transferRequest
	sender := &shair.Device{Name: names[0], DiscoveredOn: shair.Local}
	transferRequestCh <- shair.TransferRequest{
		Sender:       sender,
		FilePreviews: fp,
		AcceptCh:     acceptCh,
		ProgressCh:   downloadProgressCh,
	}

	// wait for user accepts or context cancelled
	select {
	case <-ctx.Done():
		return nil
	case accepts := <-acceptCh:
		if !accepts {
			// send to sender reject bit
			conn.Write([]byte{0})
			// TODO: return custom err?
			return nil
		}
	}

	// send to sender confirmation bit
	conn.Write([]byte{1})

	// save the files
	for i := 0; i < int(hdr.numFiles); i++ {
		size := hdr.fileSize[i]
		read, err := s.readAndSaveFile(conn, hdr.names[i], size, saveDir, downloadProgressCh)

		if err != nil {
			return fmt.Errorf("encode: can't read the file %s error: %w\n", hdr.names[i], err)
		}

		if read != size {
			return fmt.Errorf("encode: didn't read enough bytes for file %s", hdr.names[i])
		}
	}

	return nil
}

func (s *LocalShairer) readAndSaveFile(conn net.Conn, name string, size int64, saveDir string, downloadProgressCh chan<- int) (int64, error) {
	//	create the empty file that will hold the received file
	file, err := os.Create(filepath.Join(saveDir, name))
	if err != nil {
		return 0, err
	}

	trdr := io.TeeReader(conn, shair.NewProgressWriter(downloadProgressCh))

	n, err := io.CopyN(file, trdr, size)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("failed to read from conn: %w", err)
	}

	return n, nil
}

func (s *LocalShairer) readHeader(conn net.Conn) *header {
	offset := 0

	hdrSizeBytes := make([]byte, 2)
	_, err := conn.Read(hdrSizeBytes)
	if err != nil {
		panic(fmt.Errorf("err read: %w", err))
	}

	offset += 2

	hdrSize := binary.BigEndian.Uint16(hdrSizeBytes)

	hdr := bytes.NewBuffer(make([]byte, 0))
	hdr.Write(hdrSizeBytes)

	read := int64(0)
	for read < int64(hdrSize)-int64(offset) {
		n, err := io.CopyN(hdr, conn, int64(hdrSize)-int64(offset))
		if err != nil {

		}
		read += n
	}

	return decodeHeader(hdr.Bytes())
}
