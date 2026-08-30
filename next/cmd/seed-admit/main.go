// seed-admit is the enforced posture's pre-receive hook
// (docs/next-build-plan.md Phase 2 item 3; plans/os-d3591e09.md): with it
// installed, the validator is the ledger ref's sole writer. It is
// stateless by construction — every decision is rebuilt from the
// repository it guards — and it imports the same admission rule set the
// cooperative client runs (internal/admit), so postures differ in where
// the rules run, never in which rules run.
//
// Division of labor per pushed range: one full VerifyFromGenesis over
// the pushed stream proves what verification owns for every record
// (parse, linkage, signatures, actor resolution, version discipline,
// upgrade schemas), and the records beyond the previously admitted tip
// then pass the admission-only rules that verification deliberately
// tolerates in history: the halt gate, halt verb shapes, and payload
// classification. A record-level prefix check pins append-only-ness:
// commit-graph fast-forward alone would still allow a descendant commit
// whose tree rewrites admitted records.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/halt"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

const (
	defaultRef = "refs/seed/ledger"
	zeroID     = "0000000000000000000000000000000000000000"
)

func main() {
	guarded := os.Getenv("SEED_ADMIT_REF")
	if guarded == "" {
		guarded = defaultRef
	}
	os.Exit(run(os.Stdin, os.Stderr, ".", guarded))
}

// run processes the pre-receive update lines. Any refusal fails the
// whole push atomically (git applies no ref updates when the hook exits
// non-zero), which is exactly the boundary the charter wants.
func run(stdin io.Reader, stderr io.Writer, gitDir, guarded string) int {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 1<<16), 1<<20)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			fmt.Fprintf(stderr, "seed-admit: malformed update line %q\n", scanner.Text())
			return 1
		}
		oldID, newID, ref := fields[0], fields[1], fields[2]
		if ref != guarded {
			continue
		}
		if err := admitUpdate(gitDir, oldID, newID); err != nil {
			fmt.Fprintf(stderr, "seed-admit: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "seed-admit: reading updates: %v\n", err)
		return 1
	}
	return 0
}

func admitUpdate(gitDir, oldID, newID string) error {
	if newID == zeroID {
		return fmt.Errorf("rule ref: deletion of the ledger ref is refused")
	}
	if oldID != zeroID {
		if err := exec.Command("git", "--git-dir", gitDir, "merge-base", "--is-ancestor", oldID, newID).Run(); err != nil {
			return fmt.Errorf("rule ref: non-fast-forward update is refused (admitted history is append-only)")
		}
	}
	if err := validateTreeShape(gitDir, newID); err != nil {
		return err
	}

	newStore, cleanupNew, err := materialize(gitDir, newID)
	if err != nil {
		return err
	}
	defer cleanupNew()

	// One verified replay proves the whole pushed stream and hands the
	// records to the admission pass.
	resolve, _, err := genesis.Bootstrap(newStore)
	if err != nil {
		return fmt.Errorf("rule verify: %v", err)
	}
	var records []*event.Record
	if _, err := newStore.VerifyFromGenesis(resolve, ledger.WithObserver(func(pos int, rec *event.Record) {
		records = append(records, rec)
	})); err != nil {
		return fmt.Errorf("rule verify: %v", err)
	}

	// The previously admitted range is a strict record prefix: same
	// count boundary, same tip hash (hash-chain equality makes the whole
	// prefix identical).
	oldCount := 0
	if oldID != zeroID {
		oldStore, cleanupOld, err := materialize(gitDir, oldID)
		if err != nil {
			return err
		}
		defer cleanupOld()
		oldTip, n, err := oldStore.Tip()
		if err != nil {
			return fmt.Errorf("rule ref: previously admitted stream unreadable: %v", err)
		}
		oldCount = n
		if oldCount > len(records) {
			return fmt.Errorf("rule ref: pushed stream drops admitted records (%d < %d)", len(records), oldCount)
		}
		if oldCount > 0 {
			h, err := records[oldCount-1].Event.Hash()
			if err != nil {
				return fmt.Errorf("rule ref: %v", err)
			}
			if h != oldTip {
				return fmt.Errorf("rule ref: pushed stream rewrites admitted history at or before position %d", oldCount-1)
			}
		}
	}

	// Admission-only rules for the new records: what a full verification
	// deliberately tolerates in history is exactly what the boundary
	// refuses in new events.
	rules := admissionRules(admit.Default())
	prev := event.EmptyHash
	if oldCount > 0 {
		h, err := records[oldCount-1].Event.Hash()
		if err != nil {
			return fmt.Errorf("rule ref: %v", err)
		}
		prev = h
	}
	for i := oldCount; i < len(records); i++ {
		ctx := &admit.Context{
			Count:   i,
			Tip:     prev,
			Halt:    halt.StateAt(records[:i]),
			Resolve: resolve,
		}
		if err := admit.Run(ctx, records[i], rules); err != nil {
			return fmt.Errorf("position %d: %v", i, err)
		}
		h, err := records[i].Event.Hash()
		if err != nil {
			return fmt.Errorf("rule ref: %v", err)
		}
		prev = h
	}
	return nil
}

// admissionRules selects the boundary's share of the one admission set
// by exclusion, not inclusion: only the checks the completed replay has
// demonstrably proved for every pushed record (actor resolution and
// signature; version discipline) are dropped, so rules future phases
// append to admit.Default() flow through to the server boundary
// automatically instead of being silently bypassed (#94 review).
func admissionRules(all []admit.Rule) []admit.Rule {
	replayOwned := map[string]bool{"actor": true, "version": true}
	var rules []admit.Rule
	for _, r := range all {
		if !replayOwned[r.Name] {
			rules = append(rules, r)
		}
	}
	return rules
}

// validateTreeShape refuses any path on the guarded ref outside the
// ledger layout (HEAD and top-level segments/*.jsonl): the charter's
// references-not-content boundary applies to the tree itself, or a
// fast-forward push could ride arbitrary content on the authoritative
// ref beside an unchanged record stream (#94 review).
func validateTreeShape(gitDir, commit string) error {
	out, err := exec.Command("git", "--git-dir", gitDir, "ls-tree", "-r", "--name-only", commit).Output()
	if err != nil {
		return fmt.Errorf("rule ref: cannot list pushed tree %.12s: %v", commit, err)
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" || name == "HEAD" {
			continue
		}
		rest, ok := strings.CutPrefix(name, "segments/")
		if ok && strings.HasSuffix(rest, ".jsonl") && !strings.Contains(rest, "/") {
			continue
		}
		return fmt.Errorf("rule tree: %q is outside the ledger layout (only HEAD and segments/*.jsonl ride the guarded ref)", name)
	}
	return nil
}

// materialize extracts a commit's tree into a temp dir and opens it as a
// read-only ledger store.
func materialize(gitDir, commit string) (*ledger.Store, func(), error) {
	dir, err := os.MkdirTemp("", "seed-admit-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	archive := exec.Command("git", "--git-dir", gitDir, "archive", commit)
	untar := exec.Command("tar", "-x", "-C", dir)
	pipe, err := archive.StdoutPipe()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	untar.Stdin = pipe
	if err := untar.Start(); err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := archive.Run(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("rule ref: cannot read pushed tree %.12s: %v", commit, err)
	}
	if err := untar.Wait(); err != nil {
		cleanup()
		return nil, nil, err
	}
	store, err := ledger.OpenReadOnly(dir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("rule ref: pushed tree %.12s holds no ledger: %v", commit, err)
	}
	return store, cleanup, nil
}
