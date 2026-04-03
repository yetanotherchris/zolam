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

// allExtensions is the list of supported extensions available for selection.
var allExtensions = []string{
	".md", ".pdf", ".docx", ".txt",
	".py", ".cs", ".js", ".ts",
	".json", ".yml", ".yaml",
}

// IngestModel is the bubbletea model for configuring an ingest run.
type IngestModel struct {
	directoryInput textinput.Model
	step           int // 0=enter directory, 1=select extensions, 2=confirm
	directories    []string
	extSelected    []bool
	extCursor      int
	err            error
}

// NewIngestModel creates a new IngestModel with all extensions selected.
func NewIngestModel() IngestModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/directory"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	selected := make([]bool, len(allExtensions))
	for i := range selected {
		selected[i] = true
	}

	return IngestModel{
		directoryInput: ti,
		step:           0,
		extSelected:    selected,
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
				// move to confirm
				m.step = 2
			case 2:
				return m, func() tea.Msg {
					return StartIngestMsg{
						Directories: m.directories,
						Extensions:  m.selectedExtensions(),
					}
				}
			}

		case "tab":
			if m.step == 0 && len(m.directories) > 0 {
				m.step = 1
			} else if m.step == 1 {
				m.step = 2
			}

		case " ":
			if m.step == 1 {
				m.extSelected[m.extCursor] = !m.extSelected[m.extCursor]
			}

		case "a":
			if m.step == 1 {
				allSelected := true
				for _, s := range m.extSelected {
					if !s {
						allSelected = false
						break
					}
				}
				for i := range m.extSelected {
					m.extSelected[i] = !allSelected
				}
			}

		case "up", "k":
			if m.step == 1 && m.extCursor > 0 {
				m.extCursor--
			}

		case "down", "j":
			if m.step == 1 && m.extCursor < len(allExtensions)-1 {
				m.extCursor++
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

func (m IngestModel) selectedExtensions() string {
	var exts []string
	for i, ext := range allExtensions {
		if m.extSelected[i] {
			exts = append(exts, ext)
		}
	}
	return strings.Join(exts, ", ")
}

func (m IngestModel) View() string {
	s := TitleStyle.Render("Ingest Configuration") + "\n\n"

	switch m.step {
	case 0:
		s += "Enter directories to ingest (press Enter to add, Tab to continue):\n\n"
		if len(m.directories) > 0 {
			s += "Added directories:\n"
			for _, d := range m.directories {
				s += SuccessStyle.Render(fmt.Sprintf("  + %s", d)) + "\n"
			}
			s += "\n"
		}
		s += m.directoryInput.View() + "\n"
		s += "\n" + HelpStyle.Render("enter: add directory | tab: continue | esc: back to menu")

	case 1:
		s += "Select extensions (space to toggle, a to toggle all):\n\n"
		for i, ext := range allExtensions {
			cursor := "  "
			if i == m.extCursor {
				cursor = "> "
			}
			check := "[ ]"
			if m.extSelected[i] {
				check = "[x]"
			}
			line := fmt.Sprintf("%s%s %s", cursor, check, ext)
			if i == m.extCursor {
				s += SelectedMenuItemStyle.Render(line) + "\n"
			} else {
				s += line + "\n"
			}
		}
		s += "\n" + HelpStyle.Render("space: toggle | a: toggle all | tab/enter: continue | esc: back")

	case 2:
		s += "Directories:\n"
		for _, d := range m.directories {
			s += SuccessStyle.Render(fmt.Sprintf("  + %s", d)) + "\n"
		}
		s += "\n"
		s += fmt.Sprintf("Extensions: %s\n", m.selectedExtensions())
		s += "\n" + HelpStyle.Render("enter: start ingest | esc: back to menu")
	}

	return s
}
