package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	"copilot-tui/internal/composer"
	"copilot-tui/internal/copilot"
)

const (
	inputHeight    = 3
	renderThrottle = 120 * time.Millisecond
)

type model struct {
	width  int
	height int

	adapter copilot.Adapter
	events  <-chan copilot.Event

	input    composer.Model
	viewport viewport.Model

	styles styleSet
	state  ConversationState
	status string

	showPalette     bool
	paletteItems    []string
	paletteIndex    int
	currentScenario string
	useShiftEnter   bool

	lastRender      time.Time
	pendingRender   bool
	renderScheduled bool

	markdownRenderer  *glamour.TermRenderer
	markdownWrapWidth int
}

func New() tea.Model {
	return newModel(copilot.NewMockAdapter())
}

func newModel(adapter copilot.Adapter) *model {
	input := composer.New()
	input.Placeholder = "Type a prompt..."
	input.SetPromptFunc(2, func(info composer.PromptInfo) string {
		if info.LineNumber == 0 {
			return "› "
		}
		return "  "
	})
	inputStyles := input.Styles()
	inputStyles.Focused.CursorLine = lipgloss.NewStyle()
	inputStyles.Blurred.CursorLine = lipgloss.NewStyle()
	inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	inputStyles.Blurred.Prompt = inputStyles.Focused.Prompt
	inputStyles.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	inputStyles.Blurred.Placeholder = inputStyles.Focused.Placeholder
	input.SetStyles(inputStyles)
	input.Focus()
	input.CharLimit = 0
	input.ShowLineNumbers = false
	input.SetHeight(inputHeight)

	vp := viewport.New()

	m := &model{
		adapter:         adapter,
		events:          adapter.Events(),
		input:           input,
		viewport:        vp,
		styles:          newStyles(),
		state:           NewConversationState(),
		status:          "ready",
		currentScenario: "normal_markdown_stream",
		useShiftEnter:   false,
	}
	m.rebuildPalette()
	m.renderNow()
	return m
}

func (m *model) Init() tea.Cmd {
	if err := m.adapter.Start(context.Background()); err != nil {
		m.status = fmt.Sprintf("adapter start failed: %v", err)
	}
	return tea.Batch(composer.Blink, waitForAdapterEvent(m.events))
}

func (m *model) applyLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	footerHeight := 1
	separatorHeight := 1
	inputAreaHeight := inputHeight
	paletteHeight := 0
	if m.showPalette {
		paletteHeight = len(m.paletteItems) + 2
	}
	timelinePanelHeight := m.height - footerHeight - separatorHeight - inputAreaHeight - paletteHeight
	if timelinePanelHeight < 3 {
		timelinePanelHeight = 3
	}

	innerWidth := m.width
	if innerWidth < 1 {
		innerWidth = 1
	}
	m.viewport.SetWidth(innerWidth)
	m.viewport.SetHeight(timelinePanelHeight)
	m.input.SetWidth(innerWidth)
	m.input.SetHeight(inputHeight)
}

func (m *model) rebuildPalette() {
	items := make([]string, 0, 16)
	if controller, ok := m.adapter.(copilot.ScenarioController); ok {
		scenarios := controller.Scenarios()
		for _, scenario := range scenarios {
			items = append(items, "Load scenario: "+scenario)
		}
		m.currentScenario = controller.CurrentScenario()
	}

	items = append(items,
		"Set stream speed: fast",
		"Set stream speed: normal",
		"Set stream speed: slow",
	)

	m.paletteItems = items
	if m.paletteIndex >= len(m.paletteItems) {
		m.paletteIndex = 0
	}
}

func (m *model) submitPrompt() tea.Cmd {
	prompt := strings.TrimSpace(m.input.Value())
	if prompt == "" {
		m.status = "empty prompt ignored"
		return nil
	}

	m.state.AddUserPrompt(prompt)
	m.input.SetValue("")

	renderCmd := m.queueRender(true)
	if err := m.adapter.SendPrompt(context.Background(), prompt); err != nil {
		m.state.ApplyEvent(copilot.Event{
			Type: copilot.EventError,
			Text: fmt.Sprintf("send prompt failed: %v", err),
		})
		m.status = "failed to send prompt"
		return tea.Batch(renderCmd, m.queueRender(true))
	}

	m.status = "streaming response..."
	return renderCmd
}

func (m *model) handleAdapterEvent(ev copilot.Event) tea.Cmd {
	m.state.ApplyEvent(ev)

	switch ev.Type {
	case copilot.EventAssistantDelta, copilot.EventReasoningDelta:
		return m.queueRender(false)
	case copilot.EventToolStart, copilot.EventToolProgress, copilot.EventToolComplete:
		m.status = fmt.Sprintf("tool %s: %s", ev.Tool, ev.Text)
		return m.queueRender(true)
	case copilot.EventError:
		m.status = ev.Text
		return m.queueRender(true)
	case copilot.EventTurnComplete:
		m.status = "turn complete"
		return m.queueRender(true)
	default:
		return nil
	}
}

func (m *model) queueRender(force bool) tea.Cmd {
	if force {
		m.pendingRender = false
		m.renderScheduled = false
		m.renderNow()
		return nil
	}

	m.pendingRender = true
	elapsed := time.Since(m.lastRender)
	if elapsed >= renderThrottle {
		m.pendingRender = false
		m.renderNow()
		return nil
	}
	if m.renderScheduled {
		return nil
	}

	m.renderScheduled = true
	return renderTickCmd(renderThrottle - elapsed)
}

func (m *model) ensureMarkdownRenderer() {
	wrap := m.viewport.Width()
	if wrap < 20 {
		wrap = 20
	}
	if m.markdownRenderer != nil && m.markdownWrapWidth == wrap {
		return
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		m.markdownRenderer = nil
		m.markdownWrapWidth = wrap
		return
	}
	m.markdownRenderer = renderer
	m.markdownWrapWidth = wrap
}

func (m *model) renderMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	m.ensureMarkdownRenderer()
	if m.markdownRenderer == nil {
		return text
	}

	out, err := m.markdownRenderer.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimSuffix(out, "\n")
}

func (m *model) timelineContent() string {
	if len(m.state.Items) == 0 {
		return "No transcript yet.\nType below and press Enter."
	}

	var b strings.Builder
	for i, item := range m.state.Items {
		if i > 0 {
			b.WriteString("\n\n")
		}

		switch item.Kind {
		case TimelineUser:
			b.WriteString(m.styles.UserPrefix.Render("› "))
			b.WriteString(item.Text)
		case TimelineAssistant:
			b.WriteString(m.renderMarkdown(item.Text))
		case TimelineReasoning:
			b.WriteString(m.styles.ReasoningPrefix.Render("• reasoning"))
			b.WriteString("\n")
			b.WriteString(m.renderMarkdown(item.Text))
		case TimelineTool:
			prefix := "• tool"
			if item.Tool != "" {
				prefix = "• " + item.Tool
			}
			b.WriteString(m.styles.ToolPrefix.Render(prefix + " "))
			b.WriteString(item.Text)
		case TimelineError:
			b.WriteString(m.styles.ErrorPrefix.Render("x "))
			b.WriteString(item.Text)
		}
	}

	return b.String()
}

func (m *model) renderNow() {
	m.viewport.SetContent(m.timelineContent())
	m.viewport.GotoBottom()
	m.lastRender = time.Now()
}

func (m *model) updatePaletteKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.showPalette = false
		return nil
	case "up", "k":
		if len(m.paletteItems) == 0 {
			return nil
		}
		if m.paletteIndex > 0 {
			m.paletteIndex--
		}
		return nil
	case "down", "j":
		if len(m.paletteItems) == 0 {
			return nil
		}
		if m.paletteIndex < len(m.paletteItems)-1 {
			m.paletteIndex++
		}
		return nil
	case "enter":
		if len(m.paletteItems) == 0 {
			m.showPalette = false
			return nil
		}
		m.applyPaletteSelection(m.paletteItems[m.paletteIndex])
		m.showPalette = false
		return nil
	default:
		return nil
	}
}

func (m *model) applyPaletteSelection(selection string) {
	switch {
	case strings.HasPrefix(selection, "Load scenario: "):
		name := strings.TrimPrefix(selection, "Load scenario: ")
		controller, ok := m.adapter.(copilot.ScenarioController)
		if !ok {
			m.status = "adapter does not support scenarios"
			return
		}
		if err := controller.SetScenario(name); err != nil {
			m.status = fmt.Sprintf("failed to load scenario: %v", err)
			return
		}
		m.currentScenario = controller.CurrentScenario()
		m.status = "loaded scenario: " + m.currentScenario

	case strings.HasPrefix(selection, "Set stream speed: "):
		controller, ok := m.adapter.(copilot.ScenarioController)
		if !ok {
			m.status = "adapter does not support speed changes"
			return
		}
		speed := strings.TrimPrefix(selection, "Set stream speed: ")
		delay := 40 * time.Millisecond
		switch speed {
		case "fast":
			delay = 12 * time.Millisecond
		case "slow":
			delay = 120 * time.Millisecond
		}
		controller.SetStreamDelay(delay)
		m.status = "stream speed set to " + speed
	}
}

func (m *model) newlineHint() string {
	if m.useShiftEnter {
		return "Shift+Enter"
	}
	return "Ctrl+J"
}
