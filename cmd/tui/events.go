// this file defines function that listens event from the backend and forward them to the bubbletea event loop
// see https://github.com/charmbracelet/bubbletea/issues/1135 for a better way to do that
package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/masar3141/shair"
)

func listenAndForwardPeerUpdate(p *tea.Program, puCh <-chan shair.PeerUpdate) {
	for pu := range puCh {
		p.Send(peerUpdateMsg(pu))
	}
}

func listenAndForwardTransferRequest(p *tea.Program, trCh <-chan shair.TransferRequest) {
	for tr := range trCh {
		p.Send(transferRequestMsg(tr))
	}
}
