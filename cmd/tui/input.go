// file input model
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/masar3141/shair"
)

type fileInputModel struct {
	textarea         textarea.Model
	invalidFilepaths []string
	inputPaths       []string
}

func newFileInputModel() *fileInputModel {
	ti := textarea.New()
	ti.Placeholder = "/home/foo/..."
	ti.Focus()

	return &fileInputModel{
		textarea:         ti,
		invalidFilepaths: make([]string, 0),
	}
}

type validateFilepathsMsg struct {
	// Whether any invalid file paths were found
	invalid bool

	// Original file paths provided by the user; used later in SendFiles and retained in fileInputModel
	fp []string

	// FileInfo for each valid file path; used later for constructing filePreview in transfer phase
	fi []os.FileInfo

	// Indexes in fp that failed validation
	invalids []int
}

func validateFilepathsCmd(fp []string) tea.Cmd {
	return func() tea.Msg {
		invalid := false
		fi, invalidIdx := validateFilepaths(fp)
		if len(invalidIdx) != 0 {
			invalid = true
		}

		// TODO: return smallest sized files first. Order will be preserved
		// in sending, so receiver can receive files as soon as possible
		return validateFilepathsMsg{
			invalid:  invalid,
			fp:       fp,
			fi:       fi,
			invalids: invalidIdx,
		}
	}
}

// helper function that checks whether an array of filepaths contains any invalids (not found on disk)
// it calls os.Stat for each file and return the array of FileInfo given by each call to os.Stat since we need
// it for later use. It also returns an array containing indexes in fp that corresponds to invalid paths
//
// As of now, it also consider directory paths as invalid. Trying to send a directory via Shairer's SendFiles
// function will result in an error when it tries to read the content of the directory.
func validateFilepaths(fp []string) ([]os.FileInfo, []int) {
	invalidIdx := make([]int, 0)
	fis := make([]os.FileInfo, 0)
	for i, f := range fp {
		fi, err := os.Stat(f)
		// TODO: display a meaningful message when input points to a directory
		// and eventually accept the transfer of directories
		if err != nil || fi.IsDir() {
			invalidIdx = append(invalidIdx, i)
		} else {
			fis = append(fis, fi)
		}
	}

	return fis, invalidIdx
}

type changePageInputToSendingMsg struct {
	filePreviews []shair.FilePreview // constructed with the return of validateFilepaths
	filePaths    []string            // user input, split on \n
}

func changePageInputToSendingCmd(fp []shair.FilePreview, filePathsToSend []string) tea.Cmd {
	return func() tea.Msg {
		return changePageInputToSendingMsg{fp, filePathsToSend}
	}
}

func (m *fileInputModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *fileInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyTab:
			ps := make([]string, 0)
			for _, p := range strings.Split(m.textarea.Value(), "\n") {
				ps = append(ps, p)
			}
			m.inputPaths = ps
			return m, validateFilepathsCmd(m.inputPaths)
		}

	case validateFilepathsMsg:
		if msg.invalid {
			// clean previous invalid paths and display the new ones in the footer
			m.invalidFilepaths = make([]string, len(msg.fi))
			for _, i := range msg.invalids {
				m.invalidFilepaths = append(m.invalidFilepaths, m.inputPaths[i])
			}

		} else {
			// file path validation succeed
			// clean err message
			m.invalidFilepaths = make([]string, 0)

			// construct filePreview from fileInfo
			fp := make([]shair.FilePreview, len(msg.fi))
			for idx, i := range msg.fi {
				fp[idx] = shair.FilePreview{Name: i.Name(), Size: uint64(i.Size())}
			}

			// go to transfering
			cmd = changePageInputToSendingCmd(fp, m.inputPaths)
		}

	}

	var tcmd tea.Cmd
	m.textarea, tcmd = m.textarea.Update(msg)
	cmds = append(cmds, tcmd)

	return m, tea.Batch(append(cmds, cmd)...)
}

func (m *fileInputModel) View() string {
	f := "(esc) quit (tab) send"
	if len(m.invalidFilepaths) != 0 {
		f += fmt.Sprintf("   ---   cannot find files: %v", m.invalidFilepaths)
	}

	return fmt.Sprintf(
		"Pick files to send\n\n%s\n\n%s",
		m.textarea.View(),
		f,
	) + "\n\n"
}
