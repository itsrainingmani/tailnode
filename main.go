package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define the model which holds the application state
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

type model struct {
	table           table.Model
	nodeInput       textinput.Model
	servers         []table.Row
	execCtx         context.Context
	currentExitNode string
	quitting        bool
	err             error
}

func initialModel(ctx context.Context, servers []table.Row) model {

	columns := []table.Column{
		{Title: "IP", Width: 20},
		{Title: "Hostname", Width: 30},
		{Title: "Country", Width: 20},
		{Title: "City", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(servers),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		Italic(true)
	t.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "New York"
	ti.CharLimit = 30
	ti.Width = 50
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	curExitNode := getCurrentExitNode()

	return model{table: t, nodeInput: ti, execCtx: ctx, quitting: false, err: nil, currentExitNode: curExitNode, servers: servers}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.table.Focused() {
		cmd = m.updateTable(msg)
	} else {
		cmd = m.updateInput(msg)
	}

	return m, cmd
}

func (m *model) updateTable(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "tab", "shift+tab":
			m.table.Blur()
			m.nodeInput.Focus()
			s := table.DefaultStyles()
			s.Header = s.Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				BorderBottom(true).
				Bold(false)
			m.table.SetStyles(s)
		case "q", "ctrl+c":
			return tea.Quit
		case "backspace":
			m.currentExitNode = ""
			changeExitNode(m.execCtx, m.currentExitNode)
		case "enter":
			selectedRow := m.table.SelectedRow()
			exitNodeHostname := selectedRow[1]
			changeExitNode(m.execCtx, exitNodeHostname)
			m.currentExitNode = selectedRow[2] + ", " + selectedRow[3]
			return tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return cmd
}

func (m *model) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "tab", "shift+tab":
			m.nodeInput.Blur()
			m.table.Focus()
			s := table.DefaultStyles()
			s.Header = s.Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				BorderBottom(true).
				Bold(false)
			s.Selected = s.Selected.
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(true).
				Italic(true)
			m.table.SetStyles(s)
		}
	}
	m.nodeInput, cmd = m.nodeInput.Update(msg)

	filterNode := strings.ToLower(m.nodeInput.Value())

	if len(filterNode) > 0 {
		var newRows []table.Row
		curRows := m.table.Rows()
		for _, row := range curRows {
			if strings.Contains(strings.ToLower(row[2]), filterNode) || // Country
				strings.Contains(strings.ToLower(row[3]), filterNode) { // City
				newRows = append(newRows, row)
			}
		}
		m.table.SetRows(newRows)
	} else {
		// Reset to original rows when input is empty
		m.table.SetRows(m.servers)
	}
	m.table.UpdateViewport()

	return cmd
}

func (m model) View() string {
	str := "TailNode - Choose an Exit Node\n" + baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n" + m.nodeInput.View()
	if m.currentExitNode != "" {
		return str + "\n" + "Current Exit Node is: " + m.currentExitNode
	}
	return str
}

func openNewTerminalWithCommand(ctx context.Context) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/k", "pause")
	case "darwin": // macOS
		cmd = exec.CommandContext(ctx, "open", "-a", "iTerm")
	case "linux", "freebsd", "netbsd", "openbsd": // Unix-like OSes
		cmd = exec.CommandContext(ctx, "x-terminal-emulator", "-e", "bash -c 'echo Press Enter to continue; read line'")
	default:
		return fmt.Errorf("unsupported platform")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := openNewTerminalWithCommand(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open new terminal: %v\n", err)
		os.Exit(1)
	}

	servers, err := generateMullvadServers()
	if err != nil {
		fmt.Printf("Error getting Tailscale Exit Nodes: %v\n", err)
		os.Exit(1)
	}

	m := initialModel(ctx, servers)

	opts := []tea.ProgramOption{
		tea.WithContext(ctx), // Pass the context to the program
		tea.WithAltScreen(),  // Enable alternate screen buffer
	}

	// Create the Bubble Tea program
	p := tea.NewProgram(m, opts...)

	// Start the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
