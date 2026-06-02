package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v60/github"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Set GITHUB_TOKEN environment variable")
	}

	client := github.NewClient(nil).WithAuthToken(token)
	ctx := context.Background()

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Fatal(err)
	}

	prs, _, err := client.Search.Issues(ctx, fmt.Sprintf("is:pr is:open author:%s", user.GetLogin()), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Open PRs for @%s: %d\n\n", user.GetLogin(), prs.GetTotal())
	for _, pr := range prs.Issues {
		fmt.Printf("  %s\n  %s\n\n", pr.GetTitle(), pr.GetHTMLURL())
	}
}
