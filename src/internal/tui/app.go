package tui

import (
	"bytes"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yetanotherchris/zolam/internal/docker"
	"github.com/yetanotherchris/zolam/internal/domain"
	"github.com/yetanotherchris/zolam/internal/zolam"
)

type viewState int

const (
	menuView viewState = iota
	ingestView
	progressView
	settingsView
)

// AppModel is the root bubbletea model that switches between views.
type AppModel struct {
	state        viewState
	menu         MenuModel
	ingest       IngestModel
	progress     ProgressModel
	config       *domain.Config
	dockerClient *docker.DockerClient
	ingester     *zolam.Ingester
	warnings     []string
	sender       *ProgramSender
}

// ProgramSender holds a reference to the tea.Program for sending messages
// from background goroutines. It's a pointer so it survives bubbletea's
// model copying.
type ProgramSender struct {
	Program *tea.Program
}

// Sender returns the ProgramSender so callers can set the program reference.
func (m AppModel) Sender() *ProgramSender {
	return m.sender
}

// NewApp creates a new AppModel with the given dependencies.
func NewApp(cfg *domain.Config, dc *docker.DockerClient, ing *zolam.Ingester, warnings []string) AppModel {
	return AppModel{
		state:        menuView,
		menu:         NewMenuModel(),
		config:       cfg,
		dockerClient: dc,
		ingester:     ing,
		warnings:     warnings,
		sender:       &ProgramSender{},
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.menu.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case BackToMenuMsg:
		m.state = menuView
		m.menu.chosen = -1
		return m, nil

	case StartIngestMsg:
		m.progress = NewProgressModel("Ingest")
		m.state = progressView
		return m, m.runIngest(msg.Directories, msg.Extensions)
	}

	switch m.state {
	case menuView:
		return m.updateMenu(msg)
	case ingestView:
		return m.updateIngest(msg)
	case progressView:
		return m.updateProgress(msg)
	case settingsView:
		return m.updateSettings(msg)
	}

	return m, nil
}

func (m AppModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)

	if m.menu.chosen < 0 {
		return m, cmd
	}

	chosen := m.menu.chosen
	m.menu.chosen = -1

	switch chosen {
	case 0: // Ingest
		m.ingest = NewIngestModel()
		m.state = ingestView
		return m, m.ingest.Init()

	case 1: // Update Only
		m.progress = NewProgressModel("Update Only")
		m.state = progressView
		return m, m.runUpdateOnly()

	case 2: // Download (rclone)
		m.progress = NewProgressModel("Download (rclone)")
		m.state = progressView
		return m, m.runRclone()

	case 3: // Stats
		m.progress = NewProgressModel("Stats")
		m.state = progressView
		return m, m.runStats()

	case 4: // Reset Collection
		m.progress = NewProgressModel("Reset Collection")
		m.state = progressView
		return m, m.runReset()

	case 5: // Start ChromaDB
		m.progress = NewProgressModel("Start ChromaDB")
		m.state = progressView
		return m, m.runStartChromaDB()

	case 6: // Stop ChromaDB
		m.progress = NewProgressModel("Stop ChromaDB")
		m.state = progressView
		return m, m.runStopChromaDB()

	case 7: // Settings
		m.state = settingsView
		return m, nil

	case 8: // Quit
		return m, tea.Quit
	}

	return m, cmd
}

func (m AppModel) updateIngest(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ingest, cmd = m.ingest.Update(msg)
	return m, cmd
}

func (m AppModel) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
	return m, cmd
}

func (m AppModel) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "enter", "q":
			m.state = menuView
			m.menu.chosen = -1
			return m, nil
		}
	}
	return m, nil
}

func (m AppModel) View() string {
	switch m.state {
	case menuView:
		v := DocStyle.Render(m.menu.View())
		if len(m.warnings) > 0 {
			v += "\n"
			for _, w := range m.warnings {
				v += WarningStyle.Render("⚠ "+w) + "\n"
			}
		}
		return v
	case ingestView:
		return DocStyle.Render(m.ingest.View())
	case progressView:
		return DocStyle.Render(m.progress.View())
	case settingsView:
		return DocStyle.Render(m.settingsView())
	}
	return ""
}

func (m AppModel) settingsView() string {
	s := TitleStyle.Render("Settings") + "\n\n"

	s += fmt.Sprintf("  Collection Name:     %s\n", m.config.CollectionName)
	s += fmt.Sprintf("  OpenRouter Model:    %s\n", m.config.OpenRouterModel)
	s += fmt.Sprintf("  Use Local Embeddings:%v\n", m.config.UseLocalEmbeddings)
	s += fmt.Sprintf("  Rclone Remote:       %s\n", m.config.RcloneRemote)
	s += fmt.Sprintf("  Rclone Source:       %s\n", m.config.RcloneSource)
	s += fmt.Sprintf("  Data Dir:            %s\n", m.config.DataDir)
	s += fmt.Sprintf("  Extensions:          %s\n", strings.Join(m.config.Extensions, ", "))

	if m.config.OpenRouterAPIKey != "" {
		s += fmt.Sprintf("  API Key:             %s\n", "****"+m.config.OpenRouterAPIKey[max(0, len(m.config.OpenRouterAPIKey)-4):])
	} else {
		s += "  API Key:             (not set)\n"
	}

	s += "\n" + HelpStyle.Render("esc/enter/q: back to menu")
	return s
}

// --- Commands that run operations in goroutines ---

func (m AppModel) runIngest(directories []string, extensions string) tea.Cmd {
	return func() tea.Msg {
		if err := m.checkChromaDB(); err != nil {
			return OperationDoneMsg{Err: err}
		}

		exts := strings.Split(extensions, ",")
		for i := range exts {
			exts[i] = strings.TrimSpace(exts[i])
		}

		opts := zolam.IngestOptions{
			Extensions:     exts,
			CollectionName: m.config.CollectionName,
		}

		p := m.sender.Program
		err := m.ingester.Run(directories, opts, func(line string) {
			if p != nil {
				p.Send(OutputLineMsg{Line: line})
			}
		})
		if err != nil {
			return OperationDoneMsg{Err: err}
		}
		return OperationDoneMsg{}
	}
}

func (m AppModel) runUpdateOnly() tea.Cmd {
	return func() tea.Msg {
		if err := m.checkChromaDB(); err != nil {
			return OperationDoneMsg{Err: err}
		}

		dirs := m.config.Directories
		if len(dirs) == 0 {
			return OperationDoneMsg{Err: fmt.Errorf("no directories configured")}
		}

		p := m.sender.Program
		result, err := m.ingester.RunUpdateOnly(dirs, func(line string) {
			if p != nil {
				p.Send(OutputLineMsg{Line: line})
			}
		})
		if err != nil {
			return OperationDoneMsg{Err: err}
		}

		return OperationDoneMsg{Output: fmt.Sprintf("Update complete: %d added, %d changed, %d removed, %d unchanged",
			result.Added, result.Changed, result.Removed, result.Unchanged)}
	}
}

func (m AppModel) runRclone() tea.Cmd {
	return func() tea.Msg {
		cmd, err := m.dockerClient.RcloneSync(m.config.RcloneRemote, m.config.RcloneSource, m.config.DataDir, m.config.RcloneConfigDir)
		if err != nil {
			return OperationDoneMsg{Err: err}
		}

		var buf bytes.Buffer
		err = m.dockerClient.StreamOutput(cmd, &buf)
		if err != nil {
			return OperationDoneMsg{Err: fmt.Errorf("%w: %s", err, buf.String())}
		}
		return OperationDoneMsg{}
	}
}

func (m AppModel) checkChromaDB() error {
	running, _ := m.dockerClient.ChromaDBStatus()
	if !running {
		return fmt.Errorf("ChromaDB is not running. Use 'Start ChromaDB' from the menu first")
	}
	return nil
}

func (m AppModel) runStats() tea.Cmd {
	return func() tea.Msg {
		if err := m.checkChromaDB(); err != nil {
			return OperationDoneMsg{Err: err}
		}

		p := m.sender.Program
		stats, err := m.ingester.GetStats(func(line string) {
			if p != nil {
				p.Send(OutputLineMsg{Line: line})
			}
		})
		if err != nil {
			return OperationDoneMsg{Err: err}
		}

		summary := fmt.Sprintf("Collection: %s\nChromaDB:   running\nEmbeddings: %s",
			stats.CollectionName, stats.EmbeddingType)
		return OperationDoneMsg{Output: summary}
	}
}

func (m AppModel) runReset() tea.Cmd {
	return func() tea.Msg {
		if err := m.checkChromaDB(); err != nil {
			return OperationDoneMsg{Err: err}
		}

		opts := zolam.IngestOptions{
			CollectionName: m.config.CollectionName,
			Reset:          true,
		}

		p := m.sender.Program
		err := m.ingester.Run(nil, opts, func(line string) {
			if p != nil {
				p.Send(OutputLineMsg{Line: line})
			}
		})
		if err != nil {
			return OperationDoneMsg{Err: err}
		}
		return OperationDoneMsg{}
	}
}

func (m AppModel) runStartChromaDB() tea.Cmd {
	return func() tea.Msg {
		err := m.dockerClient.StartChromaDB()
		if err != nil {
			return OperationDoneMsg{Err: err}
		}
		return OperationDoneMsg{}
	}
}

func (m AppModel) runStopChromaDB() tea.Cmd {
	return func() tea.Msg {
		err := m.dockerClient.StopChromaDB()
		if err != nil {
			return OperationDoneMsg{Err: err}
		}
		return OperationDoneMsg{}
	}
}
