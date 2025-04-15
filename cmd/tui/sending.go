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

type sendingModel struct {
	receiver     *shair.Device // the other user, either the receiver or the sender
	filePreviews []shair.FilePreview
	progressCh   <-chan int
}

func newSendingModel(receiver *shair.Device, fp []shair.FilePreview, progressCh <-chan int) *sendingModel {
	return &sendingModel{
		receiver:     receiver,
		filePreviews: fp,
		progressCh:   progressCh,
	}
}

type uploadProgressMsg struct{ p int }

func (m sendingModel) listenUploadProgressCmd() tea.Msg {
	p := <-m.progressCh
	return uploadProgressMsg{p}
}

func (m *sendingModel) Init() tea.Cmd {
	return m.listenUploadProgressCmd
}

func (m *sendingModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, m.listenUploadProgressCmd)

	return m, tea.Batch(cmds...)
}

func (m sendingModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString("Sending Files\n\n")

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
