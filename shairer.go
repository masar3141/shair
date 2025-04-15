package shair

import (
	"context"
	"net"
)

type SvcType int

const (
	Bluetooth SvcType = iota
	Local
	Remote
)

func (s SvcType) String() string {
	var str string
	switch s {
	case Bluetooth:
		str = "Bluetooth"
	case Local:
		str = "Local"
	case Remote:
		str = "Remote"
	}
	return str
}

// The Shairer interface defines the methods for peer discovery, local device advertisement,
// and file transfer management.
type Shairer interface {
	// Discover listens for service announcements from other nodes.
	// It sends updates through the channel when a matching service is discovered or removed.
	// The function runs until the context is canceled or an error occurs.
	Discover(context.Context, chan<- PeerUpdate)

	// Announce makes the local device discoverable on the network with given name by advertising its service.
	// It listens for incoming transfer requests and saves the received files to the specified `saveDir`.
	// This function runs until the provided context is canceled or an error occurs.
	Announce(ctx context.Context, localDeviceName string, saveDir string, transferRequestCh chan<- TransferRequest)

	// SendFiles sends one or more files to the specified receiver.
	SendFiles(ctx context.Context, target *Device, progressCh chan<- int, filepaths ...string) error
}

// struct containing info related to a device  discovered on a local network
type LocalInfo struct {
	IP      net.IP
	SvcPort int // port on which the tcp server is listening
}

type Device struct {
	Name         string
	DiscoveredOn SvcType // holds the service type on which the device was discovered

	LocalInfo
}

type PeerStatus = uint8

const (
	Discovered PeerStatus = iota
	Removed
)

type PeerUpdate struct {
	Peer   *Device // keep the pointer here so we can index maps with the pointer. see local.LocalShairer.foundDevices
	Status PeerStatus
}

type FilePreview struct {
	Name string
	Size uint64
}

type TransferRequest struct {
	Sender       *Device
	FilePreviews []FilePreview
	AcceptCh     chan<- bool
	ProgressCh   chan int
}
