package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/v60/github"
)

type pr struct {
	title string
	url   string
	repo  string
}

type model struct {
	prs      []pr
	username string
	loading  bool
	err      error
	width    int
	height   int
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
		prs = append(prs, pr{
			title: issue.GetTitle(),
			url:   issue.GetHTMLURL(),
			repo:  issue.GetRepository().GetFullName(),
		})
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
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
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

	var leftContent strings.Builder
	leftContent.WriteString(fmt.Sprintf("@%s — %d open\n\n", m.username, len(m.prs)))
	for i, p := range m.prs {
		leftContent.WriteString(fmt.Sprintf(" %d. %s\n    %s\n\n", i+1, p.title, p.repo))
	}

	rightContent := "← select a PR to see details"

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
