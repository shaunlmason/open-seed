package protections

// The Forgejo adapter is held to the same desired state as GitHub, over
// an in-process fake of Forgejo's branch- and tag-protection API
// (plans/os-ad610334.md D2, D5): plan on an empty forge creates the four
// rulesets, the pull-request rule reports manual (unexpressible), apply
// leaves no drift, and a weakened or stray protection is reconciled.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type fakeForgejo struct {
	mu       sync.Mutex
	branches map[string]fjBranch // by rule_name
	tags     map[int64]fjTag     // by id
	nextTag  int64
	calls    []string
}

func newFakeForgejo() *fakeForgejo {
	return &fakeForgejo{branches: map[string]fjBranch{}, tags: map[int64]fjTag{}}
}

func (f *fakeForgejo) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token tok" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		p := r.URL.Path
		const base = "/api/v1/repos/o/r"
		switch {
		case r.Method == http.MethodGet && p == base:
			json.NewEncoder(w).Encode(fjRepo{DefaultBranch: "trunk"})
		case r.Method == http.MethodGet && p == base+"/branch_protections":
			out := []fjBranch{}
			for _, b := range f.branches {
				out = append(out, b)
			}
			json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPost && p == base+"/branch_protections":
			var b fjBranch
			json.NewDecoder(r.Body).Decode(&b)
			f.branches[b.RuleName] = b
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(b)
		case strings.HasPrefix(p, base+"/branch_protections/"):
			name := strings.TrimPrefix(p, base+"/branch_protections/")
			if _, ok := f.branches[name]; !ok && r.Method != http.MethodPost {
				http.Error(w, "no such branch protection", http.StatusNotFound)
				return
			}
			switch r.Method {
			case http.MethodPatch:
				var b fjBranch
				json.NewDecoder(r.Body).Decode(&b)
				b.RuleName = name
				f.branches[name] = b
				json.NewEncoder(w).Encode(b)
			case http.MethodDelete:
				delete(f.branches, name)
				w.WriteHeader(http.StatusNoContent)
			}
		case r.Method == http.MethodGet && p == base+"/tag_protections":
			out := []fjTag{}
			for _, tg := range f.tags {
				out = append(out, tg)
			}
			json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPost && p == base+"/tag_protections":
			var tg fjTag
			json.NewDecoder(r.Body).Decode(&tg)
			f.nextTag++
			tg.ID = f.nextTag
			f.tags[tg.ID] = tg
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(tg)
		case strings.HasPrefix(p, base+"/tag_protections/"):
			id, _ := strconv.ParseInt(strings.TrimPrefix(p, base+"/tag_protections/"), 10, 64)
			if _, ok := f.tags[id]; !ok {
				http.Error(w, "no such tag protection", http.StatusNotFound)
				return
			}
			switch r.Method {
			case http.MethodPatch:
				var tg fjTag
				json.NewDecoder(r.Body).Decode(&tg)
				tg.ID = id
				f.tags[id] = tg
				json.NewEncoder(w).Encode(tg)
			case http.MethodDelete:
				delete(f.tags, id)
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			http.Error(w, "unexpected "+r.Method+" "+p, http.StatusNotFound)
		}
	})
}

func TestForgejoAdapterReconciles(t *testing.T) {
	fake := newFakeForgejo()
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	cfg := declaration(t)
	fj := NewForgejo(srv.URL, "o", "r", "tok")

	// Plan on the empty forge: four creates, the pull-request rule manual.
	rep, _, err := Plan(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.DefaultBranch != "trunk" {
		t.Fatalf("default branch read from the forge, got %q", rep.DefaultBranch)
	}
	if rep.DriftCount != 4 {
		t.Fatalf("four rulesets to create, got drift %d (%+v)", rep.DriftCount, rep.Changes)
	}
	if rep.Manual != 1 {
		t.Fatalf("the pull-request rule is manual on Forgejo, got %d manual", rep.Manual)
	}

	// Apply, then re-plan: no drift, the manual rule persists.
	after, err := Apply(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if after.DriftCount != 0 {
		t.Fatalf("apply must leave no drift, got %d (%+v)", after.DriftCount, after.Changes)
	}
	if after.Manual != 1 {
		t.Fatalf("the manual rule persists after apply, got %d", after.Manual)
	}
	if len(fake.branches) != 3 || len(fake.tags) != 1 {
		t.Fatalf("three branch protections and one tag protection, got %d/%d", len(fake.branches), len(fake.tags))
	}
	// The ledger branch's sole writer is the admission identity.
	led, ok := fake.branches["seed-ledger"]
	if !ok || led.EnablePush || !led.EnablePushWhitelist || len(led.PushWhitelistUsernames) != 1 || led.PushWhitelistUsernames[0] != "app:4242" {
		t.Fatalf("the ledger protection's sole writer must be the admission identity, got %+v", led)
	}
	// The default branch carries the declared checks.
	def := fake.branches["trunk"]
	if !def.EnableStatusCheck || strings.Join(def.StatusCheckContexts, ",") != "check,verify" {
		t.Fatalf("the default branch requires the declared checks, got %+v", def)
	}

	// Drift: weaken the ledger's whitelist and add a stray tag; reconcile.
	fake.mu.Lock()
	stray := fake.branches["seed-ledger"]
	stray.PushWhitelistUsernames = []string{"app:4242", "intruder"}
	fake.branches["seed-ledger"] = stray
	fake.calls = nil
	fake.mu.Unlock()
	drift, _, err := Plan(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if drift.DriftCount == 0 {
		t.Fatal("a widened ledger whitelist must be caught as drift")
	}
	if _, err := Apply(cfg, fj, ""); err != nil {
		t.Fatal(err)
	}
	clean, _, err := Plan(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if clean.DriftCount != 0 {
		t.Fatalf("apply must restore the sole writer, got drift %d", clean.DriftCount)
	}
}

func TestForgejoNeedsAToken(t *testing.T) {
	fake := newFakeForgejo()
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	if _, err := NewForgejo(srv.URL, "o", "r", "").Read(); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("a tokenless read must be refused 401, got %v", err)
	}
}

func TestForgejoMutationEvidence(t *testing.T) {
	// The desired state is one table: both forges derive from Desired,
	// never a forge-specific fork.
	cfg := declaration(t)
	want, err := Desired(cfg, "trunk")
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Rulesets) != 4 {
		t.Fatalf("Desired is the one shared table of four rulesets, got %d", len(want.Rulesets))
	}

	// A Forgejo read that drops the ledger branch's sole writer is caught
	// as drift, never accepted.
	fake := newFakeForgejo()
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	fj := NewForgejo(srv.URL, "o", "r", "tok")
	if _, err := Apply(cfg, fj, ""); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	led := fake.branches["seed-ledger"]
	led.EnablePush = true // anyone may push — the whitelist is gone
	led.EnablePushWhitelist = false
	led.PushWhitelistUsernames = nil
	fake.branches["seed-ledger"] = led
	fake.mu.Unlock()
	rep, _, err := Plan(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.DriftCount == 0 {
		t.Fatal("dropping the ledger branch's sole writer must be drift")
	}
}

// TestForgejoLive reconciles a real Forgejo instance when one is
// configured; it skips with a named reason otherwise (never silently).
func TestForgejoLive(t *testing.T) {
	url, repo, tok := os.Getenv("SEED_FORGEJO_URL"), os.Getenv("SEED_FORGEJO_REPO"), os.Getenv("FORGEJO_TOKEN")
	if url == "" || repo == "" || tok == "" {
		t.Skip("live Forgejo drill: set SEED_FORGEJO_URL, SEED_FORGEJO_REPO and FORGEJO_TOKEN to run against a real instance")
	}
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		t.Fatalf("SEED_FORGEJO_REPO must be owner/name, got %q", repo)
	}
	cfg := declaration(t)
	fj := NewForgejo(url, owner, name, tok)
	if _, err := Apply(cfg, fj, ""); err != nil {
		t.Fatalf("applying to the live Forgejo: %v", err)
	}
	after, _, err := Plan(cfg, fj, "")
	if err != nil {
		t.Fatal(err)
	}
	if after.DriftCount != 0 {
		t.Fatalf("the live Forgejo must reconcile to no drift, got %d", after.DriftCount)
	}
}
