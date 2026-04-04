package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yetanotherchris/zolam/internal/domain"
)

type settingsField struct {
	label string
	get   func(*domain.Config) string
	set   func(*domain.Config, string)
}

var settingsFields = []settingsField{
	{
		label: "Collection Name",
		get:   func(c *domain.Config) string { return c.CollectionName },
		set:   func(c *domain.Config, v string) { c.CollectionName = v },
	},
	{
		label: "Data Dir",
		get:   func(c *domain.Config) string { return c.DataDir },
		set:   func(c *domain.Config, v string) { c.DataDir = v },
	},
	{
		label: "Rclone Source",
		get:   func(c *domain.Config) string { return c.RcloneSource },
		set:   func(c *domain.Config, v string) { c.RcloneSource = v },
	},
	{
		label: "Rclone Config Dir",
		get:   func(c *domain.Config) string { return c.RcloneConfigDir },
		set:   func(c *domain.Config, v string) { c.RcloneConfigDir = v },
	},
}

// SettingsModel is the bubbletea model for editing configuration.
type SettingsModel struct {
	config   *domain.Config
	cursor   int
	editing  bool
	input    textinput.Model
	status   string
	totalRows int // fields + directory entries
}

// NewSettingsModel creates a new SettingsModel for the given config.
func NewSettingsModel(cfg *domain.Config) SettingsModel {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60

	m := SettingsModel{
		config: cfg,
		input:  ti,
	}
	m.recalcRows()
	return m
}

func (m *SettingsModel) recalcRows() {
	m.totalRows = len(settingsFields) + len(m.config.Directories)
}

func (m SettingsModel) isFieldRow(row int) bool {
	return row < len(settingsFields)
}

func (m SettingsModel) dirIndex(row int) int {
	return row - len(settingsFields)
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNavigating(msg)
}

func (m SettingsModel) updateNavigating(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return BackToMenuMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.status = ""
			}

		case "down", "j":
			if m.cursor < m.totalRows-1 {
				m.cursor++
				m.status = ""
			}

		case "enter":
			if m.isFieldRow(m.cursor) {
				m.editing = true
				field := settingsFields[m.cursor]
				m.input.SetValue(field.get(m.config))
				m.input.Focus()
				m.input.CursorEnd()
				m.status = ""
				return m, textinput.Blink
			}
			if !m.isFieldRow(m.cursor) {
				m.status = "Press d to delete this directory"
			}

		case "d", "delete":
			if !m.isFieldRow(m.cursor) && len(m.config.Directories) > 0 {
				idx := m.dirIndex(m.cursor)
				m.config.RemoveDirectory(idx)
				m.recalcRows()
				if m.cursor >= m.totalRows && m.totalRows > 0 {
					m.cursor = m.totalRows - 1
				}
				if err := domain.SaveConfig(m.config); err != nil {
					m.status = fmt.Sprintf("Save failed: %v", err)
				} else {
					m.status = "Directory removed"
				}
			}
		}
	}
	return m, nil
}

func (m SettingsModel) updateEditing(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.status = "Value cannot be empty"
				return m, nil
			}
			field := settingsFields[m.cursor]
			field.set(m.config, value)
			m.editing = false
			m.input.Blur()
			if err := domain.SaveConfig(m.config); err != nil {
				m.status = fmt.Sprintf("Save failed: %v", err)
			} else {
				m.status = "Saved"
			}
			return m, nil

		case "esc":
			m.editing = false
			m.input.Blur()
			m.status = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SettingsModel) View() string {
	s := TitleStyle.Render("Settings") + "\n\n"

	for i, field := range settingsFields {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		if m.editing && i == m.cursor {
			s += SelectedMenuItemStyle.Render(fmt.Sprintf("%s%-20s", cursor, field.label+":"))
			s += " " + m.input.View() + "\n"
		} else {
			value := field.get(m.config)
			line := fmt.Sprintf("%s%-20s %s", cursor, field.label+":", value)
			if i == m.cursor {
				s += SelectedMenuItemStyle.Render(line) + "\n"
			} else {
				s += line + "\n"
			}
		}
	}

	if len(m.config.Directories) > 0 {
		s += "\n  Ingested directories:\n"
		for i, d := range m.config.Directories {
			row := len(settingsFields) + i
			cursor := "  "
			if row == m.cursor {
				cursor = "> "
			}
			line := fmt.Sprintf("%s  %s (%s)", cursor, d.Path, strings.Join(d.Extensions, ", "))
			if row == m.cursor {
				s += SelectedMenuItemStyle.Render(line) + "\n"
			} else {
				s += line + "\n"
			}
		}
	}

	s += "\n"
	if m.status != "" {
		s += SuccessStyle.Render("  "+m.status) + "\n"
	}

	if m.editing {
		s += HelpStyle.Render("enter: save | esc: cancel")
	} else {
		help := "up/down: navigate | enter: edit | esc: back"
		if len(m.config.Directories) > 0 {
			help += " | d: delete directory"
		}
		s += HelpStyle.Render(help)
	}

	return s
}
