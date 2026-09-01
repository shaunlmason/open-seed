// Package lane is the role surface: six lane definitions, each a
// manifest plus an ordered list of prose fragments, resolved by
// concatenation and CHECKED against the tables that already exist
// (SEED-NEXT.md §II.11 and III.J; plans/os-cf1c9688.md).
//
// The point is not that Seed has role documents. It is that their
// claims are decidable. A role file is prose, and nothing checks prose:
// v1 has four role documents and no validator, which is survivable
// where a human reads them and not survivable for a promotion criterion
// that asserts a property OF THE LANE ("runs entirely through Seed
// verbs, orienting from one position-stamped read") that only the file's
// author ever verified.
//
// So the four obligations docs/next-build-plan.md Phase 9 item 1 binds
// are DECLARED FIELDS, not paragraphs, and every field is checked
// against an authority elsewhere in the tree: capabilities against
// internal/keyring, the acts against internal/loopverb, and the
// capability a verb accepts against keyring.AcceptedCapabilities. This
// package holds no policy of its own and invents no legality; a
// hand-written list of capability or verb names here would be the bug
// it exists to prevent.
package lane

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/loopverb"
)

// Manifest is one lane's declaration. The four obligation fields are
// named for what they oblige rather than for what they contain, so a
// fragment author reading the schema meets the obligation, not a
// datatype.
type Manifest struct {
	// Lane is the charter's name for the role (§II.11).
	Lane string `json:"lane"`
	// Summary is the one-line statement of what the lane is for.
	Summary string `json:"summary"`
	// Grants are the capabilities the lane's key holds.
	Grants []string `json:"grants"`
	// OrientsFrom is the SINGLE position-stamped read the lane wakes
	// on, written as the command a lane runs.
	OrientsFrom string `json:"orients_from"`
	// ActsThrough names the loop acts the lane performs, in
	// internal/loopverb's spelling. A lane that acts through the raw
	// append seam declares nothing here and is refused.
	ActsThrough []string `json:"acts_through"`
	// LivenessFrom names the lane's own work steps whose execution
	// emits observations. Every entry must appear in ActsThrough:
	// liveness rides the work, and a step that is not work is a
	// heartbeat by another name.
	LivenessFrom []string `json:"liveness_from"`
	// Inbox is the lane's one-inbox declaration: what wakes it, and
	// what convinces it.
	Inbox string `json:"inbox"`
	// Fragments is the ORDERED list of prose files composing the role,
	// relative to the lanes directory. Order is declared, never
	// inferred from a directory listing, which would change under a
	// rename.
	Fragments []string `json:"fragments"`
}

// Finding is one validation failure, naming the lane, the field, and
// what refused it. A finding that does not name its authority is a
// finding a reader has to take on trust.
type Finding struct {
	Lane    string `json:"lane"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: %s: %s", f.Lane, f.Field, f.Message)
}

// FragmentDir is the subdirectory of prose fragments.
const FragmentDir = "fragments"

// dispatcherAllowlist is the dispatcher's permitted grant set, stated
// POSITIVELY (plans/os-cf1c9688.md D5). A blocklist would have to be
// extended every time a capability is added, and one nobody thought to
// exclude would be admitted by default: that is how `operator` — the
// strongest grant in the keyring — passed the first draft's
// "no authoring, verdict or sealing" check. The dispatcher reads the
// most hostile input in the system, so its posture is the one that
// must be stated rather than inferred.
var dispatcherAllowlist = []string{keyring.CapDispatch}

// situationFlags is the read surface orients_from is checked against.
// It lives here rather than in cmd/seed because package main is not
// importable; cmd/seed carries a drill asserting its own flag set
// matches this exactly, so the two cannot drift without a red test.
func situationFlags() []string {
	return []string{"ledger", "key", "subject", "since"}
}

// SituationFlags exposes that set for the CLI's agreement drill.
func SituationFlags() []string { return situationFlags() }

// Load reads every manifest in dir, in lane-name order.
func Load(dir string) ([]Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Manifest
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var m Manifest
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if m.Lane == "" {
			return nil, fmt.Errorf("%s: manifest names no lane", e.Name())
		}
		if want := m.Lane + ".json"; e.Name() != want {
			return nil, fmt.Errorf("%s: lane %q must live in %s", e.Name(), m.Lane, want)
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Lane < out[j].Lane })
	return out, nil
}

// Resolve concatenates the manifest's fragments IN DECLARED ORDER. It
// is a pure function of the files on disk: no position to stamp, no
// rebuild semantics, nothing written back. A resolved role written to
// disk would be a second copy that can go stale, which is the failure
// the ordered list exists to prevent.
func Resolve(dir string, m Manifest) (string, error) {
	var b strings.Builder
	for _, f := range m.Fragments {
		body, err := os.ReadFile(filepath.Join(dir, FragmentDir, f))
		if err != nil {
			return "", err
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.Write(body)
	}
	return b.String(), nil
}

// Validate runs every check over the loaded set and returns the
// findings, in a stable order. An empty result is the assertion that
// each lane's declarations are answerable by the tables the system
// already enforces.
func Validate(dir string, ms []Manifest) []Finding {
	var out []Finding
	add := func(lane, field, msg string) {
		out = append(out, Finding{Lane: lane, Field: field, Message: msg})
	}
	used := map[string]bool{}

	for _, m := range ms {
		// Grants: real capabilities, asked of the keyring.
		for _, g := range m.Grants {
			if !keyring.Known(g) {
				add(m.Lane, "grants", fmt.Sprintf("%q is not a capability in the keyring vocabulary (%s)",
					g, strings.Join(keyring.Capabilities(), ", ")))
			}
		}

		// The orienting read: one position-stamped read, named as a
		// command, with flags the surface actually takes.
		checkOrientsFrom(m, add)

		// The acts: real loop acts, and grants that INTERSECT what
		// each act's ledger verb accepts. Intersection, not
		// containment: AcceptedCapabilities is an OR-set consumed by
		// HasAnyCapability, so requiring every accepted capability
		// would hand `operator` to every lane that claims or spends
		// and dissolve the separation this check exists to protect.
		for _, name := range m.ActsThrough {
			act, ok := loopverb.ByName(name)
			if !ok {
				add(m.Lane, "acts_through", fmt.Sprintf("%q is not a loop act (%s)",
					name, strings.Join(loopverb.Names(), ", ")))
				continue
			}
			accepted := keyring.AcceptedCapabilities(act.Verb)
			if len(accepted) == 0 {
				continue
			}
			if !intersects(m.Grants, accepted) {
				add(m.Lane, "acts_through", fmt.Sprintf(
					"%s appends %s, which admits for any of {%s}, and this lane grants {%s}",
					name, act.Verb, strings.Join(accepted, ", "), strings.Join(m.Grants, ", ")))
			}
		}

		// Liveness rides the work, checked against whether this lane
		// runs a loop at all. Four of the charter's six do not: a
		// verifier acts through verdict.rendered and a dispatcher
		// through intent.filed, neither of which is a loop act, so
		// requiring loop acts of every lane would force four
		// manifests to claim work they never do.
		//
		// The obligation is therefore CONDITIONAL but not dodgeable.
		// A lane that runs a loop must say where its liveness comes
		// from, and a lane cannot escape that by declaring no acts:
		// holding the claim capability means it claims, and claiming
		// IS a loop act, so the grant it already declares decides
		// whether the obligation applies.
		runsLoop := len(m.ActsThrough) > 0
		if contains(m.Grants, keyring.CapClaim) && !runsLoop {
			add(m.Lane, "acts_through", "grants claim but declares no acts: claiming is a loop act, so a lane "+
				"holding the claim capability acts through the loop verbs and must say which")
		}
		switch {
		case runsLoop && len(m.LivenessFrom) == 0:
			add(m.Lane, "liveness_from", "declares no liveness source: observations ride the loop's own steps, "+
				"and an empty declaration would satisfy the subset rule without meaning anything")
		case !runsLoop && len(m.LivenessFrom) > 0:
			add(m.Lane, "liveness_from", "names liveness sources but no acts: liveness rides the work, so a "+
				"lane that performs no loop act has no work for it to ride")
		}
		for _, step := range m.LivenessFrom {
			if !contains(m.ActsThrough, step) {
				add(m.Lane, "liveness_from", fmt.Sprintf(
					"%q is not among this lane's acts: liveness rides the work, so a liveness source "+
						"that is not a work step is a heartbeat by another name", step))
			}
		}

		if strings.TrimSpace(m.Inbox) == "" {
			add(m.Lane, "inbox", "declares no inbox: push channels wake, position-stamped reads convince, "+
				"and a lane that does not say so has not adopted the doctrine")
		}

		// The dispatcher's posture, as an allowlist.
		if m.Lane == "dispatcher" {
			checkDispatcher(m, add)
		}

		// Fragments: declared, ordered, present.
		if len(m.Fragments) == 0 {
			add(m.Lane, "fragments", "composes from no fragments: a role is its ordered fragments")
		}
		for _, f := range m.Fragments {
			path := filepath.Join(dir, FragmentDir, f)
			body, err := os.ReadFile(path)
			if err != nil {
				add(m.Lane, "fragments", fmt.Sprintf("%s: %v", f, err))
				continue
			}
			used[f] = true
			// The belt to D4's braces: a fragment could instruct an
			// agent to heartbeat without declaring it. This IS a
			// spelling rule and is only ever the second line of
			// defence; the property check above is the argument.
			if line, found := bareHeartbeat(string(body)); found {
				add(m.Lane, "fragments", fmt.Sprintf(
					"%s instructs a bare liveness emit (%q): liveness rides the loop's own steps, "+
						"and the vocabulary carries no verb whose only purpose is to report it", f, line))
			}
		}
	}

	out = append(out, orphanFindings(dir, used)...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Lane != out[j].Lane {
			return out[i].Lane < out[j].Lane
		}
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func checkOrientsFrom(m Manifest, add func(lane, field, msg string)) {
	read := strings.TrimSpace(m.OrientsFrom)
	if read == "" {
		add(m.Lane, "orients_from", "names no orienting read: a lane orients from ONE position-stamped read, "+
			"and promotion criterion 1 is a property of this file rather than of whoever writes the agent")
		return
	}
	if !strings.HasPrefix(read, "seed situation") {
		add(m.Lane, "orients_from", fmt.Sprintf(
			"%q is not the situation read: `seed situation` is the one position-stamped read a lane wakes on", read))
		return
	}
	known := map[string]bool{}
	for _, f := range situationFlags() {
		known[f] = true
	}
	for _, tok := range strings.Fields(read) {
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		name := strings.TrimPrefix(tok, "--")
		if i := strings.Index(name, "="); i >= 0 {
			name = name[:i]
		}
		if !known[name] {
			add(m.Lane, "orients_from", fmt.Sprintf("--%s is not a flag `seed situation` takes (%s)",
				name, "--"+strings.Join(situationFlags(), ", --")))
		}
	}
}

func checkDispatcher(m Manifest, add func(lane, field, msg string)) {
	allowed := map[string]bool{}
	for _, c := range dispatcherAllowlist {
		allowed[c] = true
	}
	for _, g := range m.Grants {
		if !allowed[g] {
			add(m.Lane, "grants", fmt.Sprintf(
				"%q is outside the dispatcher's allowlist {%s}: it touches the most untrusted text in the "+
					"system and runs with least standing capability (SEED-NEXT.md §II.11)",
				g, strings.Join(dispatcherAllowlist, ", ")))
		}
	}
	for _, want := range dispatcherAllowlist {
		if !contains(m.Grants, want) {
			add(m.Lane, "grants", fmt.Sprintf("the dispatcher's allowlist names %q, which this manifest does not grant", want))
		}
	}
}

// orphanFindings names fragments no manifest composes: an unreferenced
// fragment is prose nobody reads, and the ordered lists are the only
// thing that makes a fragment part of a role.
func orphanFindings(dir string, used map[string]bool) []Finding {
	var out []Finding
	root := filepath.Join(dir, FragmentDir)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || used[rel] {
			return nil
		}
		out = append(out, Finding{Lane: "-", Field: "fragments",
			Message: fmt.Sprintf("%s is composed by no lane: an unreferenced fragment is prose nobody reads", rel)})
		return nil
	})
	return out
}

// bareHeartbeat finds a fragment line instructing a standalone
// observation emit.
func bareHeartbeat(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "seed obs emit") {
			return strings.TrimSpace(line), true
		}
	}
	return "", false
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	for _, x := range a {
		if contains(b, x) {
			return true
		}
	}
	return false
}
