package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/masar3141/shair"
)

const (
	columnFmt = "%-3s %-20s %-10s %-15s %-5s\n"
)

type transferRequest struct {
	// set by update when transferRequest received
	acceptCh           chan<- bool
	downloadProgressCh <-chan int
	filePreviews       []shair.FilePreview
	requester          *shair.Device
}

type listModel struct {
	// ui
	columns             string
	footer              string // footer that is actually displayed
	baseFooter          string
	additionalMsgFooter string // message added at the end of the footer
	cursor              int

	// discovered peers
	peers []*shair.Device

	transferRequest transferRequest
}

func newListModel() *listModel {
	f := "\n(esc) Quit, (enter) Send, (k) Up, (j) Down"

	return &listModel{
		columns:    fmt.Sprintf(columnFmt, " ", "Device", "On", "IP", "Port"),
		footer:     f,
		baseFooter: f,
		peers:      make([]*shair.Device, 0),
	}
}

type changePageListToInputMsg struct {
	dest *shair.Device
}

func changePageListToInputCmd(dest *shair.Device) tea.Cmd {
	return func() tea.Msg {
		return changePageListToInputMsg{dest}
	}
}

type changePageListToReceivingMsg struct {
	filePreviews       []shair.FilePreview
	downloadProgressCh <-chan int
	sender             *shair.Device
}

func changePageListToReceivingCmd(fp []shair.FilePreview, downloadProgressCh <-chan int, sender *shair.Device) tea.Cmd {
	return func() tea.Msg {
		return changePageListToReceivingMsg{fp, downloadProgressCh, sender}
	}
}

func (m *listModel) Init() tea.Cmd {
	return nil
}

func (m *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "k", "ctrl-p", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "j", "ctrl-n", "down":
			if m.cursor < len(m.peers)-1 {
				m.cursor++
			}

		case "enter":
			return m, changePageListToInputCmd(m.peers[m.cursor])

		case "y":
			if m.transferRequest.acceptCh != nil { // if transfer requested
				m.transferRequest.acceptCh <- true
				close(m.transferRequest.acceptCh)
				m.transferRequest.acceptCh = nil
				m.additionalMsgFooter = ""
				return m, changePageListToReceivingCmd(
					m.transferRequest.filePreviews,
					m.transferRequest.downloadProgressCh,
					m.transferRequest.requester,
				)
			}

		case "n":
			if m.transferRequest.acceptCh != nil {
				m.transferRequest.acceptCh <- false
				close(m.transferRequest.acceptCh)
				m.transferRequest.acceptCh = nil
				m.additionalMsgFooter = ""
			}
		}

	case peerUpdateMsg:
		if msg.Status == shair.Discovered {
			m.peers = append(m.peers, msg.Peer)

		} else {
			m.peers = slices.DeleteFunc(m.peers, func(p *shair.Device) bool { return p == msg.Peer })
		}
		return m, cmd

	case transferRequestMsg:
		m.transferRequest.acceptCh = msg.AcceptCh
		m.transferRequest.downloadProgressCh = msg.ProgressCh
		m.transferRequest.filePreviews = msg.FilePreviews
		m.additionalMsgFooter = fmt.Sprintf(" (y/n) %s wants to transfer %d files", msg.Sender.Name, len(m.transferRequest.filePreviews))

	case errMsg:
		if errors.Is(msg, shair.TransferRejected) {
			// if dest rejects transfer, go back to list page and inform the user
			m.additionalMsgFooter = fmt.Sprintf(" --- %s didn't accept the files", m.peers[m.cursor].Name)
		} else {
			// TODO: probably a good thing to send a generic error message to the ui
			m.additionalMsgFooter = fmt.Sprintf(" --- %s ", msg.Error())
		}

	case sendingDoneMsg:
		m.additionalMsgFooter = " --- transfer done"

	case receivingDoneMsg:
		if msg.expected != msg.received {
			m.additionalMsgFooter = " --- transfer incomplete"
		} else {
			m.additionalMsgFooter = " --- files received"
		}
	}

	return m, cmd
}

func (m *listModel) View() string {
	//TODO: better string concatenation
	s := m.columns

	var selected string
	for idx, p := range m.peers {
		if idx == m.cursor {
			selected = ">"
		} else {
			selected = " "
		}
		s += fmt.Sprintf(columnFmt, selected, p.Name, p.DiscoveredOn.String(), p.LocalInfo.IP, strconv.Itoa(p.LocalInfo.SvcPort))
	}

	s += m.footer + m.additionalMsgFooter

	return s
}
