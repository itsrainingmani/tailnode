package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func changeExitNode(ctx context.Context, exitNode string) error {
	cmd := exec.CommandContext(ctx, "tailscale", "set", "--exit-node="+exitNode)

	return cmd.Run()
}

func getCurrentExitNode() string {
	cmd := exec.Command("tailscale", "exit-node", "list")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "selected") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				country := fields[2]
				city := fields[3]
				if city == "Any" {
					continue
				}
				return country + ", " + city
			}
		}
	}

	return ""
}

func generateMullvadServers() ([]table.Row, error) {
	cmd := exec.Command("tailscale", "exit-node", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute tailscale command: %v", err)
	}

	var servers []table.Row
	lines := strings.Split(string(output), "\n")
	re := regexp.MustCompile(`\s{2,}`)

	for _, line := range lines[2:] { // Skip the header line
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := re.Split(strings.TrimSpace(line), -1)
		if len(fields) >= 5 {
			ip := fields[0]
			hostname := fields[1]
			country := fields[2]
			city := fields[3]
			if city == "Any" {
				continue
			}
			servers = append(servers, table.Row{ip, hostname, country, city})
		}
	}

	return servers, nil
}

// Define the model which holds the application state
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table           table.Model
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
		Bold(false)
	t.SetStyles(s)

	curExitNode := getCurrentExitNode()

	return model{table: t, execCtx: ctx, quitting: false, err: nil, currentExitNode: curExitNode}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			selectedRow := m.table.SelectedRow()
			exitNodeHostname := selectedRow[1]
			changeExitNode(m.execCtx, exitNodeHostname)
			m.currentExitNode = selectedRow[2] + ", " + selectedRow[3]
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	str := baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n"
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
