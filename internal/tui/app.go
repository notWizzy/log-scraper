package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/notWizzy/log-scraper/internal/model"
)

type viewMode int

const (
	viewSummary viewMode = iota
	viewList
	viewDetail
)

type App struct {
	result   *model.Result
	mode     viewMode
	cursor   int
	expanded map[int]bool
	scroll   int
	width    int
	height   int
}

func NewApp(result *model.Result) App {
	return App{
		result:   result,
		mode:     viewSummary,
		expanded: make(map[int]bool),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		action := parseKey(msg)
		switch a.mode {
		case viewSummary:
			return a.updateSummary(action)
		case viewList:
			return a.updateList(action)
		case viewDetail:
			return a.updateDetail(action)
		}
	}
	return a, nil
}

func (a App) View() string {
	switch a.mode {
	case viewSummary:
		return a.viewSummary()
	case viewList:
		return a.viewList()
	case viewDetail:
		return a.viewDetail()
	default:
		return ""
	}
}

// --- Summary View ---

func (a App) updateSummary(action keyAction) (tea.Model, tea.Cmd) {
	switch action {
	case keyQuit:
		return a, tea.Quit
	case keyEnter:
		if len(a.result.Chains) > 0 {
			a.mode = viewList
			a.cursor = 0
			a.scroll = 0
		}
	}
	return a, nil
}

func (a App) viewSummary() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" LOG SCRAPER "))
	b.WriteString("\n\n")

	// Stats
	b.WriteString(boldStyle.Render("Scan Complete"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Lines scanned:   %s\n", formatNum(a.result.TotalLines)))
	b.WriteString(fmt.Sprintf("  Matches found:   %s\n", formatNum(int64(a.result.TotalMatches))))
	b.WriteString(fmt.Sprintf("  Failure chains:  %d\n", len(a.result.Chains)))
	b.WriteString(fmt.Sprintf("  Duration:        %s\n", a.result.Duration.Round(1_000_000)))
	b.WriteString(fmt.Sprintf("  Sources:         %s\n", strings.Join(a.result.Sources, ", ")))
	b.WriteString("\n")

	// Category breakdown
	if len(a.result.ByCategory) > 0 {
		b.WriteString(boldStyle.Render("By Category"))
		b.WriteString("\n")
		categories := []model.Category{
			model.CategoryOOMKill, model.CategoryTimeout, model.CategoryDBLock,
			model.CategoryPanic, model.CategorySignal, model.CategoryDiskIO,
			model.CategoryNetwork,
		}
		for _, cat := range categories {
			count, ok := a.result.ByCategory[cat]
			if !ok || count == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  %s %d\n", categoryBadge(cat), count))
		}
		b.WriteString("\n")
	}

	// Severity breakdown
	if len(a.result.BySeverity) > 0 {
		b.WriteString(boldStyle.Render("By Severity"))
		b.WriteString("\n")
		severities := []model.Severity{
			model.SeverityFatal, model.SeverityCritical, model.SeverityError, model.SeverityWarn,
		}
		for _, sev := range severities {
			count, ok := a.result.BySeverity[sev]
			if !ok || count == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  %s %d\n",
				severityStyle(sev).Render(sev.String()), count))
		}
		b.WriteString("\n")
	}

	if len(a.result.Chains) > 0 {
		b.WriteString(helpStyle.Render("enter: view failures • q: quit"))
	} else {
		b.WriteString(helpStyle.Render("No failures detected • q: quit"))
	}
	b.WriteString("\n")

	return b.String()
}

// --- List View ---

func (a App) updateList(action keyAction) (tea.Model, tea.Cmd) {
	switch action {
	case keyQuit:
		return a, tea.Quit
	case keyUp:
		if a.cursor > 0 {
			a.cursor--
		}
	case keyDown:
		if a.cursor < len(a.result.Chains)-1 {
			a.cursor++
		}
	case keyEnter:
		a.mode = viewDetail
		a.scroll = 0
	case keyBack, keyTab:
		a.mode = viewSummary
	case keyExpand:
		a.expanded[a.cursor] = !a.expanded[a.cursor]
	}
	return a, nil
}

func (a App) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" FAILURE CHAINS "))
	b.WriteString(fmt.Sprintf("  %d chains found\n\n",
		len(a.result.Chains)))

	maxVisible := a.height - 6
	if maxVisible < 5 {
		maxVisible = 20
	}

	startIdx := 0
	if a.cursor >= maxVisible {
		startIdx = a.cursor - maxVisible + 1
	}

	for i := startIdx; i < len(a.result.Chains) && i < startIdx+maxVisible; i++ {
		chain := a.result.Chains[i]
		prefix := "  "
		if i == a.cursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s#%d %s %s [%d errors] %s",
			prefix,
			chain.ID,
			categoryBadge(chain.Category),
			severityStyle(chain.Severity).Render(chain.Severity.String()),
			len(chain.Cascade)+1,
			truncate(string(chain.RootCause.Entry.Line), 60),
		)

		if i == a.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		if a.expanded[i] {
			b.WriteString(dimStyle.Render(fmt.Sprintf("    Root: %s:%d %s",
				chain.RootCause.Entry.Source,
				chain.RootCause.Entry.LineNum,
				chain.RootCause.Pattern.Name)))
			b.WriteString("\n")
			for j, m := range chain.Cascade {
				if j >= 5 {
					b.WriteString(dimStyle.Render(fmt.Sprintf("    ... and %d more",
						len(chain.Cascade)-5)))
					b.WriteString("\n")
					break
				}
				b.WriteString(dimStyle.Render(fmt.Sprintf("    └─ %s:%d %s",
					m.Entry.Source, m.Entry.LineNum, m.Pattern.Name)))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate • space: expand • enter: detail • tab: summary • q: quit"))
	b.WriteString("\n")

	return b.String()
}

// --- Detail View ---

func (a App) updateDetail(action keyAction) (tea.Model, tea.Cmd) {
	switch action {
	case keyQuit:
		return a, tea.Quit
	case keyBack:
		a.mode = viewList
	case keyUp:
		if a.scroll > 0 {
			a.scroll--
		}
	case keyDown:
		a.scroll++
	}
	return a, nil
}

func (a App) viewDetail() string {
	if a.cursor >= len(a.result.Chains) {
		return ""
	}
	chain := a.result.Chains[a.cursor]

	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" CHAIN #%d ", chain.ID)))
	b.WriteString("\n\n")

	// Root cause
	b.WriteString(severityStyle(chain.RootCause.Pattern.Severity).Render("ROOT CAUSE"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Category:  %s\n", categoryBadge(chain.RootCause.Pattern.Category)))
	b.WriteString(fmt.Sprintf("  Pattern:   %s\n", chain.RootCause.Pattern.Name))
	b.WriteString(fmt.Sprintf("  Source:    %s:%d\n", chain.RootCause.Entry.Source, chain.RootCause.Entry.LineNum))
	if !chain.RootCause.Timestamp.IsZero() {
		b.WriteString(fmt.Sprintf("  Time:      %s\n", chain.RootCause.Timestamp.Format("2006-01-02 15:04:05.000")))
	}
	b.WriteString("\n")
	b.WriteString(renderLogLine(chain.RootCause))
	b.WriteString("\n\n")

	// Cascade
	if len(chain.Cascade) > 0 {
		b.WriteString(boldStyle.Render(fmt.Sprintf("CASCADE (%d related errors)", len(chain.Cascade))))
		b.WriteString("\n\n")

		visibleStart := a.scroll
		if visibleStart > len(chain.Cascade) {
			visibleStart = len(chain.Cascade)
		}

		maxShow := a.height - 20
		if maxShow < 5 {
			maxShow = 10
		}

		for i := visibleStart; i < len(chain.Cascade) && i < visibleStart+maxShow; i++ {
			m := chain.Cascade[i]
			b.WriteString(fmt.Sprintf("  %s %s %s:%d\n",
				categoryBadge(m.Pattern.Category),
				dimStyle.Render(m.Pattern.Name),
				m.Entry.Source,
				m.Entry.LineNum,
			))
			b.WriteString(fmt.Sprintf("  %s\n\n",
				dimStyle.Render(truncate(string(m.Entry.Line), 100))))
		}

		if len(chain.Cascade) > visibleStart+maxShow {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more (scroll with j/k)",
				len(chain.Cascade)-visibleStart-maxShow)))
			b.WriteString("\n")
		}
	}

	if chain.TimeSpan > 0 {
		b.WriteString(fmt.Sprintf("\n  Time span: %s\n", chain.TimeSpan))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: scroll • esc/h: back to list • q: quit"))
	b.WriteString("\n")

	return b.String()
}

// --- Helpers ---

func renderLogLine(m model.Match) string {
	lineStr := string(m.Entry.Line)
	maxWidth := 120
	if len(lineStr) > maxWidth {
		lineStr = lineStr[:maxWidth] + "..."
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(categoryColors[m.Pattern.Category]).
		Padding(0, 1).
		Render(lineStr)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatNum(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1_000_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
}
