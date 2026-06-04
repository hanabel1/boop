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
	// https://github.com/{owner}/{repo}/pull/{number}
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

type model struct {
	prs      []pr
	username string
	loading  bool
	err      error
	width    int
	height   int
	selected int
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

func initialModel() model {
	return model{loading: true}
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
	if m.loading {
		return "\n  Loading PRs..."
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}

	leftWidth := m.width/2 - 2
	rightWidth := m.width - leftWidth - 3
	contentHeight := m.height - 2

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
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(1)

	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var leftContent strings.Builder
	leftContent.WriteString(fmt.Sprintf("@%s — %d open\n\n", m.username, len(m.prs)))
	for i, p := range m.prs {
		prefix := "  "
		titleLine := p.title
		repoLine := dimStyle.Render(p.repo)
		if i == m.selected {
			prefix = "> "
			titleLine = selectedStyle.Render(p.title)
			repoLine = selectedStyle.Render(p.repo)
		}
		leftContent.WriteString(fmt.Sprintf("%s%s\n    %s\n\n", prefix, titleLine, repoLine))
	}

	var rightContent string
	if len(m.prs) > 0 {
		sp := m.prs[m.selected]
		var detail strings.Builder
		detail.WriteString(selectedStyle.Render(sp.title) + "\n\n")
		detail.WriteString(dimStyle.Render("repo  ") + sp.repo + "\n")
		detail.WriteString(dimStyle.Render("url   ") + sp.url + "\n")

		if sp.body != "" {
			detail.WriteString("\n" + dimStyle.Render("description") + "\n")
			body := sp.body
			if len(body) > 300 {
				body = body[:300] + "..."
			}
			detail.WriteString(body + "\n")
		}

		if len(sp.commits) > 0 {
			detail.WriteString("\n" + dimStyle.Render("recent commits") + "\n")
			for _, msg := range sp.commits {
				firstLine := strings.SplitN(msg, "\n", 2)[0]
				if len(firstLine) > 60 {
					firstLine = firstLine[:57] + "..."
				}
				detail.WriteString("  • " + firstLine + "\n")
			}
		}

		rightContent = detail.String()
	} else {
		rightContent = "no open PRs"
	}

	left := leftStyle.Render(leftContent.String())
	right := rightStyle.Render(rightContent)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return layout + "\n  q to quit"
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
