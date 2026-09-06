package protections

// The forge's word on a pull request's head (plans/os-0cd18799.md D5;
// next/spec/observations-forge.md): the combined state of the checks
// it ran, the review threads still open, and the review state, read
// the way Merged reads the merge state and never written. GitHub and
// Forgejo answer the same shape from their own endpoints; the snapshot
// arm answers it from a file so every drill is credential-free. What a
// forge cannot say is nil, never zero: Forgejo has no thread
// resolution, and the observation says so rather than reporting none.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Observation is what the forge says about a pull request's head:
// forge-neutral, and exactly what check.observed carries.
type Observation struct {
	// Head is the pull request's current head commit, a full sha.
	Head string
	// Checks is green, red or pending: every check run completed
	// passing; any completed otherwise; else something still running.
	// A pull request the forge lists no checks for is green: nothing
	// failed and nothing is running, the v1 engine's gate posture.
	Checks string
	// UnresolvedThreads is the count of review threads not marked
	// resolved, nil where the forge cannot say.
	UnresolvedThreads *int
	// Review is approved, changes_requested or none: the latest
	// review state per reviewer, reduced.
	Review string
}

// prHead is the head-bearing subset both forges' pull-request objects
// carry beside the merge state.
type prHead struct {
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// Checks reads a GitHub pull request's head, check runs, review
// threads and reviews.
func (g *GitHub) Checks(pr string) (Observation, error) {
	n, err := prNumber(pr)
	if err != nil {
		return Observation{}, err
	}
	var head prHead
	if _, err := g.do(http.MethodGet, g.repoPath()+"/pulls/"+n, nil, &head); err != nil {
		return Observation{}, err
	}
	if head.Head.SHA == "" {
		return Observation{}, fmt.Errorf("pull request %s names no head", pr)
	}
	var runs struct {
		CheckRuns []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"check_runs"`
	}
	if _, err := g.do(http.MethodGet, g.repoPath()+"/commits/"+head.Head.SHA+"/check-runs?per_page=100", nil, &runs); err != nil {
		return Observation{}, err
	}
	checks := "green"
	for _, c := range runs.CheckRuns {
		if c.Status != "completed" {
			if checks == "green" {
				checks = "pending"
			}
			continue
		}
		switch c.Conclusion {
		case "success", "neutral", "skipped":
		default:
			checks = "red"
		}
	}
	threads, err := g.unresolvedThreads(n)
	if err != nil {
		return Observation{}, err
	}
	var reviews []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		State string `json:"state"`
	}
	if _, err := g.do(http.MethodGet, g.repoPath()+"/pulls/"+n+"/reviews?per_page=100", nil, &reviews); err != nil {
		return Observation{}, err
	}
	latest := map[string]string{}
	order := []string{}
	for _, r := range reviews {
		switch r.State {
		case "APPROVED", "CHANGES_REQUESTED":
			if _, seen := latest[r.User.Login]; !seen {
				order = append(order, r.User.Login)
			}
			latest[r.User.Login] = r.State
		}
	}
	return Observation{Head: head.Head.SHA, Checks: checks, UnresolvedThreads: &threads, Review: reduceReviews(latest, "CHANGES_REQUESTED", "APPROVED")}, nil
}

// unresolvedThreads counts the pull request's review threads not marked
// resolved through the GraphQL API, the one place GitHub exposes
// thread resolution, paging past the first hundred.
func (g *GitHub) unresolvedThreads(n string) (int, error) {
	owner, repo := g.Owner, g.Repo
	const query = `query($o:String!,$n:String!,$pr:Int!,$after:String){repository(owner:$o,name:$n){pullRequest(number:$pr){reviewThreads(first:100,after:$after){pageInfo{hasNextPage endCursor}nodes{isResolved}}}}}`
	var number int
	if _, err := fmt.Sscanf(n, "%d", &number); err != nil {
		return 0, fmt.Errorf("pr %q is not a number", n)
	}
	unresolved := 0
	var after *string
	for {
		body := map[string]any{"query": query, "variables": map[string]any{"o": owner, "n": repo, "pr": number, "after": after}}
		var resp struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						ReviewThreads struct {
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
							Nodes []struct {
								IsResolved bool `json:"isResolved"`
							} `json:"nodes"`
						} `json:"reviewThreads"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if _, err := g.do(http.MethodPost, "/graphql", body, &resp); err != nil {
			return 0, err
		}
		if len(resp.Errors) > 0 {
			return 0, fmt.Errorf("github graphql: %s", resp.Errors[0].Message)
		}
		page := resp.Data.Repository.PullRequest.ReviewThreads
		for _, node := range page.Nodes {
			if !node.IsResolved {
				unresolved++
			}
		}
		if !page.PageInfo.HasNextPage {
			return unresolved, nil
		}
		cursor := page.PageInfo.EndCursor
		after = &cursor
	}
}

// Checks reads a Forgejo pull request's head, combined commit status
// and reviews. Forgejo has no thread resolution, so the thread count
// is nil: what the forge cannot say is named, never dropped.
func (f *Forgejo) Checks(pr string) (Observation, error) {
	n, err := prNumber(pr)
	if err != nil {
		return Observation{}, err
	}
	var head prHead
	if _, err := f.do(http.MethodGet, f.repoPath()+"/pulls/"+n, nil, &head); err != nil {
		return Observation{}, err
	}
	if head.Head.SHA == "" {
		return Observation{}, fmt.Errorf("pull request %s names no head", pr)
	}
	var status struct {
		State string `json:"state"`
	}
	if _, err := f.do(http.MethodGet, f.repoPath()+"/commits/"+head.Head.SHA+"/status", nil, &status); err != nil {
		return Observation{}, err
	}
	checks := "green"
	switch strings.ToLower(status.State) {
	case "failure", "error":
		checks = "red"
	case "pending":
		checks = "pending"
	}
	var reviews []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		State string `json:"state"`
	}
	if _, err := f.do(http.MethodGet, f.repoPath()+"/pulls/"+n+"/reviews", nil, &reviews); err != nil {
		return Observation{}, err
	}
	latest := map[string]string{}
	for _, r := range reviews {
		switch r.State {
		case "APPROVED", "REQUEST_CHANGES":
			latest[r.User.Login] = r.State
		}
	}
	return Observation{Head: head.Head.SHA, Checks: checks, Review: reduceReviews(latest, "REQUEST_CHANGES", "APPROVED")}, nil
}

// reduceReviews folds the latest review state per reviewer into the
// observation's literal: any reviewer still requesting changes wins,
// else any approval, else none.
func reduceReviews(latest map[string]string, changes, approved string) string {
	review := "none"
	for _, state := range latest {
		if state == changes {
			return "changes_requested"
		}
		if state == approved {
			review = "approved"
		}
	}
	return review
}

// Checks reads the pull request's observation from the snapshot:
// {"pulls": {"pr/1": {"head": "<sha>", "checks": "red",
// "unresolved_threads": 1, "review": "none", ...}}}, beside the merge
// fields Merged reads.
func (s SnapshotObserver) Checks(pr string) (Observation, error) {
	var doc struct {
		Pulls map[string]struct {
			Head    string `json:"head"`
			Checks  string `json:"checks"`
			Threads *int   `json:"unresolved_threads"`
			Review  string `json:"review"`
		} `json:"pulls"`
	}
	if err := s.load(&doc); err != nil {
		return Observation{}, err
	}
	st, ok := doc.Pulls[pr]
	if !ok {
		return Observation{}, fmt.Errorf("the snapshot names no pull request %q", pr)
	}
	if st.Head == "" || st.Checks == "" {
		return Observation{}, fmt.Errorf("the snapshot's pull request %q carries no head or checks", pr)
	}
	review := st.Review
	if review == "" {
		review = "none"
	}
	return Observation{Head: st.Head, Checks: st.Checks, UnresolvedThreads: st.Threads, Review: review}, nil
}

// load reads and parses the snapshot file into the caller's shape.
func (s SnapshotObserver) load(out any) error {
	b, err := readSnapshot(s.Path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("the pull-request snapshot does not parse: %w", err)
	}
	return nil
}
