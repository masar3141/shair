package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/masar3141/shair"
)

// shared state between models
type store struct {
	// set when listModel kp enter
	// used after fileinput model kp tab
	destForSend *shair.Device

	// set when listModel kp y after request
	// used after sending screen returns doneReceivingMsg to compare
	// if transfer is complete
	transferRequestTotSize int
}

type state uint8

const (
	list state = iota
	fileInput
	receiving
	sending
	quit
)

type Sender interface {
	SendFiles(ctx context.Context, target *shair.Device, progressCh chan<- int, filepaths []string) error
}

type rootModel struct {
	sender Sender

	state  state
	models map[state]tea.Model

	dest *shair.Device // device selected in list model

	//  store that holds shared state
	store store
}

func newRootModel(sender Sender, quitter Quitter) *rootModel {
	return &rootModel{
		sender: sender,
		state:  list,
		models: map[state]tea.Model{list: newListModel(), fileInput: newFileInputModel(), quit: newQuitModel(quitter)},
		dest:   &shair.Device{},
	}
}

// https://github.com/charmbracelet/bubbletea/issues/1135
// peerUpdate and transferRequest events are emitted by the backend
// and handled in events.go, which forwards them directly to the program.
// This avoids awkward command chains that would otherwise need to be returned
// and re-invoked after each model Update.
type peerUpdateMsg shair.PeerUpdate
type transferRequestMsg shair.TransferRequest

type sendingDoneMsg struct{}
type errMsg error

func (m rootModel) sendFilesCmd(uploadProgressCh chan<- int, dest *shair.Device, fp []string) tea.Cmd {
	return func() tea.Msg {
		err := m.sender.SendFiles(context.Background(), dest, uploadProgressCh, fp)
		if err != nil {
			return errMsg(err)
		}

		return sendingDoneMsg{}
	}
}

func (m *rootModel) Init() tea.Cmd { return nil }

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// TODO: understand why ctrl-c doesn't work and make it work
		case "esc":
			m.state = quit
		}

	case peerUpdateMsg:
		m.models[list], cmd = m.models[list].Update(msg)
		return m, cmd

	case transferRequestMsg:
		m.models[list], cmd = m.models[list].Update(msg)
		return m, cmd

	case changePageListToInputMsg:
		m.store.destForSend = msg.dest
		m.state = fileInput

	case changePageListToReceivingMsg:
		rqTotSize := 0
		for _, fp := range msg.filePreviews {
			rqTotSize += int(fp.Size)
		}
		m.store.transferRequestTotSize = rqTotSize
		m.models[receiving] = newReceivingModel(msg.sender, msg.filePreviews, msg.downloadProgressCh)
		m.state = receiving

	case changePageInputToSendingMsg:
		uploadProgressCh := make(chan int)
		m.models[sending] = newSendingModel(m.store.destForSend, msg.filePreviews, uploadProgressCh)
		m.state = sending
		cmd = m.sendFilesCmd(uploadProgressCh, m.store.destForSend, msg.filePaths)

	case errMsg:
		// go back to list model and display the error
		m.models[list], cmd = m.models[list].Update(msg)
		m.state = list
		return m, cmd

	case sendingDoneMsg:
		m.models[list], cmd = m.models[list].Update(msg)
		m.state = list
		return m, cmd

	case receivingDoneMsg:
		msg.expected = m.store.transferRequestTotSize
		m.models[list], cmd = m.models[list].Update(msg)
		m.state = list

	}

	var mcmd tea.Cmd
	m.models[m.state], mcmd = m.models[m.state].Update(msg)
	cmds = append(cmds, mcmd)

	return m, tea.Batch(append(cmds, cmd)...)
}

func (m *rootModel) View() string {
	return m.models[m.state].View()
}
