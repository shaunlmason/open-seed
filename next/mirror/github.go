package mirror

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GitHubTokenEnv is the environment variable the GitHub exporter reads
// its token from when the config names none.
const GitHubTokenEnv = "SEED_MIRROR_GITHUB_TOKEN"

// github mirrors into a GitHub repository's issues through the REST
// API: list (pull requests filtered out, since GitHub lists them as
// issues), create, and edit. Labels are named strings on GitHub, so
// the managed label needs no id lookup.
type github struct {
	c    *forgeClient
	repo string
}

func newGitHub(cfg Config) (Adapter, error) {
	if cfg.Owner == "" || cfg.Repo == "" {
		return nil, errors.New("the github exporter needs --owner and --repo")
	}
	env := cfg.TokenEnv
	if env == "" {
		env = GitHubTokenEnv
	}
	tok, err := tokenFrom(env)
	if err != nil {
		return nil, err
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	return &github{
		c:    &forgeClient{base: base, token: tok, scheme: "Bearer", accept: "application/vnd.github+json", http: http.DefaultClient},
		repo: "/repos/" + cfg.Owner + "/" + cfg.Repo,
	}, nil
}

func (g *github) Name() string { return "github" }

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest *struct{} `json:"pull_request,omitempty"`
}

func (i ghIssue) issue() Issue {
	labels := []string{}
	for _, l := range i.Labels {
		labels = append(labels, l.Name)
	}
	return Issue{ID: strconv.Itoa(i.Number), Title: i.Title, Body: i.Body, Labels: labels, Closed: i.State == "closed"}
}

func (g *github) List(ctx context.Context) ([]Issue, error) {
	out := []Issue{}
	for page := 1; ; page++ {
		var got []ghIssue
		if _, err := g.c.do(ctx, http.MethodGet, fmt.Sprintf("%s/issues?state=all&per_page=100&page=%d", g.repo, page), nil, &got); err != nil {
			return nil, err
		}
		for _, i := range got {
			if i.PullRequest != nil {
				continue
			}
			out = append(out, i.issue())
		}
		if len(got) < 100 {
			return out, nil
		}
	}
}

func ghState(closed bool) string {
	if closed {
		return "closed"
	}
	return "open"
}

func (g *github) Create(ctx context.Context, d Desired) (Issue, error) {
	var got ghIssue
	body := map[string]any{"title": d.Title, "body": d.Body, "labels": []string{d.Label}}
	if _, err := g.c.do(ctx, http.MethodPost, g.repo+"/issues", body, &got); err != nil {
		return Issue{}, err
	}
	if d.Closed {
		if _, err := g.c.do(ctx, http.MethodPatch, g.repo+"/issues/"+strconv.Itoa(got.Number), map[string]any{"state": "closed"}, &got); err != nil {
			return Issue{}, err
		}
	}
	return got.issue(), nil
}

func (g *github) Update(ctx context.Context, id string, d Desired, foreign []string) error {
	body := map[string]any{"title": d.Title, "body": d.Body, "state": ghState(d.Closed), "labels": append(append([]string{}, foreign...), d.Label)}
	_, err := g.c.do(ctx, http.MethodPatch, g.repo+"/issues/"+id, body, nil)
	return err
}
