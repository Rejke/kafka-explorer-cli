package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kafka-explorer-cli/internal/core/domain"
	"github.com/kafka-explorer-cli/internal/core/ports"
)

// Color scheme
var (
	primaryColor     = lipgloss.Color("#FF5F87") // Pink
	secondaryColor   = lipgloss.Color("#AF87FF") // Purple
	accentColor      = lipgloss.Color("#5FD7FF") // Cyan
	textColor        = lipgloss.Color("#FFFFFF") // White
	subtextColor     = lipgloss.Color("#AAAAAA") // Light gray
	successColor     = lipgloss.Color("#5FFF87") // Green
	highlightColor   = lipgloss.Color("#FFAF5F") // Orange
	borderColor      = lipgloss.Color("#5F87FF") // Blue
	tabActiveColor   = lipgloss.Color("#FF5F87") // Pink
	tabInactiveColor = lipgloss.Color("#5F5F5F") // Dark gray
)

// Component styles
var (
	// Base styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// App title
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1).
			Align(lipgloss.Center)

	// Tab styles
	tabStyle = lipgloss.NewStyle().
			Padding(0, 3).
			Foreground(textColor).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(tabInactiveColor)

	activeTabStyle = tabStyle.
			Foreground(textColor).
			Background(tabActiveColor).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(tabActiveColor)

	// Info blocks
	infoStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(secondaryColor).
			Padding(0, 1).
			MarginBottom(1)

	// Metric style
	metricStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	metricValueStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// Dialog box style
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	// Button styles
	buttonStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(secondaryColor).
			Padding(0, 3).
			Margin(0, 1).
			Align(lipgloss.Center)

	// Status style
	statusStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 1).
			MarginTop(1).
			Bold(true)
)

// TUI implements the ports.UI interface using bubbletea
type TUI struct {
	searchConfig ports.SearchConfig
	outputFile   string
	limit        int
	program      *tea.Program
	model        Model
	stopChan     chan struct{}
}

// NewTUI creates a new TUI instance
func NewTUI(searchConfig ports.SearchConfig, outputFile string, limit int) ports.UI {
	model := Model{
		metrics: domain.Metrics{
			StartTime:        time.Now(),
			PartitionMetrics: []domain.PartitionMetrics{},
		},
		searchConfig: searchConfig,
		outputFile:   outputFile,
		limit:        limit,
	}

	tui := &TUI{
		searchConfig: searchConfig,
		outputFile:   outputFile,
		limit:        limit,
		model:        model,
		stopChan:     make(chan struct{}),
	}

	return tui
}

// Start starts the UI and returns a channel that will be closed when the UI is done
func (t *TUI) Start() <-chan struct{} {
	t.program = tea.NewProgram(t.model)

	// Start the UI in a separate goroutine
	go func() {
		if _, err := t.program.Run(); err != nil {
			fmt.Printf("Error running UI: %v\n", err)
		}
		close(t.stopChan)
	}()

	return t.stopChan
}

// UpdateMetrics sends updated metrics to the UI
func (t *TUI) UpdateMetrics(metrics domain.Metrics) {
	if t.program != nil {
		t.program.Send(MetricsMsg{Metrics: metrics})
	}
}

// DisplayResult shows a new search result
func (t *TUI) DisplayResult(result *domain.MatchResult) {
	if t.program != nil {
		t.program.Send(ResultMsg{Result: result})
	}
}

// Stop gracefully stops the UI
func (t *TUI) Stop() {
	if t.program != nil {
		t.program.Send(tea.Quit())
	}
}

// Model represents the TUI state
type Model struct {
	metrics      domain.Metrics
	searchConfig ports.SearchConfig
	latestResult *domain.MatchResult
	outputFile   string
	width        int
	height       int
	limit        int
	activeTab    int
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1", "2", "3", "4", "5":
			// Switch tabs
			tabIndex := int(msg.Runes[0] - '1')
			if tabIndex >= 0 && tabIndex < 5 {
				m.activeTab = tabIndex
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case MetricsMsg:
		m.metrics = msg.Metrics
	case ResultMsg:
		m.latestResult = msg.Result
	}
	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	// Apply app style
	docStyle := appStyle.Width(m.width).Height(m.height)

	// Create tabs
	tabs := []string{"Overview", "Partitions", "Matches", "Settings", "Help"}
	renderedTabs := make([]string, len(tabs))

	for i, tab := range tabs {
		if i == m.activeTab {
			renderedTabs[i] = activeTabStyle.Render(tab)
		} else {
			renderedTabs[i] = tabStyle.Render(tab)
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// App title with sparkles
	title := titleStyle.Render("✨ Kafka Explorer ✨")

	// Create header with tabs
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		tabBar,
	)

	// Main content based on active tab
	var content string

	switch m.activeTab {
	case 0: // Overview tab
		content = m.renderOverviewTab()
	case 1: // Partitions tab
		content = m.renderPartitionsTab()
	case 2: // Matches tab
		content = m.renderMatchesTab()
	case 3: // Settings tab
		content = m.renderSettingsTab()
	case 4: // Help tab
		content = m.renderHelpTab()
	}

	// Status bar at the bottom
	statusBar := statusStyle.Render(fmt.Sprintf(" STATUS | %s | UTF-8 | 🔍 %s ",
		"Active", m.searchConfig.SearchString))

	// Combine all sections
	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		statusBar,
	)

	// Apply app style
	return docStyle.Render(fullContent)
}

// formatNumber formats a number with K for thousands and M for millions
func formatNumber(num float64) string {
	if num >= 1000000 {
		return fmt.Sprintf("%.1fM", num/1000000)
	} else if num >= 1000 {
		return fmt.Sprintf("%.1fK", num/1000)
	}
	return fmt.Sprintf("%.0f", num)
}

// formatNumberWithSuffix formats a number with K for thousands and M for millions
// and adds a custom suffix
func formatNumberWithSuffix(num float64, suffix string) string {
	return formatNumber(num) + suffix
}

// renderOverviewTab renders the overview tab content
func (m Model) renderOverviewTab() string {
	// Search info
	searchInfo := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(fmt.Sprintf("🔍 Searching for: %s", m.searchConfig.SearchString))

	// Work mode and limit info
	limitText := "🔢 Limit: "
	if m.limit > 0 {
		limitText += fmt.Sprintf("%d matches", m.limit)
	} else {
		limitText += "unlimited"
	}

	// Global metrics
	elapsed := m.metrics.ElapsedTime.Round(time.Second)

	// Format global speed considering magnitude
	var speedText string
	if m.metrics.AvgMsgPerSecond >= 1000 {
		speedText = formatNumberWithSuffix(m.metrics.AvgMsgPerSecond, " msg/sec")
	} else {
		speedText = fmt.Sprintf("%.1f msg/sec", m.metrics.AvgMsgPerSecond)
	}

	// Create a colorful metrics box
	metricsBox := dialogBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Width(15).Render("⏱️  Time:"),
				metricValueStyle.Width(15).Render(elapsed.String()),
				metricStyle.Width(15).Render("📊 Messages:"),
				metricValueStyle.Width(15).Render(formatNumber(float64(m.metrics.TotalMessagesRead))),
			),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Width(15).Render("🔍 Matches:"),
				metricValueStyle.Width(15).Render(formatNumber(float64(m.metrics.TotalMatchesFound))),
				metricStyle.Width(15).Render("🚀 Speed:"),
				metricValueStyle.Width(15).Render(speedText),
			),
		),
	)

	// Latest match info
	var matchInfo string
	if m.latestResult == nil {
		matchInfo = "Matches not found yet"
	} else {
		msg := m.latestResult
		matchInfo = lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Render("Topic: "),
				metricValueStyle.Render(msg.Topic),
			),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Render("Partition: "),
				metricValueStyle.Render(fmt.Sprintf("%d", msg.Partition)),
			),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Render("Offset: "),
				metricValueStyle.Render(fmt.Sprintf("%d", msg.Offset)),
			),
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				metricStyle.Render("Time: "),
				metricValueStyle.Render(msg.Time.Format("15:04:05 02.01.2006")),
			),
		)
	}

	// Create a dialog box for the latest match
	matchBox := dialogBoxStyle.
		BorderForeground(primaryColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Foreground(primaryColor).
					Bold(true).
					Render("🔎 LATEST MATCH"),
				matchInfo,
			),
		)

	// Save info in a nice box
	saveBox := dialogBoxStyle.
		BorderForeground(accentColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Foreground(accentColor).
					Bold(true).
					Render("💾 OUTPUT"),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Render("Saving to: "),
					metricValueStyle.Render(m.outputFile),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Render("Limit: "),
					metricValueStyle.Render(limitText),
				),
			),
		)

	// Arrange boxes in a grid layout
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, metricsBox)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, matchBox, saveBox)

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchInfo,
		topRow,
		bottomRow,
		metricStyle.Render("Press 'q' to quit, '1-5' to switch tabs"),
	)
}

// renderPartitionsTab renders the partitions tab content
func (m Model) renderPartitionsTab() string {
	// Partition metrics section
	partitionsSection := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("📈 PARTITION METRICS")

	var partitionsContent string
	if len(m.metrics.PartitionMetrics) == 0 {
		partitionsContent = "No partitions available"
	} else {
		// Create sorted list of partitions
		sortedPartitions := make([]int, 0, len(m.metrics.PartitionMetrics))
		partitionMap := make(map[int]domain.PartitionMetrics)

		// Fill map and partition list
		for _, pm := range m.metrics.PartitionMetrics {
			partitionMap[pm.Partition] = pm
			// Add partition to list only if not already there
			found := false
			for _, p := range sortedPartitions {
				if p == pm.Partition {
					found = true
					break
				}
			}
			if !found {
				sortedPartitions = append(sortedPartitions, pm.Partition)
			}
		}

		// Sort partition list
		sort.Ints(sortedPartitions)

		// Create table for partition metrics in modern style
		columns := []string{"Partition", "Messages", "Matches", "Speed", "Last Message Time"}

		// Define column widths
		widths := []int{11, 14, 10, 12, 30}

		// Base table style
		tableStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

		// Header styles
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 1).
			Align(lipgloss.Center)

		// Cell styles for different types
		partitionCellStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1).
			Align(lipgloss.Left)

		messageCellStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1).
			Align(lipgloss.Right)

		matchCellStyle := lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Padding(0, 1).
			Align(lipgloss.Right)

		speedCellStyle := lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			Padding(0, 1).
			Align(lipgloss.Right)

		timeCellStyle := lipgloss.NewStyle().
			Foreground(subtextColor).
			Padding(0, 1).
			Align(lipgloss.Center)

		// Create headers with correct styles and widths
		renderedHeaders := make([]string, len(columns))
		for i, col := range columns {
			renderedHeaders[i] = headerStyle.Width(widths[i]).Render(col)
		}

		// Combine headers into a row
		headerRow := lipgloss.JoinHorizontal(lipgloss.Center, renderedHeaders...)

		// Create a nicer divider
		dividerStyle := lipgloss.NewStyle().
			Foreground(primaryColor)

		divider := dividerStyle.Render(strings.Repeat("━", lipgloss.Width(headerRow)))

		// Create table starting with header and divider
		tableContent := []string{headerRow, divider}

		// Add data rows
		for _, partition := range sortedPartitions {
			pm := partitionMap[partition]

			// Format speed
			var speedText string
			if pm.MsgPerSecond >= 1000 {
				speedText = formatNumberWithSuffix(pm.MsgPerSecond, "/s")
			} else {
				speedText = fmt.Sprintf("%.1f/s", pm.MsgPerSecond)
			}

			// Format last message time
			var timeText string
			if pm.LastMessageTime.IsZero() {
				timeText = "N/A"
			} else {
				timeText = pm.LastMessageTime.Format("15:04:05 02.01.2006")
			}

			// Add nice activity indicator for partition
			partitionText := fmt.Sprintf("%d", pm.Partition)
			messagesText := formatNumber(float64(pm.MessagesRead))
			matchesText := formatNumber(float64(pm.MatchesFound))

			// Highlight row by activity
			rowStyle := lipgloss.NewStyle()
			// Add slight gradient for alternating rows

			// Create activity indicator
			var activityIndicator string
			if pm.MsgPerSecond > 0 {
				activityIndicator = lipgloss.NewStyle().
					Foreground(successColor).
					SetString("● ").
					String()
			} else {
				activityIndicator = lipgloss.NewStyle().
					Foreground(subtextColor).
					SetString("○ ").
					String()
			}
			partitionText = activityIndicator + partitionText

			// Create table row with applied styles
			row := []string{
				partitionCellStyle.Width(widths[0]).Render(partitionText),
				messageCellStyle.Width(widths[1]).Render(messagesText),
				matchCellStyle.Width(widths[2]).Render(matchesText),
				speedCellStyle.Width(widths[3]).Render(speedText),
				timeCellStyle.Width(widths[4]).Render(timeText),
			}

			// Join cells into a row
			renderedRow := lipgloss.JoinHorizontal(lipgloss.Top, row...)
			// Apply row style
			renderedRow = rowStyle.Render(renderedRow)
			// Add row to table
			tableContent = append(tableContent, renderedRow)
		}

		// Join all table rows into one visual table
		table := lipgloss.JoinVertical(lipgloss.Left, tableContent...)

		// Apply final table style
		partitionsContent = tableStyle.Render(table)
	}

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		partitionsSection,
		partitionsContent,
	)
}

// renderMatchesTab renders the matches tab content
func (m Model) renderMatchesTab() string {
	matchesSection := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("🔎 MATCHES")

	var matchContent string
	if m.latestResult == nil {
		matchContent = "No matches found yet"
	} else {
		// Create a more detailed view of the latest match
		msg := m.latestResult

		// Create a styled box for the match details
		matchBox := dialogBoxStyle.
			BorderForeground(accentColor).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					lipgloss.JoinHorizontal(
						lipgloss.Top,
						metricStyle.Width(12).Render("Topic:"),
						metricValueStyle.Render(msg.Topic),
					),
					lipgloss.JoinHorizontal(
						lipgloss.Top,
						metricStyle.Width(12).Render("Partition:"),
						metricValueStyle.Render(fmt.Sprintf("%d", msg.Partition)),
					),
					lipgloss.JoinHorizontal(
						lipgloss.Top,
						metricStyle.Width(12).Render("Offset:"),
						metricValueStyle.Render(fmt.Sprintf("%d", msg.Offset)),
					),
					lipgloss.JoinHorizontal(
						lipgloss.Top,
						metricStyle.Width(12).Render("Time:"),
						metricValueStyle.Render(msg.Time.Format("15:04:05 02.01.2006")),
					),
					"",
					lipgloss.NewStyle().
						Foreground(highlightColor).
						Bold(true).
						Render("Message Content:"),
					lipgloss.NewStyle().
						Foreground(textColor).
						Render("(Preview of message content would be shown here)"),
				),
			)

		matchContent = matchBox
	}

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		matchesSection,
		matchContent,
		"",
		lipgloss.NewStyle().
			Foreground(subtextColor).
			Render("Total matches found: "+formatNumber(float64(m.metrics.TotalMatchesFound))),
	)
}

// renderSettingsTab renders the settings tab content
func (m Model) renderSettingsTab() string {
	settingsSection := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("⚙️ SETTINGS")

	// Create a settings form
	settingsForm := dialogBoxStyle.
		BorderForeground(secondaryColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Width(15).Render("Search String:"),
					metricValueStyle.Render(m.searchConfig.SearchString),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Width(15).Render("Output File:"),
					metricValueStyle.Render(m.outputFile),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Width(15).Render("Match Limit:"),
					metricValueStyle.Render(fmt.Sprintf("%d", m.limit)),
				),
				"",
				lipgloss.JoinHorizontal(
					lipgloss.Center,
					buttonStyle.Render("Save"),
					buttonStyle.
						Background(primaryColor).
						Render("Reset"),
				),
			),
		)

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		settingsSection,
		settingsForm,
	)
}

// renderHelpTab renders the help tab content
func (m Model) renderHelpTab() string {
	helpSection := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("❓ HELP")

	// Create a help box
	helpBox := dialogBoxStyle.
		BorderForeground(highlightColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Foreground(highlightColor).
					Bold(true).
					Render("Keyboard Shortcuts:"),
				"",
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Width(15).Render("1-5:"),
					metricValueStyle.Render("Switch tabs"),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					metricStyle.Width(15).Render("q, Ctrl+C:"),
					metricValueStyle.Render("Quit application"),
				),
				"",
				lipgloss.NewStyle().
					Foreground(highlightColor).
					Bold(true).
					Render("About:"),
				"",
				lipgloss.NewStyle().
					Foreground(textColor).
					Render("Kafka Explorer CLI - A tool for searching Kafka topics"),
				lipgloss.NewStyle().
					Foreground(subtextColor).
					Render("Version 1.0.0"),
			),
		)

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		helpSection,
		helpBox,
	)
}

// Message types
type MetricsMsg struct {
	Metrics domain.Metrics
}

type ResultMsg struct {
	Result *domain.MatchResult
}
