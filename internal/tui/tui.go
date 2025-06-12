package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"var-sync/internal/config"
	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/pkg/models"
)

type screen int

const (
	screenMain screen = iota
	screenAddRule
	screenEditRule
	screenSelectKey
	screenBrowseFile
)

type App struct {
	config     *models.Config
	logger     *logger.Logger
	configPath string
	
	screen     screen
	list       list.Model
	inputs     []textinput.Model
	parser     *parser.Parser
	
	selectedRule *models.SyncRule
	fileKeys     []string
	keySelector  list.Model
	fileBrowser  list.Model
	currentPath  string
	
	width  int
	height int
}

type ruleItem struct {
	models.SyncRule
}

func (r ruleItem) Title() string       { return r.Name }
func (r ruleItem) Description() string { return fmt.Sprintf("%s -> %s", r.SourceKey, r.TargetKey) }
func (r ruleItem) FilterValue() string { return r.Name }

type keyItem string

func (k keyItem) Title() string       { return string(k) }
func (k keyItem) Description() string { return "" }
func (k keyItem) FilterValue() string { return string(k) }

type fileItem struct {
	name   string
	path   string
	isDir  bool
}

func (f fileItem) Title() string {
	if f.isDir {
		return f.name + "/"
	}
	return f.name
}

func (f fileItem) Description() string {
	if f.isDir {
		return "directory"
	}
	ext := filepath.Ext(f.name)
	switch ext {
	case ".json":
		return "JSON file"
	case ".yaml", ".yml":
		return "YAML file"
	case ".toml":
		return "TOML file"
	default:
		return "file"
	}
}

func (f fileItem) FilterValue() string { return f.name }

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87"))
)

func New(cfg *models.Config, logger *logger.Logger) *App {
	inputs := make([]textinput.Model, 6)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Rule name"
	inputs[0].Focus()
	inputs[0].CharLimit = 50
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Description (optional)"
	inputs[1].CharLimit = 100
	inputs[1].Width = 50

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Source file path"
	inputs[2].CharLimit = 200
	inputs[2].Width = 50

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Source key path (e.g., database.host)"
	inputs[3].CharLimit = 100
	inputs[3].Width = 40

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Target file path"
	inputs[4].CharLimit = 200
	inputs[4].Width = 50

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "Target key path (e.g., config.db.host)"
	inputs[5].CharLimit = 100
	inputs[5].Width = 40

	items := make([]list.Item, len(cfg.Rules))
	for i, rule := range cfg.Rules {
		items[i] = ruleItem{rule}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Sync Rules"

	keySelector := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	keySelector.Title = "Select Key"

	fileBrowser := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fileBrowser.Title = "Browse Files"

	currentPath, _ := os.Getwd()

	return &App{
		config:      cfg,
		logger:      logger,
		configPath:  "var-sync.json",
		screen:      screenMain,
		list:        l,
		inputs:      inputs,
		parser:      parser.New(),
		keySelector: keySelector,
		fileBrowser: fileBrowser,
		currentPath: currentPath,
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.list.SetSize(msg.Width-2, msg.Height-8)
		a.keySelector.SetSize(msg.Width-2, msg.Height-8)
		a.fileBrowser.SetSize(msg.Width-2, msg.Height-8)
		return a, nil

	case tea.KeyMsg:
		switch a.screen {
		case screenMain:
			return a.updateMain(msg)
		case screenAddRule, screenEditRule:
			return a.updateForm(msg)
		case screenSelectKey:
			return a.updateKeySelector(msg)
		case screenBrowseFile:
			return a.updateFileBrowser(msg)
		}
	}

	return a, nil
}

func (a *App) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
		return a, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		a.screen = screenAddRule
		a.clearInputs()
		a.inputs[0].Focus()
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if selected := a.list.SelectedItem(); selected != nil {
			rule := selected.(ruleItem).SyncRule
			a.removeRule(rule.ID)
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if selected := a.list.SelectedItem(); selected != nil {
			rule := selected.(ruleItem).SyncRule
			a.selectedRule = &rule
			a.screen = screenEditRule
			a.populateInputs(rule)
			a.inputs[0].Focus()
		}
		return a, nil
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a *App) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		return a, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		a.screen = screenMain
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
		if a.screen == screenAddRule {
			a.saveNewRule()
		} else {
			a.saveEditedRule()
		}
		a.screen = screenMain
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		a.nextInput()
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		a.prevInput()
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+f"))):
		focusedIdx := a.getFocusedInputIndex()
		if focusedIdx == 2 || focusedIdx == 4 {
			a.loadFileBrowser()
			a.screen = screenBrowseFile
			return a, nil
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+k"))):
		focusedIdx := a.getFocusedInputIndex()
		if focusedIdx == 3 || focusedIdx == 5 {
			filepath := ""
			if focusedIdx == 3 {
				filepath = a.inputs[2].Value()
			} else {
				filepath = a.inputs[4].Value()
			}
			if filepath != "" {
				a.loadFileKeys(filepath, focusedIdx)
				a.screen = screenSelectKey
				return a, nil
			}
		}
		return a, nil
	}

	for i := range a.inputs {
		if a.inputs[i].Focused() {
			var cmd tea.Cmd
			a.inputs[i], cmd = a.inputs[i].Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a *App) updateKeySelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		return a, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		a.screen = screenAddRule
		if a.selectedRule != nil {
			a.screen = screenEditRule
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if selected := a.keySelector.SelectedItem(); selected != nil {
			key := string(selected.(keyItem))
			focusedIdx := a.getFocusedInputIndex()
			if focusedIdx >= 0 && focusedIdx < len(a.inputs) {
				a.inputs[focusedIdx].SetValue(key)
			}
			a.screen = screenAddRule
			if a.selectedRule != nil {
				a.screen = screenEditRule
			}
		}
		return a, nil
	}

	var cmd tea.Cmd
	a.keySelector, cmd = a.keySelector.Update(msg)
	return a, cmd
}

func (a *App) updateFileBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		return a, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		a.screen = screenAddRule
		if a.selectedRule != nil {
			a.screen = screenEditRule
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if selected := a.fileBrowser.SelectedItem(); selected != nil {
			file := selected.(fileItem)
			if file.isDir {
				a.currentPath = file.path
				a.loadFileBrowser()
				return a, nil
			} else {
				// Select file
				focusedIdx := a.getFocusedInputIndex()
				if focusedIdx >= 0 && focusedIdx < len(a.inputs) {
					a.inputs[focusedIdx].SetValue(file.path)
				}
				a.screen = screenAddRule
				if a.selectedRule != nil {
					a.screen = screenEditRule
				}
			}
		}
		return a, nil
	}

	var cmd tea.Cmd
	a.fileBrowser, cmd = a.fileBrowser.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	switch a.screen {
	case screenMain:
		return a.viewMain()
	case screenAddRule:
		return a.viewForm("Add New Sync Rule")
	case screenEditRule:
		return a.viewForm("Edit Sync Rule")
	case screenSelectKey:
		return a.viewKeySelector()
	case screenBrowseFile:
		return a.viewFileBrowser()
	}
	return ""
}

func (a *App) viewMain() string {
	title := titleStyle.Render("var-sync Configuration")
	help := helpStyle.Render("a: add rule • enter: edit • d: delete • q: quit")
	
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		title,
		a.list.View(),
		help,
	)
}

func (a *App) viewForm(title string) string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	labels := []string{
		"Name:",
		"Description:",
		"Source File:",
		"Source Key:",
		"Target File:",
		"Target Key:",
	}

	for i, input := range a.inputs {
		b.WriteString(fmt.Sprintf("%s\n%s\n\n", labels[i], input.View()))
	}

	help := helpStyle.Render("ctrl+s: save • tab: next field • ctrl+f: browse file • ctrl+k: select key • esc: cancel")
	b.WriteString(help)

	return b.String()
}

func (a *App) viewKeySelector() string {
	title := titleStyle.Render("Select Key Path")
	help := helpStyle.Render("enter: select • esc: cancel")
	
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		title,
		a.keySelector.View(),
		help,
	)
}

func (a *App) viewFileBrowser() string {
	title := titleStyle.Render(fmt.Sprintf("Browse Files - %s", a.currentPath))
	help := helpStyle.Render("enter: select/navigate • esc: cancel")
	
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		title,
		a.fileBrowser.View(),
		help,
	)
}

func (a *App) saveNewRule() {
	rule := models.SyncRule{
		ID:          uuid.New().String(),
		Name:        a.inputs[0].Value(),
		Description: a.inputs[1].Value(),
		SourceFile:  a.inputs[2].Value(),
		SourceKey:   a.inputs[3].Value(),
		TargetFile:  a.inputs[4].Value(),
		TargetKey:   a.inputs[5].Value(),
		Enabled:     true,
		Created:     time.Now(),
	}

	a.config.Rules = append(a.config.Rules, rule)
	a.updateList()
	a.saveConfig()
}

func (a *App) saveEditedRule() {
	if a.selectedRule == nil {
		return
	}

	for i, rule := range a.config.Rules {
		if rule.ID == a.selectedRule.ID {
			a.config.Rules[i].Name = a.inputs[0].Value()
			a.config.Rules[i].Description = a.inputs[1].Value()
			a.config.Rules[i].SourceFile = a.inputs[2].Value()
			a.config.Rules[i].SourceKey = a.inputs[3].Value()
			a.config.Rules[i].TargetFile = a.inputs[4].Value()
			a.config.Rules[i].TargetKey = a.inputs[5].Value()
			break
		}
	}

	a.updateList()
	a.saveConfig()
	a.selectedRule = nil
}

func (a *App) removeRule(id string) {
	for i, rule := range a.config.Rules {
		if rule.ID == id {
			a.config.Rules = append(a.config.Rules[:i], a.config.Rules[i+1:]...)
			break
		}
	}
	a.updateList()
	a.saveConfig()
}

func (a *App) updateList() {
	items := make([]list.Item, len(a.config.Rules))
	for i, rule := range a.config.Rules {
		items[i] = ruleItem{rule}
	}
	a.list.SetItems(items)
}

func (a *App) saveConfig() {
	if err := config.Save(a.config, a.configPath); err != nil {
		a.logger.Error("Failed to save config: %v", err)
	}
}

func (a *App) clearInputs() {
	for i := range a.inputs {
		a.inputs[i].SetValue("")
		a.inputs[i].Blur()
	}
	a.inputs[0].Focus()
}

func (a *App) populateInputs(rule models.SyncRule) {
	a.inputs[0].SetValue(rule.Name)
	a.inputs[1].SetValue(rule.Description)
	a.inputs[2].SetValue(rule.SourceFile)
	a.inputs[3].SetValue(rule.SourceKey)
	a.inputs[4].SetValue(rule.TargetFile)
	a.inputs[5].SetValue(rule.TargetKey)
}

func (a *App) nextInput() {
	for i, input := range a.inputs {
		if input.Focused() {
			a.inputs[i].Blur()
			next := (i + 1) % len(a.inputs)
			a.inputs[next].Focus()
			break
		}
	}
}

func (a *App) prevInput() {
	for i, input := range a.inputs {
		if input.Focused() {
			a.inputs[i].Blur()
			prev := (i - 1 + len(a.inputs)) % len(a.inputs)
			a.inputs[prev].Focus()
			break
		}
	}
}

func (a *App) getFocusedInputIndex() int {
	for i, input := range a.inputs {
		if input.Focused() {
			return i
		}
	}
	return -1
}

func (a *App) loadFileKeys(filepath string, inputIdx int) {
	data, err := a.parser.LoadFile(filepath)
	if err != nil {
		return
	}

	keys := a.parser.GetAllKeys(data, "")
	items := make([]list.Item, len(keys))
	for i, key := range keys {
		items[i] = keyItem(key)
	}

	a.keySelector.SetItems(items)
}

func (a *App) loadFileBrowser() {
	entries, err := os.ReadDir(a.currentPath)
	if err != nil {
		return
	}

	var items []list.Item
	
	// Add parent directory option if not at root
	if a.currentPath != "/" && a.currentPath != "" {
		parent := filepath.Dir(a.currentPath)
		items = append(items, fileItem{
			name:  "..",
			path:  parent,
			isDir: true,
		})
	}

	for _, entry := range entries {
		fullPath := filepath.Join(a.currentPath, entry.Name())
		items = append(items, fileItem{
			name:  entry.Name(),
			path:  fullPath,
			isDir: entry.IsDir(),
		})
	}

	a.fileBrowser.SetItems(items)
}

func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}