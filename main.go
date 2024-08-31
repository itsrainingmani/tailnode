package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

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

	CountryCityMap = map[string][]string{
		"Albania":        {"Tirana"},
		"Australia":      {"Adelaide", "Brisbane", "Melbourne", "Perth", "Sydney"},
		"Austria":        {"Vienna"},
		"Belgium":        {"Brussels"},
		"Brazil":         {"Sao Paulo"},
		"Bulgaria":       {"Sofia"},
		"Canada":         {"Calgary", "Montreal", "Toronto", "Vancouver"},
		"Chile":          {"Santiago"},
		"Colombia":       {"Bogota"},
		"Croatia":        {"Zagreb"},
		"Czech Republic": {"Prague"},
		"Denmark":        {"Copenhagen"},
		"Estonia":        {"Tallinn"},
		"Finland":        {"Helsinki"},
		"France":         {"Bordeaux", "Marseille", "Paris"},
		"Germany":        {"Berlin", "Dusseldorf", "Frankfurt"},
		"Greece":         {"Athens"},
		"Hong Kong":      {"Hong Kong"},
		"Hungary":        {"Budapest"},
		"Indonesia":      {"Jakarta"},
		"Ireland":        {"Dublin"},
		"Israel":         {"Tel Aviv"},
		"Italy":          {"Milan", "Palermo"},
		"Japan":          {"Osaka", "Tokyo"},
		"Latvia":         {"Riga"},
		"Mexico":         {"Queretaro"},
		"Netherlands":    {"Amsterdam"},
		"New Zealand":    {"Auckland"},
		"Norway":         {"Oslo", "Stavanger"},
		"Peru":           {"Lima"},
		"Poland":         {"Warsaw"},
		"Portugal":       {"Lisbon"},
		"Romania":        {"Bucharest"},
		"Serbia":         {"Belgrade"},
		"Singapore":      {"Singapore"},
		"Slovakia":       {"Bratislava"},
		"Slovenia":       {"Ljubljana"},
		"South Africa":   {"Johannesburg"},
		"Spain":          {"Barcelona", "Madrid", "Valencia"},
		"Sweden":         {"Gothenburg", "Malmo", "Stockholm"},
		"Switzerland":    {"Zurich"},
		"Thailand":       {"Bangkok"},
		"Turkey":         {"Istanbul"},
		"UK":             {"Glasgow", "London", "Manchester"},
		"USA": {
			"Ashburn", "Atlanta", "Boston", "Chicago", "Dallas", "Denver",
			"Detroit", "Houston", "LosAngeles", "McAllen", "Miami", "NewYork",
			"Phoenix", "Raleigh", "SaltLakeCity", "SanJose", "Seattle", "Secaucus",
		},
		"Ukraine": {"Kyiv"},
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
	countryList list.Model
	cityList    list.Model
	state       string // "country" or "city"
	choice      string
	quitting    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.countryList.SetWidth(msg.Width)
		m.cityList.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			time.Sleep(1 * time.Second)
			return m, tea.Quit

		case "enter":
			if m.state == "country" {
				country := string(m.countryList.SelectedItem().(item))
				m.cityList = list.New(createCityList(country), itemDelegate{}, 20, 40)
				m.state = "city"
				return m, nil
			} else if m.state == "city" {
				city := string(m.cityList.SelectedItem().(item))
				m.choice = city
				time.Sleep(1 * time.Second)
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	if m.state == "country" {
		m.countryList, cmd = m.countryList.Update(msg)
	} else {
		m.cityList, cmd = m.cityList.Update(msg)
	}
	return m, cmd
}
func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not interested? That's cool")
	}
	if m.state == "country" {
		return "\n" + m.countryList.View()
	}
	return "\n" + m.cityList.View()
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

func createCountriesList() []list.Item {
	var items []list.Item
	for _, country := range countries {
		items = append(items, item(country))
	}
	return items
}

func createCityList(country string) []list.Item {
	var items []list.Item
	for _, city := range CountryCityMap[country] {
		items = append(items, item(city))
	}
	return items
}

func main() {
	countryItems := createCountriesList()
	m := model{
		countryList: list.New(countryItems, itemDelegate{}, 20, 40),
		state:       "country",
	}
	m.countryList.Title = "Select a Country"
	m.countryList.SetShowStatusBar(false)
	m.countryList.Styles.Title = titleStyle
	m.countryList.Styles.PaginationStyle = paginationStyle
	m.countryList.Styles.HelpStyle = helpStyle

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
