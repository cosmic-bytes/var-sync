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
	
	// UI state
	message     string
	messageType string // "success", "error", "info"
	showHelp    bool
}

type ruleItem struct {
	models.SyncRule
}

func (r ruleItem) Title() string { 
	status := "🟢"
	if !r.Enabled {
		status = "🔴"
	}
	return fmt.Sprintf("%s %s", status, r.Name)
}

func (r ruleItem) Description() string { 
	desc := fmt.Sprintf("%s -> %s", r.SourceKey, r.TargetKey)
	if r.SyncRule.Description != "" {
		desc = fmt.Sprintf("%s | %s", r.SyncRule.Description, desc)
	}
	return desc
}

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
		return "📁 " + f.name
	}
	switch filepath.Ext(f.name) {
	case ".json":
		return "📝 " + f.name
	case ".yaml", ".yml":
		return "📜 " + f.name
	case ".toml":
		return "📄 " + f.name
	default:
		return "📄 " + f.name
	}
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
		Padding(1, 1).
		Margin(0, 0)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Italic(true)

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true)

	// Enhanced styles - optimized for full screen
	boxStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0)

	formBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#626262")).
		Padding(0, 1).
		Margin(0, 0)

	focusedInputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	blurredInputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#626262")).
		Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Bold(true).
		MarginBottom(1)

	enabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	disabledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true)

	metadataStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Italic(true)

	breadcrumbStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	// Additional engaging styles
	separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true)

	accentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)
)

func New(cfg *models.Config, logger *logger.Logger) *App {
	// Standard input width for consistency
	standardWidth := 60
	
	inputs := make([]textinput.Model, 6)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Rule name"
	inputs[0].Focus()
	inputs[0].CharLimit = 50
	inputs[0].Width = standardWidth

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Description (optional)"
	inputs[1].CharLimit = 100
	inputs[1].Width = standardWidth

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Source file path"
	inputs[2].CharLimit = 200
	inputs[2].Width = standardWidth

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Source key path (e.g., database.host)"
	inputs[3].CharLimit = 100
	inputs[3].Width = standardWidth

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Target file path"
	inputs[4].CharLimit = 200
	inputs[4].Width = standardWidth

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "Target key path (e.g., config.db.host)"
	inputs[5].CharLimit = 100
	inputs[5].Width = standardWidth

	items := make([]list.Item, len(cfg.Rules))
	for i, rule := range cfg.Rules {
		items[i] = ruleItem{rule}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Sync Rules"
	// Ensure filtering is enabled
	l.SetShowHelp(false) // We provide our own help
	l.SetFilteringEnabled(true)

	keySelector := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	keySelector.Title = "Select Key"
	keySelector.SetShowHelp(false)
	keySelector.SetFilteringEnabled(true)

	fileBrowser := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fileBrowser.Title = "Browse Files"
	fileBrowser.SetShowHelp(false)
	fileBrowser.SetFilteringEnabled(true)

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
		// Use most of the screen for lists, leaving space for title and help
		a.list.SetSize(msg.Width, msg.Height-6)
		a.keySelector.SetSize(msg.Width, msg.Height-6)
		a.fileBrowser.SetSize(msg.Width, msg.Height-6)
		
		// Update input widths based on window size
		inputWidth := msg.Width - 10 // Leave some margin
		if inputWidth > 80 {
			inputWidth = 80 // Cap at reasonable maximum
		}
		if inputWidth < 30 {
			inputWidth = 30 // Ensure minimum usability
		}
		
		for i := range a.inputs {
			a.inputs[i].Width = inputWidth
		}
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
	case key.Matches(msg, key.NewBinding(key.WithKeys("?", "h"))):
		a.showHelp = !a.showHelp
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		a.screen = screenAddRule
		a.clearInputs()
		a.inputs[0].Focus()
		a.clearMessage()
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if selected := a.list.SelectedItem(); selected != nil {
			rule := selected.(ruleItem).SyncRule
			a.removeRule(rule.ID)
			a.setMessage(fmt.Sprintf("Deleted rule: %s", rule.Name), "success")
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("t"))):
		if selected := a.list.SelectedItem(); selected != nil {
			rule := selected.(ruleItem).SyncRule
			a.toggleRule(rule.ID)
			status := "enabled"
			if !rule.Enabled {
				status = "disabled"
			}
			a.setMessage(fmt.Sprintf("Rule %s %s", rule.Name, status), "info")
		}
		return a, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if selected := a.list.SelectedItem(); selected != nil {
			rule := selected.(ruleItem).SyncRule
			a.selectedRule = &rule
			a.screen = screenEditRule
			a.populateInputs(rule)
			a.inputs[0].Focus()
			a.clearMessage()
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
		// If filtering is active, let the list handle esc to clear filter
		if a.keySelector.FilterState() == list.Filtering {
			var cmd tea.Cmd
			a.keySelector, cmd = a.keySelector.Update(msg)
			return a, cmd
		}
		// Otherwise, go back to form
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
		// If filtering is active, let the list handle esc to clear filter
		if a.fileBrowser.FilterState() == list.Filtering {
			var cmd tea.Cmd
			a.fileBrowser, cmd = a.fileBrowser.Update(msg)
			return a, cmd
		}
		// Otherwise, go back to form
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
	// Elegant title with separator
	titleText := fmt.Sprintf("🚀 Var-Sync Configuration — %d Rules", len(a.config.Rules))
	title := titleStyle.Width(a.width).Align(lipgloss.Center).Render(titleText)
	separator := separatorStyle.Width(a.width).Render(strings.Repeat("─", a.width))
	
	// Build help text
	var helpText string
	if a.showHelp {
		helpText = helpStyle.Render(
			"Navigation: ↑/↓ to select • enter: edit • a: add • d: delete • t: toggle enable/disable\n" +
			"Filter: /: search/filter list • esc: clear filter\n" +
			"Help: h/?: toggle this help • q/ctrl+c: quit\n" +
			"Shortcuts: ctrl+f: file browser • ctrl+k: key selector")
	} else {
		helpText = helpStyle.Render("Press h or ? for help • a: add • enter: edit • /: filter • d: delete • t: toggle • q: quit")
	}
	
	// Status bar with message
	var statusBar string
	if a.message != "" {
		switch a.messageType {
		case "success":
			statusBar = statusStyle.Width(a.width).Render("✓ " + a.message)
		case "error":
			statusBar = errorStyle.Width(a.width).Render("✗ " + a.message)
		case "info":
			statusBar = helpStyle.Width(a.width).Render("ℹ " + a.message)
		}
		statusBar += "\n"
	}
	
	// Full-width help bar
	helpBar := helpStyle.Width(a.width).Align(lipgloss.Center).Render(helpText)
	
	return fmt.Sprintf("%s\n%s\n%s%s\n%s",
		title,
		separator,
		a.list.View(),
		statusBar,
		helpBar,
	)
}

func (a *App) viewForm(title string) string {
	// Elegant title with separator
	titleText := titleStyle.Width(a.width).Align(lipgloss.Center).Render("✏️ " + title)
	separator := separatorStyle.Width(a.width).Render(strings.Repeat("─", a.width))
	
	labels := []string{
		"Name:",
		"Description:",
		"Source File:",
		"Source Key:",
		"Target File:",
		"Target Key:",
	}

	icons := []string{
		"🏷️",
		"📝",
		"📁",
		"🔑",
		"📂",
		"🎯",
	}
	
	// Center the form on screen
	formWidth := a.width - 4
	if formWidth > 100 {
		formWidth = 100 // Max form width for readability
	}
	
	var formContent strings.Builder
	for i, input := range a.inputs {
		label := labelStyle.Render(fmt.Sprintf("%s %s", icons[i], labels[i]))
		var inputView string
		if input.Focused() {
			inputView = focusedInputStyle.Width(formWidth).Render(input.View())
		} else {
			inputView = blurredInputStyle.Width(formWidth).Render(input.View())
		}
		
		formContent.WriteString(fmt.Sprintf("%s\n%s\n\n", label, inputView))
	}
	
	// Center the form content
	centeredForm := lipgloss.NewStyle().
		Width(a.width).
		Align(lipgloss.Center).
		Render(formContent.String())
	
	// Status bar for errors
	var statusBar string
	if a.message != "" && a.messageType == "error" {
		statusBar = errorStyle.Width(a.width).Align(lipgloss.Center).Render("✗ " + a.message) + "\n"
	}
	
	// Full-width help bar
	helpBar := helpStyle.Width(a.width).Align(lipgloss.Center).Render(
		"Navigation: tab/shift+tab: next/prev field • ctrl+s: save • esc: cancel\n" +
		"Helpers: ctrl+f: file browser • ctrl+k: key selector")
	
	return fmt.Sprintf("%s\n%s\n\n%s%s%s",
		titleText,
		separator,
		centeredForm,
		statusBar,
		helpBar,
	)
}

func (a *App) viewKeySelector() string {
	title := titleStyle.Width(a.width).Align(lipgloss.Center).Render("🔑 Select Key Path")
	separator := separatorStyle.Width(a.width).Render(strings.Repeat("─", a.width))
	helpBar := helpStyle.Width(a.width).Align(lipgloss.Center).Render("Navigation: ↑/↓ to select • /: filter • enter: choose key • esc: cancel")
	
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		title,
		separator,
		a.keySelector.View(),
		helpBar,
	)
}

func (a *App) viewFileBrowser() string {
	title := titleStyle.Width(a.width).Align(lipgloss.Center).Render("📁 File Browser")
	separator := separatorStyle.Width(a.width).Render(strings.Repeat("─", a.width))
	breadcrumb := breadcrumbStyle.Width(a.width).Align(lipgloss.Left).Render(fmt.Sprintf("📂 %s", a.currentPath))
	helpBar := helpStyle.Width(a.width).Align(lipgloss.Center).Render("Navigation: ↑/↓ to select • /: filter • enter: choose/open • esc: cancel")
	
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		title,
		separator,
		breadcrumb,
		a.fileBrowser.View(),
		helpBar,
	)
}

func (a *App) saveNewRule() {
	if err := a.validateForm(); err != nil {
		a.setMessage(err.Error(), "error")
		return
	}
	
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
	a.setMessage(fmt.Sprintf("Created rule: %s", rule.Name), "success")
}

func (a *App) saveEditedRule() {
	if a.selectedRule == nil {
		return
	}
	
	if err := a.validateForm(); err != nil {
		a.setMessage(err.Error(), "error")
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
	a.setMessage(fmt.Sprintf("Updated rule: %s", a.inputs[0].Value()), "success")
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

func (a *App) toggleRule(id string) {
	for i, rule := range a.config.Rules {
		if rule.ID == id {
			a.config.Rules[i].Enabled = !a.config.Rules[i].Enabled
			break
		}
	}
	a.updateList()
	a.saveConfig()
}

func (a *App) setMessage(msg, msgType string) {
	a.message = msg
	a.messageType = msgType
}

func (a *App) clearMessage() {
	a.message = ""
	a.messageType = ""
}

func (a *App) validateForm() error {
	if strings.TrimSpace(a.inputs[0].Value()) == "" {
		return fmt.Errorf("Name is required")
	}
	if strings.TrimSpace(a.inputs[2].Value()) == "" {
		return fmt.Errorf("Source file is required")
	}
	if strings.TrimSpace(a.inputs[3].Value()) == "" {
		return fmt.Errorf("Source key is required")
	}
	if strings.TrimSpace(a.inputs[4].Value()) == "" {
		return fmt.Errorf("Target file is required")
	}
	if strings.TrimSpace(a.inputs[5].Value()) == "" {
		return fmt.Errorf("Target key is required")
	}
	return nil
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