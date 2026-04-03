package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// MenuItem represents a single entry in the main menu.
type MenuItem struct {
	Name        string
	Description string
}

var menuItems = []MenuItem{
	{Name: "Ingest", Description: "Run the full ingestion pipeline"},
	{Name: "Update Only", Description: "Re-ingest only changed files"},
	{Name: "Download (rclone)", Description: "Download files from Google Drive"},
	{Name: "Stats", Description: "Show collection statistics"},
	{Name: "Reset Collection", Description: "Delete and recreate collection"},
	{Name: "Start ChromaDB", Description: "Start the ChromaDB container"},
	{Name: "Stop ChromaDB", Description: "Stop the ChromaDB container"},
	{Name: "Settings", Description: "View current configuration"},
	{Name: "Quit", Description: "Exit the application"},
}

// MenuModel is the bubbletea model for the main menu.
type MenuModel struct {
	items  []MenuItem
	cursor int
	chosen int
}

// NewMenuModel creates a MenuModel with the default menu items.
func NewMenuModel() MenuModel {
	return MenuModel{
		items:  menuItems,
		cursor: 0,
		chosen: -1,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.cursor
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := TitleStyle.Render("Ingester") + "\n\n"

	for i, item := range m.items {
		if i == m.cursor {
			s += SelectedMenuItemStyle.Render(fmt.Sprintf("> %s", item.Name))
		} else {
			s += MenuItemStyle.Render(fmt.Sprintf("  %s", item.Name))
		}
		s += "  " + HelpStyle.Render(item.Description) + "\n"
	}

	s += "\n" + HelpStyle.Render("↑/↓: navigate • enter: select • q: quit")
	return s
}
