package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v60/github"
)

func parseGitHubPRURL(url string) (owner, repo string, number int) {
	parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
	if len(parts) >= 4 && parts[2] == "pull" {
		number, _ = strconv.Atoi(parts[3])
		return parts[0], parts[1], number
	}
	return "", "", 0
}

type pr struct {
	title   string
	url     string
	repo    string
	body    string
	commits []string
}

type theme struct {
	primaryBorder string
	dimBorder     string
	selectedRow   string
	dimText       string
	headerText    string
	accent        string
	bullet        string
	cursor        string
	sparkle       string
	flower        string
	loading       string
	empty         string
	banner        string
}

var themes = map[string]theme{
	"pastel": {
		primaryBorder: "211",
		dimBorder:     "183",
		selectedRow:   "219",
		dimText:       "182",
		headerText:    "213",
		accent:        "117",
		bullet:        "🌸",
		cursor:        "🐾",
		sparkle:       "✨",
		flower:        "🌷",
		loading:       "       /\\_/\\\n      ( o.o )\n       > ^ <\n      /|   |\\\n     (_|   |_)\n\n    🐱 fetching your PRs~",
		empty:         "🌙 nothing open — go take a nap!",
		banner:        "·˚ ♡ ·˚ ✧ ·˚ ♡ ·˚ ✧",
	},
	"y2k": {
		primaryBorder: "198",
		dimBorder:     "154",
		selectedRow:   "201",
		dimText:       "120",
		headerText:    "198",
		accent:        "51",
		bullet:        "💎",
		cursor:        "⭐",
		sparkle:       "💫",
		flower:        "🦋",
		loading:       "    ╔══════════════╗\n    ║  ★ B O O P ★ ║\n    ║   ·411·1010·  ║\n    ╚══════════════╝\n\n     💿 loading ur PRs...",
		empty:         "🛸 inbox zero bb!",
		banner:        "★·.·´¯`·.·★ boop ★·.·´¯`·.·★",
	},
	"cottagecore": {
		primaryBorder: "181",
		dimBorder:     "107",
		selectedRow:   "223",
		dimText:       "144",
		headerText:    "181",
		accent:        "150",
		bullet:        "🌿",
		cursor:        "🍄",
		sparkle:       "🌻",
		flower:        "🌼",
		loading:       "                    .-'~~~-.\n                   .'o  oOOOo`.\n                  :~~~-.oOo   o`.\n                   `. \\ ~-.  oOOo.\n                     `.; / ~.  OO:\n                     .'  ;-- `.o.'\n                    ,'  ; ~~--'~\n                    ;  ;\n  _______\\|/__________\\\\;_\\\\//___\\|/________\n\n         🐝 gathering your PRs\n            from the garden...",
		empty:         "🌾 the meadow is quiet — no open PRs",
		banner:        "~* 🌻 ~ 🌸 ~ 🌼 ~ 🌿 *~",
	},
}

func loadTheme() theme {
	home, err := os.UserHomeDir()
	if err != nil {
		return themes["pastel"]
	}
	data, err := os.ReadFile(home + "/.boop")
	if err != nil {
		return themes["pastel"]
	}
	line := strings.TrimSpace(string(data))
	if strings.HasPrefix(line, "theme=") {
		name := strings.TrimPrefix(line, "theme=")
		if t, ok := themes[name]; ok {
			return t
		}
	}
	return themes["pastel"]
}

func themeName(t theme) string {
	for name, th := range themes {
		if th == t {
			return name
		}
	}
	return "pastel"
}

type model struct {
	prs      []pr
	username string
	loading  bool
	err      error
	width    int
	height   int
	selected int
	theme    theme
}

type prsLoadedMsg struct {
	prs      []pr
	username string
}

type errMsg struct{ err error }

func fetchPRs() tea.Msg {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errMsg{fmt.Errorf("GITHUB_TOKEN not set")}
	}

	client := github.NewClient(nil).WithAuthToken(token)
	ctx := context.Background()

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return errMsg{err}
	}

	result, _, err := client.Search.Issues(ctx, fmt.Sprintf("is:pr is:open author:%s", user.GetLogin()), nil)
	if err != nil {
		return errMsg{err}
	}

	var prs []pr
	for _, issue := range result.Issues {
		htmlURL := issue.GetHTMLURL()
		owner, repo, number := parseGitHubPRURL(htmlURL)

		p := pr{
			title: issue.GetTitle(),
			url:   htmlURL,
			repo:  owner + "/" + repo,
			body:  issue.GetBody(),
		}

		if owner != "" && repo != "" && number > 0 {
			commits, _, err := client.PullRequests.ListCommits(ctx, owner, repo, number, &github.ListOptions{PerPage: 5})
			if err == nil {
				for _, c := range commits {
					p.commits = append(p.commits, c.GetCommit().GetMessage())
				}
			}
		}

		prs = append(prs, p)
	}

	return prsLoadedMsg{prs: prs, username: user.GetLogin()}
}

func initialModel(t theme) model {
	return model{loading: true, theme: t}
}

func (m model) Init() tea.Cmd {
	return fetchPRs
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.prs)-1 {
				m.selected++
			}
		case "enter":
			if len(m.prs) > 0 {
				url := m.prs[m.selected].url
				exec.Command("open", url).Start()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case prsLoadedMsg:
		m.loading = false
		m.prs = msg.prs
		m.username = msg.username
	case errMsg:
		m.loading = false
		m.err = msg.err
	}
	return m, nil
}

func (m model) View() string {
	t := m.theme
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.dimText))

	if m.loading {
		loadBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.primaryBorder)).
			Padding(2, 4).
			Render(t.loading)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loadBox)
	}
	if m.err != nil {
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 3).
			Render(fmt.Sprintf("😿 oh no: %v", m.err))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errBox)
	}

	// Header
	bannerLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.dimBorder)).
		Render(t.banner)

	boopTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.primaryBorder)).
		Render(fmt.Sprintf("%s boop", t.cursor))

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.headerText)).
		Render(fmt.Sprintf("%s @%s · %d open PRs", t.sparkle, m.username, len(m.prs)))

	separator := dimStyle.
		Render(strings.Repeat("─", m.width))

	header := bannerLine + "\n" + boopTitle + "\n" + subtitle + "\n" + separator

	// Layout dimensions
	leftWidth := m.width/2 - 2
	rightWidth := m.width - leftWidth - 3
	contentHeight := m.height - 5

	if leftWidth < 10 {
		leftWidth = 40
	}
	if rightWidth < 10 {
		rightWidth = 40
	}
	if contentHeight < 5 {
		contentHeight = 20
	}

	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.primaryBorder)).
		Padding(1)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.dimBorder)).
		Padding(1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.selectedRow)).
		Bold(true)
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.accent))

	// Left panel — PR list
	var leftContent strings.Builder
	for i, p := range m.prs {
		if i == m.selected {
			leftContent.WriteString(selectedStyle.Render(fmt.Sprintf("%s %s", t.cursor, p.title)) + "\n")
			leftContent.WriteString("    " + dimStyle.Render(p.repo) + "\n\n")
		} else {
			leftContent.WriteString(fmt.Sprintf("  %s %s\n", t.bullet, p.title))
			leftContent.WriteString("    " + dimStyle.Render(p.repo) + "\n\n")
		}
	}

	// Right panel — PR details
	var rightContent string
	if len(m.prs) > 0 {
		sp := m.prs[m.selected]
		var detail strings.Builder

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.headerText))
		detail.WriteString(titleStyle.Render(sp.title) + "\n\n")

		detail.WriteString(fmt.Sprintf("%s  %s\n", t.flower, sp.repo))
		detail.WriteString(fmt.Sprintf("%s  %s\n", t.sparkle, accentStyle.Render(sp.url)))

		detail.WriteString("\n" + dimStyle.Render(strings.Repeat("~ ", (rightWidth-4)/2)) + "\n")

		if sp.body != "" {
			bodyPreview := sp.body
			if len(bodyPreview) > 200 {
				bodyPreview = bodyPreview[:200] + "..."
			}
			detail.WriteString("\n" + dimStyle.Render(bodyPreview) + "\n")
		}

		if len(sp.commits) > 0 {
			detail.WriteString(fmt.Sprintf("\n%s recent commits\n", t.bullet))
			for _, c := range sp.commits {
				msg := c
				if idx := strings.Index(msg, "\n"); idx != -1 {
					msg = msg[:idx]
				}
				if len(msg) > 50 {
					msg = msg[:50] + "..."
				}
				detail.WriteString(dimStyle.Render(fmt.Sprintf("  · %s\n", msg)))
			}
		}

		rightContent = detail.String()
	} else {
		rightContent = t.empty
	}

	left := leftStyle.Render(leftContent.String())
	right := rightStyle.Render(rightContent)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	themeHint := dimStyle.Render(fmt.Sprintf("%s theme: %s", t.sparkle, themeName(m.theme)))
	navHint := dimStyle.Render("j/k to navigate · enter to open · q to quit")
	full := header + "\n" + layout + "\n  " + navHint + "  ·  " + themeHint

	return lipgloss.NewStyle().Height(m.height).Render(full)
}

func main() {
	t := loadTheme()
	p := tea.NewProgram(initialModel(t), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
