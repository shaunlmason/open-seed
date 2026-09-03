package main

// The code-ref half of the enforced hook (plans/os-465e356e.md D1, D2;
// SEED-NEXT.md §II.2 "an actor's credential allows proposing to the
// ledger and pushing to its own authorized code branches", §II.14 "the
// protected surface … write-denied to every agent key whose work it
// gates"). Every ref update that is not the guarded ledger ref is judged
// from two things: the ledger as it stands at the guarded ref's current
// tip (standing from the keyring, claim holders from the lifecycle
// fold, derived exactly as the ledger half derives them) and the
// identity the transport asserted for the pusher. Nothing here reads
// the pushed tree for its inputs, and nothing survives the process: a
// replacement hook host rebuilds every decision from the repository.
//
// The rules, in the reference deployment's terms:
//
//   - the default branch (the repository's HEAD symref) fast-forwards
//     for operator standing only, and never force-updates or deletes;
//   - a contract branch refs/heads/seed/<contract> takes any update
//     from the actor holding the ACTIVE claim on <contract>, and
//     nothing from anyone else — authorized by the claim, so it closes
//     when the window closes;
//   - tags are created by operator standing and immutable after;
//   - operator standing may otherwise create or fast-forward any ref;
//     an agent credential may touch nothing else; a push asserting no
//     identity is refused on every code ref;
//   - a push by a credential without operator standing whose new
//     commits touch a protected path is refused on any ref, the list
//     read from the deployment declaration at the default branch's
//     CURRENT tip, never from the pushed commits, with the
//     declaration's own path protected by construction.

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/posture"
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// pusherEnv carries the authenticated pusher's identity — the actor's
// key fingerprint — from the transport into the hook. In the reference
// deployment the ssh forced command derives it from the authenticated
// key (the credential and the signing key are one ed25519 key, so the
// fingerprint is derived, never configured); a fixture's harness sets
// it when it hands out a credential. The hook trusts it the way it
// trusts the transport: forging it is stealing another actor's
// credential, which the security invariant places outside the
// adversary (SEED-NEXT.md §I.2).
const pusherEnv = "SEED_PUSHER"

// contractBranchPrefix is where an actor's authorized code surface
// lives: one branch per contract, authorized by the active claim.
const contractBranchPrefix = "refs/heads/seed/"

// codeRefContext is what the code-ref half decides from, loaded once
// per push from the repository as it stands before the push.
type codeRefContext struct {
	guarded       string
	pusher        string
	defaultBranch string
	// defaultTip is the default branch's current commit, empty when the
	// branch is unborn.
	defaultTip string
	// operator reports whether the pusher holds operator standing
	// (a governance root in active standing, or an operator grant).
	operator bool
	// active reports whether the pusher still holds active standing to
	// push code at all: an enrolled actor with the claim capability, or
	// operator standing. A revoked or suspended key holds neither, so
	// its contract branch closes with its standing even before its
	// still-open claim is reaped (plans/os-465e356e.md D9).
	active bool
	// root reports whether the pusher is a governance root in active
	// standing. The protected surface is changed only by the governance
	// root (SEED-NEXT.md §II.14), so an operator — the maintenance lane
	// holds operator and is an agent key — may fast-forward the default
	// branch but a commit of its touching a protected path refuses; only
	// root is exempt from the protected check (Copilot review on #247).
	root bool
	// holder reports whether the pusher holds the active claim on the
	// contract; nil when no ledger stands yet.
	fold *transition.Fold
	// declaration is the deployment declaration at the default branch's
	// tip; nil when the branch is unborn or carries none, in which case
	// the surface is the declaration path alone.
	declaration *posture.Config
	// declarationErr records a declaration that exists and does not
	// parse: agent pushes refuse until an operator fixes it, and the
	// message says so.
	declarationErr error
}

// loadCodeRefContext derives the context. A repository whose guarded
// ref holds no admitted genesis has no standing to judge code refs by,
// so every code push refuses until one is admitted — the ledger half
// admits the genesis push whoever carries it.
func loadCodeRefContext(gitDir, guarded, pusher string) (*codeRefContext, error) {
	ctx := &codeRefContext{guarded: guarded, pusher: pusher}
	out, err := exec.Command("git", "--git-dir", gitDir, "symbolic-ref", "HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("rule ref: cannot determine the default branch (HEAD is not a symbolic ref): %v", err)
	}
	ctx.defaultBranch = strings.TrimSpace(string(out))
	if tip, err := exec.Command("git", "--git-dir", gitDir, "rev-parse", "--verify", "--quiet", ctx.defaultBranch+"^{commit}").Output(); err == nil {
		ctx.defaultTip = strings.TrimSpace(string(tip))
	}

	tipOut, err := exec.Command("git", "--git-dir", gitDir, "rev-parse", "--verify", "--quiet", guarded+"^{commit}").Output()
	if err != nil {
		// No ledger yet: the context judges every code ref refused.
		return ctx, nil
	}
	store, cleanup, err := materialize(gitDir, strings.TrimSpace(string(tipOut)))
	if err != nil {
		return nil, err
	}
	defer cleanup()
	resolve, _, err := genesis.Bootstrap(store)
	if err != nil {
		return nil, fmt.Errorf("rule ref: the guarded ref's ledger does not bootstrap: %v", err)
	}
	var records []*event.Record
	if _, err := store.VerifyFromGenesis(resolve, ledger.WithObserver(func(pos int, rec *event.Record) {
		records = append(records, rec)
	})); err != nil {
		return nil, fmt.Errorf("rule ref: the guarded ref's ledger does not verify: %v", err)
	}
	ring, _, err := keyring.StateAt(records)
	if err != nil {
		return nil, fmt.Errorf("rule ref: %v", err)
	}
	table, err := transition.Default()
	if err != nil {
		return nil, fmt.Errorf("rule ref: %v", err)
	}
	ctx.fold = table.FoldRecords(records)
	ctx.operator = pusher != "" && ring.HasAnyCapability(pusher, []string{keyring.CapOperator})
	ctx.active = pusher != "" && ring.HasAnyCapability(pusher, []string{keyring.CapClaim, keyring.CapOperator})
	ctx.root = pusher != "" && ring.IsActiveRoot(pusher)

	if ctx.defaultTip != "" {
		body, err := exec.Command("git", "--git-dir", gitDir, "show", ctx.defaultTip+":"+posture.DeclarationPath).Output()
		if err == nil {
			cfg, perr := posture.Parse(body)
			if perr != nil {
				ctx.declarationErr = perr
			} else {
				ctx.declaration = cfg
			}
		}
	}
	return ctx, nil
}

// protects reports whether a path is on the protected surface as the
// default branch declares it. With no declaration the surface is the
// declaration's own path.
func (c *codeRefContext) protects(p string) bool {
	if c.declaration != nil {
		return c.declaration.Protects(p)
	}
	return p == posture.DeclarationPath || strings.HasPrefix(p, posture.DeclarationPath+"/")
}

// authorize judges one non-ledger ref update.
func (c *codeRefContext) authorize(gitDir, oldID, newID, ref string) error {
	if c.pusher == "" {
		return fmt.Errorf("rule ref: %s: no pusher identity asserted (%s) — code refs are refused without one", ref, pusherEnv)
	}
	if c.fold == nil {
		return fmt.Errorf("rule ref: %s: the guarded ref %s holds no admitted ledger, so no standing exists to authorize a code push — admit the genesis first", ref, c.guarded)
	}
	create, del := oldID == zeroID, newID == zeroID
	fastForward := func() bool {
		if create {
			return true
		}
		return exec.Command("git", "--git-dir", gitDir, "merge-base", "--is-ancestor", oldID, newID).Run() == nil
	}
	switch {
	case ref == c.defaultBranch:
		if del {
			return fmt.Errorf("rule ref: %s: deletion of the default branch is refused for everyone", ref)
		}
		if !fastForward() {
			return fmt.Errorf("rule ref: %s: non-fast-forward update of the default branch is refused for everyone (admitted history is append-only on the code side too)", ref)
		}
		if !c.operator {
			return fmt.Errorf("rule ref: %s: %s holds no operator standing — the default branch fast-forwards for operator standing only; an agent credential proposes on its contract branch and the merge is an operator's act", ref, c.pusher)
		}
		if !c.root {
			return c.checkProtected(gitDir, oldID, newID, ref)
		}
		return nil
	case strings.HasPrefix(ref, "refs/tags/"):
		if !create {
			return fmt.Errorf("rule ref: %s: tags are immutable — an update or deletion of an existing tag is refused for everyone (SEED-NEXT.md §II.14)", ref)
		}
		if !c.operator {
			return fmt.Errorf("rule ref: %s: %s holds no operator standing — tags are created by operator standing only", ref, c.pusher)
		}
		return nil
	case strings.HasPrefix(ref, contractBranchPrefix):
		contract := strings.TrimPrefix(ref, contractBranchPrefix)
		s, ok := c.fold.State(contract)
		switch {
		case !ok:
			return fmt.Errorf("rule ref: %s: no contract %s exists on the ledger, so no claim authorizes the branch", ref, contract)
		case s.Claim == nil:
			return fmt.Errorf("rule ref: %s: no claim is active on %s (state %s) — a contract branch is authorized by the active claim, and a closed window authorizes nothing", ref, contract, s.State)
		case s.Claim.Holder != c.pusher:
			return fmt.Errorf("rule ref: %s: %s does not hold the active claim on %s (holder %s, fence %d) — a contract branch is authorized by the active claim and nothing else", ref, c.pusher, contract, s.Claim.Holder, s.Claim.Fence)
		case !c.active:
			return fmt.Errorf("rule ref: %s: %s holds the claim on %s but no active standing — a revoked or suspended key's contract branch closes with its standing, even before its claim is reaped", ref, c.pusher, contract)
		}
		if del || c.root {
			return nil
		}
		return c.checkProtected(gitDir, oldID, newID, ref)
	default:
		if !c.operator {
			return fmt.Errorf("rule ref: %s is outside %s's authorized code surface — an agent credential pushes its own contract branch (%s<contract>) while it holds the claim, and nothing else", ref, c.pusher, contractBranchPrefix)
		}
		if del {
			return fmt.Errorf("rule ref: %s: deletion is refused — operator standing creates or fast-forwards refs, and rewriting or deleting one is not a push the reference deployment admits", ref)
		}
		if !fastForward() {
			return fmt.Errorf("rule ref: %s: non-fast-forward update is refused — operator standing creates or fast-forwards refs, and rewriting one is not a push the reference deployment admits", ref)
		}
		if !c.root {
			return c.checkProtected(gitDir, oldID, newID, ref)
		}
		return nil
	}
}

// checkProtected refuses a NON-ROOT push whose NEW commits touch a
// protected path (the governance root alone changes the surface,
// SEED-NEXT.md §II.14). New means introduced by this push: reachable from
// the new tip, not from the old tip, not from the default branch — a
// merge of the default branch into a contract branch carries the
// operator's protected-surface commits without introducing them, and
// the combined diff of a merge commit lists only what the merge itself
// changed against every parent.
func (c *codeRefContext) checkProtected(gitDir, oldID, newID, ref string) error {
	if c.declarationErr != nil {
		return fmt.Errorf("rule protected: %s: the deployment declaration at %s:%s does not parse (%v) — agent pushes refuse until an operator repairs it", ref, c.defaultBranch, posture.DeclarationPath, c.declarationErr)
	}
	args := []string{"--git-dir", gitDir, "rev-list", newID}
	if oldID != zeroID {
		args = append(args, "^"+oldID)
	}
	if c.defaultTip != "" {
		args = append(args, "^"+c.defaultTip)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return fmt.Errorf("rule protected: %s: cannot enumerate the pushed commits: %v", ref, err)
	}
	for _, commit := range strings.Fields(string(out)) {
		paths, err := exec.Command("git", "--git-dir", gitDir, "diff-tree", "--no-commit-id", "--name-only", "-r", "-c", "--root", commit).Output()
		if err != nil {
			return fmt.Errorf("rule protected: %s: cannot read commit %.12s: %v", ref, commit, err)
		}
		for _, p := range strings.Split(strings.TrimSpace(string(paths)), "\n") {
			if p == "" {
				continue
			}
			if c.protects(p) {
				return fmt.Errorf("rule protected: %s: commit %.12s touches %q, which is on the protected surface declared at %s:%s — the surface is write-denied to every agent key whose work it gates (SEED-NEXT.md §II.14); a change there is the governance root's, through PR and owner review", ref, commit, p, c.defaultBranch, posture.DeclarationPath)
			}
		}
	}
	return nil
}
