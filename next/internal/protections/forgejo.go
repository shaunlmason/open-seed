package protections

// Forgejo is the second forge adapter (plans/os-ad610334.md D1, D2):
// the same Forge interface and the same Desired decision table as
// GitHub, over Forgejo's (Gitea-compatible) REST API. Forgejo splits
// protection across branch protections and tag protections rather than
// GitHub's one ruleset model, and it keys a branch protection by its
// glob (rule_name) with no separate name, so Read maps globs back to the
// four ruleset names and remembers them for later update and delete.
//
// What Forgejo cannot express — the pull-request requirement's
// conversation-thread-resolution and code-owner-review gates — is named
// in State.Unexpressible, so Diff reports it manual rather than dropping
// it. The reconciler reconciles by rule type, so the whole pull-request
// rule is manual: applying approvals while reading back as compliant on
// the gates Forgejo does not enforce would be a false compliance the
// mutation drill forbids (D2, reconciled).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Forgejo talks to a Forgejo/Gitea instance's REST API (/api/v1). Base
// is the instance URL; Token authenticates as the admission identity.
type Forgejo struct {
	Base  string
	Owner string
	Repo  string
	Token string
	HTTP  *http.Client
	// LedgerBranch is the branch the ledger rides (from the declaration's
	// admission.ledger_ref); Read maps its protection back to the ledger
	// ruleset. Empty falls back to the default (seed-ledger), so the
	// convention still works when the caller does not set it.
	LedgerBranch string

	branchRule map[string]string // ruleset name -> Forgejo rule_name, from Read
	tagID      map[string]int64  // ruleset name -> tag-protection id, from Read
	defBranch  string            // the repository's default branch, from Read
}

// NewForgejo constructs the adapter with a bounded-timeout client.
func NewForgejo(base, owner, repo, token string) *Forgejo {
	return &Forgejo{Base: strings.TrimRight(base, "/"), Owner: owner, Repo: repo, Token: token, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (f *Forgejo) client() *http.Client {
	if f.HTTP != nil {
		return f.HTTP
	}
	return http.DefaultClient
}

func (f *Forgejo) repoPath() string { return "/api/v1/repos/" + f.Owner + "/" + f.Repo }

func (f *Forgejo) do(method, path string, body, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, f.Base+path, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if f.Token != "" {
		req.Header.Set("Authorization", "token "+f.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("forgejo %s %s: %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("forgejo %s %s: decoding response: %w", method, path, err)
		}
	}
	return resp.StatusCode, nil
}

const forgejoLedgerBranch = "seed-ledger" // DefaultLedgerRef's branch; the ledger protection's glob
const releaseTagPattern = "v*"

type fjRepo struct {
	DefaultBranch string `json:"default_branch"`
}

// fjBranch is a Gitea branch protection. rule_name is the glob it
// matches; a protected branch inherently blocks force-push and deletion,
// so RuleNonFastForward and RuleDeletion ride the protection's existence.
type fjBranch struct {
	RuleName               string   `json:"rule_name"`
	EnablePush             bool     `json:"enable_push"`
	EnablePushWhitelist    bool     `json:"enable_push_whitelist"`
	PushWhitelistUsernames []string `json:"push_whitelist_usernames"`
	EnableStatusCheck      bool     `json:"enable_status_check"`
	StatusCheckContexts    []string `json:"status_check_contexts"`
	RequiredApprovals      int64    `json:"required_approvals"`
}

// fjTag is a Gitea tag protection: a glob and the usernames that may
// create, update or delete matching tags.
type fjTag struct {
	ID                 int64    `json:"id,omitempty"`
	NamePattern        string   `json:"name_pattern"`
	WhitelistUsernames []string `json:"whitelist_usernames"`
}

// forgejoUnexpressible names the rule types Forgejo's branch protection
// cannot state in full: the pull-request requirement (no
// conversation-resolution or code-owner-review gate).
var forgejoUnexpressible = []string{RulePullRequest}

// Read maps Forgejo's branch and tag protections into the four named
// rulesets, remembering each protection's key for a later Apply.
func (f *Forgejo) Read() (*State, error) {
	f.branchRule = map[string]string{}
	f.tagID = map[string]int64{}

	var repo fjRepo
	if _, err := f.do(http.MethodGet, f.repoPath(), nil, &repo); err != nil {
		return nil, err
	}
	f.defBranch = repo.DefaultBranch
	st := &State{DefaultBranch: repo.DefaultBranch, Rulesets: map[string]Ruleset{}, Unexpressible: append([]string(nil), forgejoUnexpressible...)}

	var branches []fjBranch
	if _, err := f.do(http.MethodGet, f.repoPath()+"/branch_protections", nil, &branches); err != nil {
		return nil, err
	}
	for _, b := range branches {
		name := f.rulesetFor(b.RuleName)
		if name == "" {
			continue
		}
		f.branchRule[name] = b.RuleName
		st.Rulesets[name] = branchToRuleset(name, b)
	}

	var tags []fjTag
	if _, err := f.do(http.MethodGet, f.repoPath()+"/tag_protections", nil, &tags); err != nil {
		return nil, err
	}
	for _, t := range tags {
		if t.NamePattern != releaseTagPattern {
			continue
		}
		f.tagID[RulesetTags] = t.ID
		st.Rulesets[RulesetTags] = Ruleset{
			Name: RulesetTags, Target: TargetTag, Refs: []string{"refs/tags/" + t.NamePattern},
			Rules:  []Rule{{Type: RuleDeletion}, {Type: RuleNonFastForward}, {Type: RuleUpdate}},
			Bypass: append([]string(nil), t.WhitelistUsernames...),
		}
	}
	return st, nil
}

// rulesetFor maps a branch-protection glob back to the seed ruleset it
// stands for. The default branch is read from the repo; the contract
// glob and the ledger branch are conventional (seed/* and seed-ledger).
func (f *Forgejo) rulesetFor(glob string) string {
	ledger := f.LedgerBranch
	if ledger == "" {
		ledger = forgejoLedgerBranch
	}
	switch glob {
	case "seed/*":
		return RulesetContracts
	case ledger:
		return RulesetLedger
	case f.defBranch:
		return RulesetDefault
	default:
		return ""
	}
}

// branchToRuleset maps a Forgejo branch protection into a Ruleset. The
// protection's existence carries deletion and non-fast-forward; the push
// whitelist carries update (the sole writer); status checks map their
// contexts. The pull-request rule is not reconstructed — it is
// unexpressible, so Diff strips it from the desired ruleset and reports
// it manual.
func branchToRuleset(name string, b fjBranch) Ruleset {
	rs := Ruleset{Name: name, Target: TargetBranch, Refs: []string{"refs/heads/" + b.RuleName}, Rules: []Rule{{Type: RuleDeletion}, {Type: RuleNonFastForward}}}
	if !b.EnablePush && b.EnablePushWhitelist {
		rs.Rules = append(rs.Rules, Rule{Type: RuleUpdate})
		rs.Bypass = append([]string(nil), b.PushWhitelistUsernames...)
	}
	if b.EnableStatusCheck && len(b.StatusCheckContexts) > 0 {
		checks := append([]string(nil), b.StatusCheckContexts...)
		sort.Strings(checks)
		rs.Rules = append(rs.Rules, Rule{Type: RuleStatusChecks, Params: map[string]any{"contexts": checks}})
	}
	return rs
}

// glob strips the refs/heads/ prefix from a ruleset's first ref to get
// the Forgejo rule_name (a branch glob).
func glob(rs Ruleset) string {
	if len(rs.Refs) == 0 {
		return rs.Name
	}
	return strings.TrimPrefix(rs.Refs[0], "refs/heads/")
}

// Apply reconciles Forgejo's protections to the desired state.
func (f *Forgejo) Apply(changes []Change, desired *State) error {
	if f.branchRule == nil {
		if _, err := f.Read(); err != nil {
			return err
		}
	}
	for _, c := range changes {
		switch c.Kind {
		case ChangeManual:
			continue
		case ChangeCreate, ChangeUpdate:
			rs, ok := desired.Rulesets[c.Ruleset]
			if !ok {
				return fmt.Errorf("no desired ruleset %q to apply", c.Ruleset)
			}
			if rs.Target == TargetTag {
				if err := f.putTag(c.Kind, rs); err != nil {
					return err
				}
				continue
			}
			if err := f.putBranch(c.Kind, rs); err != nil {
				return err
			}
		case ChangeDelete:
			if err := f.remove(c.Ruleset); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Forgejo) putBranch(kind string, rs Ruleset) error {
	body := toForgejoBranch(rs)
	if kind == ChangeCreate {
		var created fjBranch
		if _, err := f.do(http.MethodPost, f.repoPath()+"/branch_protections", body, &created); err != nil {
			return err
		}
		f.branchRule[rs.Name] = body.RuleName
		return nil
	}
	rule, ok := f.branchRule[rs.Name]
	if !ok {
		rule = body.RuleName
	}
	_, err := f.do(http.MethodPatch, f.repoPath()+"/branch_protections/"+rule, body, nil)
	return err
}

func (f *Forgejo) putTag(kind string, rs Ruleset) error {
	body := fjTag{NamePattern: releaseTagPattern, WhitelistUsernames: append([]string(nil), rs.Bypass...)}
	if kind == ChangeCreate {
		var created fjTag
		if _, err := f.do(http.MethodPost, f.repoPath()+"/tag_protections", body, &created); err != nil {
			return err
		}
		f.tagID[RulesetTags] = created.ID
		return nil
	}
	id, ok := f.tagID[RulesetTags]
	if !ok {
		return fmt.Errorf("no release-tag protection id to update")
	}
	_, err := f.do(http.MethodPatch, fmt.Sprintf("%s/tag_protections/%d", f.repoPath(), id), body, nil)
	return err
}

func (f *Forgejo) remove(name string) error {
	if name == RulesetTags {
		id, ok := f.tagID[RulesetTags]
		if !ok {
			return fmt.Errorf("no release-tag protection id to delete")
		}
		_, err := f.do(http.MethodDelete, fmt.Sprintf("%s/tag_protections/%d", f.repoPath(), id), nil, nil)
		delete(f.tagID, name)
		return err
	}
	rule, ok := f.branchRule[name]
	if !ok {
		return fmt.Errorf("no branch protection %q to delete", name)
	}
	_, err := f.do(http.MethodDelete, f.repoPath()+"/branch_protections/"+rule, nil, nil)
	delete(f.branchRule, name)
	return err
}

// toForgejoBranch maps a desired Ruleset into a Forgejo branch
// protection. rule_name is the branch glob; update becomes the push
// whitelist; approvals and status checks map their fields (approvals
// best-effort, since the pull-request rule is otherwise manual).
func toForgejoBranch(rs Ruleset) fjBranch {
	b := fjBranch{RuleName: glob(rs), EnablePush: true}
	for _, r := range rs.Rules {
		switch r.Type {
		case RuleUpdate:
			b.EnablePush = false
			b.EnablePushWhitelist = true
			b.PushWhitelistUsernames = append([]string(nil), rs.Bypass...)
		case RulePullRequest:
			if n, ok := ruleInt(r.Params, "required_approving_review_count"); ok {
				b.RequiredApprovals = int64(n)
			}
		case RuleStatusChecks:
			if cs, ok := ruleStrings(r.Params, "contexts"); ok {
				b.EnableStatusCheck = true
				b.StatusCheckContexts = cs
			}
		}
	}
	return b
}

func ruleInt(p map[string]any, key string) (int, bool) {
	switch v := p[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n, true
		}
	}
	return 0, false
}

func ruleStrings(p map[string]any, key string) ([]string, bool) {
	switch v := p[key].(type) {
	case []string:
		out := append([]string(nil), v...)
		sort.Strings(out)
		return out, len(out) > 0
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out, len(out) > 0
	}
	return nil, false
}
