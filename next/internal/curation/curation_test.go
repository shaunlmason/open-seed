package curation

// The curation package's own drills (plans/os-f30ee0d3.md AC6, step
// 8): the shapes, the id derivation, the fold, anomalies, unbound, and
// the lessons store's frontmatter lint over every file it holds.

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func testKey(first byte) ed25519.PrivateKey {
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = first
	return ed25519.NewKeyFromSeed(seed)
}

func signed(t *testing.T, key ed25519.PrivateKey, v, verb, subject, payload string) *event.Record {
	t.Helper()
	fp, err := event.Fingerprint(key.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := event.Sign(event.Event{V: v, TS: "2026-09-02T00:00:00Z", Actor: fp, Verb: verb, Subject: subject,
		Payload: json.RawMessage(payload), Prev: strings.Repeat("0", 64)}, key)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestHypothesisIDIsDerivedFromTheCanonicalClaim(t *testing.T) {
	a := HypothesisID("retry the fetch once")
	if !IsHypothesisSubject(a) {
		t.Fatalf("the id has the h-<12 hex> shape: %s", a)
	}
	if b := HypothesisID("  retry   the fetch\nonce "); b != a {
		t.Fatalf("whitespace does not change the claim: %s vs %s", a, b)
	}
	if c := HypothesisID("retry the fetch twice"); c == a {
		t.Fatal("a different claim derives a different subject")
	}
	if IsHypothesisSubject("c-1") || IsHypothesisSubject("h-xyz") {
		t.Fatal("only the derived shape is a hypothesis subject")
	}
}

func TestShapesRefuseNamingThePart(t *testing.T) {
	good := `{"fence": "3", "tried": "x", "outcome": "y", "condition": "z", "environment": "w", "pointer": "a/b.md @ 0123456"}`
	if _, err := ParseDeadEnd("c-1", []byte(good)); err != nil {
		t.Fatalf("a complete dead end parses: %v", err)
	}
	for name, body := range map[string]string{
		"missing":  `{"fence": "3", "tried": "x", "outcome": "y", "condition": "", "environment": "w"}`,
		"unknown":  `{"fence": "3", "tried": "x", "outcome": "y", "condition": "z", "environment": "w", "conclusion": "so"}`,
		"bare":     `{"fence": "3", "tried": "x", "outcome": "y", "condition": "z", "environment": "w", "pointer": "a/b.md"}`,
		"fence":    `{"fence": "three", "tried": "x", "outcome": "y", "condition": "z", "environment": "w"}`,
		"not-json": `nope`,
		"trailing": good + ` {}`,
	} {
		if _, err := ParseDeadEnd("c-1", []byte(body)); err == nil {
			t.Errorf("%s: a malformed dead end refuses", name)
		}
	}
	claim := "retry once"
	id := HypothesisID(claim)
	h := fmt.Sprintf(`{"claim": %q, "applies_when": "flaky", "support": ["c-1@4", "c-2@9"], "exceptions": [], "provenance": ["plans/x.md @ 0123456"]}`, claim)
	if _, err := ParseHypothesis(id, []byte(h)); err != nil {
		t.Fatalf("a well-formed proposal parses: %v", err)
	}
	if _, err := ParseHypothesis("h-000000000000", []byte(h)); err == nil || !strings.Contains(err.Error(), id) {
		t.Fatalf("a subject not derived from the claim refuses naming the derived one: %v", err)
	}
	for name, body := range map[string]string{
		"citation":   fmt.Sprintf(`{"claim": %q, "applies_when": "f", "support": ["c-1"], "exceptions": [], "provenance": []}`, claim),
		"provenance": fmt.Sprintf(`{"claim": %q, "applies_when": "f", "support": [], "exceptions": [], "provenance": ["plans/x.md"]}`, claim),
		"no-support": fmt.Sprintf(`{"claim": %q, "applies_when": "f", "exceptions": [], "provenance": []}`, claim),
		"empty":      fmt.Sprintf(`{"claim": %q, "applies_when": " ", "support": [], "exceptions": [], "provenance": []}`, claim),
	} {
		if _, err := ParseHypothesis(id, []byte(body)); err == nil {
			t.Errorf("%s: a malformed proposal refuses", name)
		}
	}
	l := fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "%s@7", "pr": "pr/3 @ 0123456"}`, LessonsDir, id)
	if _, err := ParseLesson(id, []byte(l)); err != nil {
		t.Fatalf("a well-formed promotion parses: %v", err)
	}
	for name, body := range map[string]string{
		"elsewhere": fmt.Sprintf(`{"lesson": "docs/x.md @ 0123456", "hypothesis": "%s@7", "pr": "pr/3 @ 0123456"}`, id),
		"bare":      fmt.Sprintf(`{"lesson": "%s/x.md", "hypothesis": "%s@7", "pr": "pr/3 @ 0123456"}`, LessonsDir, id),
		"mismatch":  fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "h-000000000000@7", "pr": "pr/3 @ 0123456"}`, LessonsDir),
		"contract":  fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "c-1@7", "pr": "pr/3 @ 0123456"}`, LessonsDir),
		"pr":        fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "%s@7", "pr": "3"}`, LessonsDir, id),
	} {
		if _, err := ParseLesson(id, []byte(body)); err == nil {
			t.Errorf("%s: a malformed promotion refuses", name)
		}
	}
	if c, ok := ParseCitation("c-1@12"); !ok || c.Contract != "c-1" || c.Position != 12 {
		t.Fatalf("a citation splits at the last @: %+v %v", c, ok)
	}
	for _, bad := range []string{"c-1", "@3", "c-1@", "c-1@-1", "c-1@x"} {
		if _, ok := ParseCitation(bad); ok {
			t.Errorf("%q is not a citation", bad)
		}
	}
}

// The fold renders the dead ends, counts a malformed raw fact as an
// anomaly, and on a bare chain (no keyring, so no proposal can have
// passed the boundary) counts every proposal an anomaly and folds
// every promotion unbound: a hypothesis folds only once admission
// re-judges it (review finding on the task PR), and the admitted fold
// is drilled where a boundary stands, in internal/admit. Facts before
// the keyring boundary and lifecycle verbs alike are ignored.
func TestFoldRendersStagesAndCountsAnomalies(t *testing.T) {
	holder, curator, observer := testKey(1), testKey(2), testKey(3)
	claim := "retry once"
	id := HypothesisID(claim)
	dead := `{"fence": "3", "tried": "x", "outcome": "y", "condition": "z", "environment": "w"}`
	hyp := fmt.Sprintf(`{"claim": %q, "applies_when": "flaky", "support": ["c-1@0", "c-2@1"], "exceptions": [], "provenance": []}`, claim)
	records := []*event.Record{
		signed(t, holder, version.Seed1, DeadEndVerb, "c-1", dead),
		signed(t, holder, version.Seed1, DeadEndVerb, "c-2", dead),
		signed(t, holder, version.Seed1, DeadEndVerb, "c-2", `{"fence": "3"}`),
		signed(t, holder, version.Protocol, DeadEndVerb, "c-3", dead),
		signed(t, curator, version.Seed1, HypothesisVerb, id, hyp),
		signed(t, curator, version.Seed1, HypothesisVerb, id, hyp),
		signed(t, holder, version.Seed1, "claim.taken", "c-1", `{}`),
		signed(t, observer, version.Seed1, LessonVerb, "h-000000000000",
			fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "h-000000000000@9", "pr": "pr/1 @ 0123456"}`, LessonsDir)),
		signed(t, observer, version.Seed1, LessonVerb, id,
			fmt.Sprintf(`{"lesson": "%s/x.md @ 0123456", "hypothesis": "%s@4", "pr": "pr/1 @ 0123456"}`, LessonsDir, id)),
	}
	st := Fold(records)
	if len(st.DeadEnds["c-1"]) != 1 || len(st.DeadEnds["c-2"]) != 1 || len(st.DeadEnds["c-3"]) != 0 {
		t.Fatalf("dead ends fold by contract, malformed and pre-boundary ones excluded: %+v", st.DeadEnds)
	}
	if st.DeadEnds["c-1"][0].Actor == "" || st.DeadEnds["c-1"][0].Pos != 0 || st.DeadEnds["c-1"][0].Condition != "z" {
		t.Fatalf("a dead end carries its actor, position and fields: %+v", st.DeadEnds["c-1"][0])
	}
	if _, ok := st.Hypothesis(id); ok || len(st.HypothesisIDs()) != 0 {
		t.Fatalf("no proposal passed a boundary, so none folds as a hypothesis: %v", st.HypothesisIDs())
	}
	if len(st.Lessons) != 0 || len(st.Unbound) != 2 || st.Unbound[0].Pos != 7 || st.Unbound[1].Pos != 8 {
		t.Fatalf("a promotion citing no admitted hypothesis is unbound, never a lesson: %+v %+v", st.Lessons, st.Unbound)
	}
	// One malformed dead end and two proposals that passed no boundary.
	if st.Anomalies != 3 {
		t.Fatalf("malformed and unadmitted raw facts count anomalies, %d", st.Anomalies)
	}
	if !st.Any() || Fold(nil).Any() {
		t.Fatal("Any reports whether the prefix carries a curation fact")
	}
}

// A promotion names a file inside the validated store and nothing
// else: an anchor whose path climbs out of it matches the anchor
// grammar and must still refuse (review finding on the task PR).
func TestLessonAnchorsStayUnderTheStore(t *testing.T) {
	for anchor, want := range map[string]bool{
		LessonsDir + "/retry.md @ 0123456":         true,
		LessonsDir + "/nested/retry.md @ 0123456":  true,
		LessonsDir + "/../../secrets.md @ 0123456": false,
		LessonsDir + "/a/../../../x.md @ 0123456":  false,
		LessonsDir + "/./retry.md @ 0123456":       false,
		LessonsDir + "//retry.md @ 0123456":        false,
		LessonsDir + " @ 0123456":                  false,
		LessonsDir + "/ @ 0123456":                 false,
		"/" + LessonsDir + "/retry.md @ 0123456":   false,
		LessonsDir + "x/retry.md @ 0123456":        false,
		"next/knowledge/other/retry.md @ 0123456":  false,
		"../" + LessonsDir + "/retry.md @ 0123456": false,
	} {
		if got := UnderLessonsDir(anchor); got != want {
			t.Errorf("UnderLessonsDir(%q) = %v, want %v", anchor, got, want)
		}
	}
	id := HypothesisID("retry once")
	body := fmt.Sprintf(`{"lesson": "%s/../../x.md @ 0123456", "hypothesis": "%s@4", "pr": "pr/1 @ 0123456"}`, LessonsDir, id)
	var se *ShapeError
	if _, err := ParseLesson(id, []byte(body)); !errors.As(err, &se) || !strings.Contains(err.Error(), "climbs out") {
		t.Fatalf("a traversal path refuses as a shape refusal naming the store: %v", err)
	}
}

func TestLintRequiresTheFrontmatterKeys(t *testing.T) {
	good := "---\nhypothesis: h-0123456789ab@4\napplies-when: flaky\nsupport: c-1@4, c-2@9\nprovenance: plans/x.md @ 0123456\nlast-validated: 2026-09-02\nexpires: 2026-12-02\n---\n\n# Lesson\n"
	if err := Lint([]byte(good)); err != nil {
		t.Fatalf("a complete frontmatter lints: %v", err)
	}
	for name, body := range map[string]string{
		"no-block": "# Lesson\n",
		"unclosed": "---\nhypothesis: x\n",
		"missing":  "---\nhypothesis: h-0123456789ab@4\napplies-when: flaky\n---\n",
		"empty":    strings.Replace(good, "expires: 2026-12-02", "expires:", 1),
	} {
		if err := Lint([]byte(body)); err == nil {
			t.Errorf("%s: an incomplete frontmatter refuses", name)
		}
	}
}

// Every file in the store honors the contract its README states.
func TestEveryLessonInTheStoreLints(t *testing.T) {
	dir := filepath.Join("..", "..", "knowledge", "lessons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	readme := false
	for _, e := range entries {
		if e.Name() == "README.md" {
			readme = true
			continue
		}
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := Lint(b); err != nil {
			t.Errorf("%s: %v", e.Name(), err)
		}
	}
	if !readme {
		t.Fatal("the store's README states the frontmatter contract")
	}
}
