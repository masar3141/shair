package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/masar3141/shair"
	"github.com/masar3141/shair/local"

	"github.com/brutella/dnssd/log"
)

func init() {
	log.Info.Disable()
	log.Debug.Disable()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	app := shair.NewApplication(
		logger,
		hostname,
		filepath.Join(dirname, "/Downloads"), // TODO: need flag
		local.NewLocalShairer(logger, 8085),
	)

	pgrm := tea.NewProgram(newRootModel(app, app))

	peerUpdateCh := make(chan shair.PeerUpdate)
	transferRequestCh := make(chan shair.TransferRequest)

	go app.Start(context.Background(), peerUpdateCh, transferRequestCh)

	go listenAndForwardPeerUpdate(pgrm, peerUpdateCh)
	go listenAndForwardTransferRequest(pgrm, transferRequestCh)

	_, err = pgrm.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
