// TODO: progress is not used yet. the local implementation relies on io.Copy that
// reports the total bytes written once the transfer is done, we'll have to change
// for an implementation that reports incremental progress
package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/masar3141/shair"

	"github.com/dustin/go-humanize"
)

type receivingModel struct {
	sender       *shair.Device
	filePreviews []shair.FilePreview
	progressCh   <-chan int
	received     int
}

func newReceivingModel(sender *shair.Device, fp []shair.FilePreview, progressCh <-chan int) *receivingModel {
	return &receivingModel{
		sender:       sender,
		filePreviews: fp,
		progressCh:   progressCh,
	}
}

type downloadProgressMsg struct{ p int }

type receivingDoneMsg struct {
	received int
	expected int // set in root. TODO: investigate a less confusing pattern
}

func (m receivingModel) listenDownloadProgressCmd() tea.Msg {
	p, ok := <-m.progressCh
	if !ok {
		return receivingDoneMsg{
			received: m.received,
		}
	}

	m.received += p
	return downloadProgressMsg{p}
}

func (m *receivingModel) Init() tea.Cmd {
	return m.listenDownloadProgressCmd
}

func (m *receivingModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, m.listenDownloadProgressCmd)

	return m, tea.Batch(cmds...)
}

func (m receivingModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString("Receiving Files\n\n")

	// File list with sizes
	if len(m.filePreviews) == 0 {
		b.WriteString("No files to display.\n")
		return b.String()
	}

	for i, f := range m.filePreviews {
		line := fmt.Sprintf("  %2d. %-30s %10s\n", i+1, f.Name, humanize.Bytes(uint64(f.Size)))
		b.WriteString(line)
	}

	// Summary
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total files: %d\n", len(m.filePreviews)))

	return b.String()
}
