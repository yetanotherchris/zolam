package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
)

// OutputLineMsg carries a single line of output from a running operation.
type OutputLineMsg struct {
	Line string
}

// OperationDoneMsg signals that the running operation has completed.
type OperationDoneMsg struct {
	Err error
}

// ProgressModel displays streaming output from a long-running operation.
type ProgressModel struct {
	viewport viewport.Model
	content  string
	done     bool
	title    string
	err      error
}

// NewProgressModel creates a ProgressModel with the given title.
func NewProgressModel(title string) ProgressModel {
	vp := viewport.New(80, 20)
	vp.SetContent("")

	return ProgressModel{
		viewport: vp,
		title:    title,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case OutputLineMsg:
		if m.content != "" {
			m.content += "\n"
		}
		m.content += msg.Line
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()
		return m, nil

	case OperationDoneMsg:
		m.done = true
		m.err = msg.Err
		if msg.Err != nil {
			if m.content != "" {
				m.content += "\n"
			}
			m.content += ErrorStyle.Render(fmt.Sprintf("Error: %v", msg.Err))
			m.viewport.SetContent(m.content)
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4 // leave room for title and status
		return m, nil

	case tea.KeyMsg:
		if m.done {
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ProgressModel) View() string {
	s := TitleStyle.Render(m.title) + "\n\n"
	s += m.viewport.View() + "\n\n"

	if m.err != nil {
		s += ErrorStyle.Render("Operation failed. Press any key to continue.")
	} else if m.done {
		s += SuccessStyle.Render("Done. Press any key to continue.")
	} else {
		s += StatusBarStyle.Render("Running...")
	}

	return s
}
