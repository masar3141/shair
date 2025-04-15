// quitting screen
package main

import tea "github.com/charmbracelet/bubbletea"

type Quitter interface {
	Stop()
}

type quitModel struct {
	Quitter
}

func newQuitModel(q Quitter) quitModel {
	return quitModel{q}
}

func (m *quitModel) quitCmd() tea.Msg {
	m.Stop()
	return tea.Quit()
}

func (m quitModel) Init() tea.Cmd {
	return nil
}

func (m quitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.quitCmd
}

func (m quitModel) View() string {
	return "quitting..."
}
