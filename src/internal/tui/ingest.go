package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

// StartIngestMsg is sent when the user confirms the ingest configuration.
type StartIngestMsg struct {
	Directories []string
	Extensions  string
}

// BackToMenuMsg is sent when the user wants to return to the main menu.
type BackToMenuMsg struct{}

// IngestModel is the bubbletea model for configuring an ingest run.
type IngestModel struct {
	directoryInput textinput.Model
	extensions     string
	step           int // 0=enter directory, 1=confirm extensions, 2=ready to run
	directories    []string
	err            error
}

// NewIngestModel creates a new IngestModel with the given default extensions.
func NewIngestModel(defaultExtensions string) IngestModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/directory"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	return IngestModel{
		directoryInput: ti,
		extensions:     defaultExtensions,
		step:           0,
	}
}

func (m IngestModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m IngestModel) Update(msg tea.Msg) (IngestModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToMenuMsg{} }

		case "enter":
			switch m.step {
			case 0:
				val := strings.TrimSpace(m.directoryInput.Value())
				if val != "" {
					m.directories = append(m.directories, val)
					m.directoryInput.SetValue("")
				}
			case 1:
				m.step = 2
				return m, func() tea.Msg {
					return StartIngestMsg{
						Directories: m.directories,
						Extensions:  m.extensions,
					}
				}
			}

		case "tab":
			if m.step == 0 && len(m.directories) > 0 {
				m.step = 1
			}
		}
	}

	if m.step == 0 {
		var cmd tea.Cmd
		m.directoryInput, cmd = m.directoryInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m IngestModel) View() string {
	s := TitleStyle.Render("Ingest Configuration") + "\n\n"

	switch m.step {
	case 0:
		s += "Enter directories to ingest (press Enter to add, Tab to continue):\n\n"
		if len(m.directories) > 0 {
			s += "Added directories:\n"
			for _, d := range m.directories {
				s += SuccessStyle.Render(fmt.Sprintf("  ✓ %s", d)) + "\n"
			}
			s += "\n"
		}
		s += m.directoryInput.View() + "\n"
		s += "\n" + HelpStyle.Render("enter: add directory • tab: continue • esc: back to menu")

	case 1:
		s += "Directories:\n"
		for _, d := range m.directories {
			s += SuccessStyle.Render(fmt.Sprintf("  ✓ %s", d)) + "\n"
		}
		s += "\n"
		s += fmt.Sprintf("Extensions: %s\n", m.extensions)
		s += "\n" + HelpStyle.Render("enter: start ingest • esc: back to menu")

	case 2:
		s += SuccessStyle.Render("Starting ingest...") + "\n"
	}

	return s
}
