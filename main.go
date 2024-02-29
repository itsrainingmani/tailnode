package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	countries         = []string{
		"Albania",
		"Australia",
		"Austria",
		"Belgium",
		"Brazil",
		"Bulgaria",
		"Canada",
		"Colombia",
		"Croatia",
		"Czech Republic",
		"Denmark",
		"Estonia",
		"Finland",
		"France",
		"Germany",
		"Greece",
		"Hong Kong",
		"Hungary",
		"Ireland",
		"Israel",
		"Italy",
		"Japan",
		"Latvia",
		"Mexico",
		"Netherlands",
		"New Zealand",
		"Norway",
		"Poland",
		"Portugal",
		"Romania",
		"Serbia",
		"Singapore",
		"Slovakia",
		"South Africa",
		"Spain",
		"Sweden",
		"Switzerland",
		"UK",
		"USA",
		"Ukraine",
	}
)

type item string

type CityInfo struct {
	City     string
	Hostname string
	IP       string
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func changeExitNode(ctx context.Context, exitNode string) error {
	cmd := exec.CommandContext(ctx, "tailscale", "set", "--exit-node="+exitNode)

	return cmd.Run()
}

// Define the model which holds the application state
type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not interested? That's cool")
	}
	return "\n" + m.list.View()
}

func openNewTerminalWithCommand(ctx context.Context) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/k", "pause")
	default:
		return fmt.Errorf("unsupported platform")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createCountriesList() []list.Item {
	var items []list.Item
	for _, country := range countries {
		items = append(items, item(country))
	}
	return items
}

func main() {
	items := createCountriesList()
	m := model{list: list.New(items, itemDelegate{}, 20, 30)}
	m.list.Title = "Tailscale Exit Nodes"
	m.list.SetShowStatusBar(false)
	m.list.Styles.Title = titleStyle
	m.list.Styles.PaginationStyle = paginationStyle
	m.list.Styles.HelpStyle = helpStyle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := openNewTerminalWithCommand(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open new terminal: %v\n", err)
		os.Exit(1)
	}

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
