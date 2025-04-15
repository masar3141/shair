package shair

import (
	"context"
	"log/slog"
	"sync"
)

// Application manages the lifecycle of all shairers and serves as the main api
// for the UI. It handles service management, peer updates,
// transfer requests, and the storage location for received files.
type Application struct {
	logger *slog.Logger

	Shairer

	localDeviceName string // name by the device is discoverable by other peers
	saveDir         string // destination directory for received files

	// waits for all goroutines to finish
	wg   sync.WaitGroup
	stop context.CancelFunc
}

func NewApplication(logger *slog.Logger, localDeviceName string, saveDir string, sh Shairer) *Application {

	return &Application{
		logger: logger,

		Shairer: sh,

		localDeviceName: localDeviceName,
		saveDir:         saveDir,

		wg: sync.WaitGroup{},
	}
}

// Start initializes and runs a service, making the local device discoverable and enabling
// the transfer mechanism. It performs the following actions:
// - Makes the local device discoverable by other peers searching for that specific service,
// - Discovers peers registered to the same service, with updates sent to peerUpdateCh,
// - Enables a 4-step file transfer process:
//  1. The sender requests permission and sends information about the files to be transferred,
//  2. The user accepts or rejects the request,
//  3. If accepted, the sender proceeds to send the files,
//  4. The files are saved to the specified saveDir.
//
// The service runs until the provided context is canceled.
func (a *Application) Start(ctx context.Context, puCh chan<- PeerUpdate, trCh chan<- TransferRequest) {
	ctx, cancel := context.WithCancel(ctx)
	a.stop = cancel

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.Discover(ctx, puCh)
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.Announce(ctx, a.localDeviceName, a.saveDir, trCh)
	}()

	a.wg.Wait()
}

func (a *Application) Stop() {
	if a.stop == nil {
		return
	}
	a.stop()
	a.wg.Wait()
}

func (a *Application) SendFiles(ctx context.Context, target *Device, uploadProgressCh chan<- int, filepaths []string) error {
	return a.Shairer.SendFiles(ctx, target, uploadProgressCh, filepaths...)
}
