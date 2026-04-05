package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type PasswordSubmitMsg struct {
	Password string
}

type PasswordModel struct {
	input  textinput.Model
	prompt string
}

func NewPasswordModel(prompt string) PasswordModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 256
	ti.Width = 60
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.Focus()

	return PasswordModel{
		input:  ti,
		prompt: prompt,
	}
}

func (m PasswordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PasswordModel) Update(msg tea.Msg) (PasswordModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, func() tea.Msg {
				return PasswordSubmitMsg{Password: m.input.Value()}
			}
		case "esc":
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m PasswordModel) View() string {
	s := TitleStyle.Render(m.prompt) + "\n\n"
	s += "  " + m.input.View() + "\n\n"
	s += HelpStyle.Render("enter: submit | esc: cancel")
	return s
}
