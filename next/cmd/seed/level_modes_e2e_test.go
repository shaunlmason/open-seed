package main

// The independence levels in both modes (plans/os-99829835.md D6,
// AC7): a critical contract, which the tier table holds to L2, reaches
// done through the production machinery only when the verifier's
// declaration separates from the window's, and an executable gated
// spec reaches L3 with one family.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/artifact"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/loop"
	"github.com/shaunlmason/open-seed/next/internal/seal"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// upgradeTo walks the remote chain's active version up to the named
// one, one register entry at a time.
func (m *modeStand) upgradeTo(t *testing.T, to string) {
	t.Helper()
	for _, v := range []string{version.Seed2, version.Seed3, version.Seed4} {
		m.appendRaw(ledger.UpgradeVerb, "system", `{"to": "`+v+`"}`)
		m.active = v
		if v == to {
			return
		}
	}
}

// criticalContract stages a critical contract as BACKGROUND (the
// fixture's D3 posture): the filing, the spec (prose, or executable
// and gated), the operator's plan approval the plan gate reads, the
// seal the tier requires, and the offer. The seal is built through the
// library and signed by the sealer-capable verifier, because `seal
// create` reads a ledger directory and the modes run on the remote
// posture; nothing the drills assert comes from it.
func (m *modeStand) criticalContract(t *testing.T, subject, sealer string, executable bool) {
	t.Helper()
	m.appendRaw("intent.filed", subject, `{"intent": "drill", "tier": "critical", "budget": "small", "routing": "core"}`)
	spec := fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": false}}`, m.spec)
	if executable {
		spec = fmt.Sprintf(`{"acceptance": {"ref": "accept.md @ %s", "executable": true, "gate": "pr/6 @ %s"}}`, m.spec, m.spec)
	}
	m.appendRaw("contract.specified", subject, spec)
	// The receipt binds the approved plan's bytes at the merge-base,
	// so the anchor names a file the fixture repository holds there:
	// the submissions below range from the spec commit. From seed/4
	// the approval carries the plan's content digest
	// (plans/os-6bd9ffff.md D5); a background fact, so the zero digest
	// is enough for the shape the boundary demands.
	approval := fmt.Sprintf(`{"plan": "accept.md @ %s", "pr": "pr/3 @ %s"}`, m.spec, m.spec)
	if version.LevelsApply(m.active) {
		approval = fmt.Sprintf(`{"plan": "accept.md @ %s", "pr": "pr/3 @ %s", "digest": "%s"}`, m.spec, m.spec, strings.Repeat("0", 64))
	}
	m.appendRaw(transition.PlanApprovedVerb, subject, approval)

	env, err := seal.NewEnvelope([]string{"true"})
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := env.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	commitment, err := env.Commitment()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := seal.Encrypt(plaintext, recipientKeys(eligibleRecipients(m.ringOf(t))))
	if err != nil {
		t.Fatal(err)
	}
	if err := artifact.Open(artifactsDir("", m.src)).PutSealed(commitment, ct); err != nil {
		t.Fatal(err)
	}
	if e, code := runEnv(t, "ledger", "append", "--remote", m.remote, "--state", m.state,
		"--key", m.keys[sealer], "--verb", transition.CheckSealedVerb, "--subject", subject,
		"--payload", `{"commitment": "`+commitment+`"}`); code != 0 {
		t.Fatalf("seal: %d %+v", code, e)
	}
	offer := fmt.Sprintf(`{"eligibility": {"capabilities": ["claim"], "tiers": ["critical"]}, "expires": %q}`,
		time.Now().UTC().Add(time.Hour).Format(time.RFC3339))
	if e, code := runEnv(t, "ledger", "append", "--remote", m.remote, "--state", m.state,
		"--key", m.keys["supervisor"], "--verb", "offer.published", "--subject", subject,
		"--payload", offer); code != 0 {
		t.Fatalf("offer: %d %+v", code, e)
	}
}

// work runs the implementing actor's loop once: it claims, and inside
// the window the supervisor's start declares the worker's
// configuration under the named model, so the window carries the
// declaration every level is compared against; the loop submits.
func (m *modeStand) work(t *testing.T, worker, model string) string {
	t.Helper()
	d, err := loop.New(implementerManifest(t), loopVerbs{}, m.posture(), m.keys[worker],
		loop.WorkFunc(func(s string, sit loop.Situation) (int, error) {
			e, code := runEnv(t, append(append([]string{"run", "start"}, m.posture()...),
				"--key", m.keys["supervisor"], "--subject", s,
				"--principal", "acme", "--model", model, "--tool-policy", "default")...)
			if code != 0 {
				return 0, fmt.Errorf("the supervisor's start: %d %+v", code, e.Error)
			}
			return 2, nil
		}), loop.WithBase(m.spec+".."+m.head))
	if err != nil {
		t.Fatal(err)
	}
	step, err := d.Step(5)
	if err != nil || step.Outcome != loop.Submitted {
		t.Fatalf("the loop claims, works and submits: %+v %v", step, err)
	}
	return step.Subject
}

// renderAs renders a pass by the named verifier, declaring the
// verifier's configuration when a model is given.
func (m *modeStand) renderAs(t *testing.T, verifier, subject, model string) (ledgerEnv, int) {
	t.Helper()
	args := append(append([]string{"verdict", "render"}, m.posture()...),
		"--subject", subject, "--repo", m.src, "--key", m.keys[verifier], "--verdict", "pass")
	if model != "" {
		args = append(args, "--principal", "acme", "--model", model, "--tool-policy", "default")
	}
	return runEnv(t, args...)
}

// land requests and observes the merge: the terminal chain, each step
// its own admitted event, the verdict's level reapplied along it.
func (m *modeStand) land(t *testing.T, subject, holder, pr string) {
	t.Helper()
	if e, code := runEnv(t, append(append([]string{"merge", "request"}, m.posture()...),
		"--subject", subject, "--key", m.keys[holder])...); code != 0 {
		t.Fatalf("merge request: %d %+v", code, e)
	}
	if e, code := runEnv(t, append(append([]string{"merge", "observe"}, m.posture()...),
		"--subject", subject, "--key", m.keys["observer"], "--merged", m.head, "--pr", pr)...); code != 0 {
		t.Fatalf("merge observe: %d %+v", code, e)
	}
	if got := m.stateOf(t, subject); got != "done" {
		t.Fatalf("the full loop ends at done, got %q", got)
	}
}

// conformance: plans/os-99829835.md AC7 (small-team) — a critical
// contract reaches done with the verifier declaring a second model
// family, and the same family refuses level_short: the charter's floor
// of two keys on one machine can still separate the verifier's
// configuration from the worker's, and the level records that it did.
func TestSmallTeamCriticalContractReachesDoneAtL2(t *testing.T) {
	m := buildMode(t, smallTeam)
	m.upgradeTo(t, version.Seed4)
	const subject = "c-crit"
	m.criticalContract(t, subject, "verify", false)
	if got := m.work(t, "impl", "fable/5.1"); got != subject {
		t.Fatalf("the loop took the critical contract: %s", got)
	}

	e, code := m.renderAs(t, "verify", subject, "")
	if code != 64 || e.Error == nil || !strings.Contains(e.Error.Message, "--principal") {
		t.Fatalf("a critical contract refuses an undeclared verifier at usage, naming the flags: %d %+v", code, e.Error)
	}
	e, code = m.renderAs(t, "verify", subject, "fable/5.1")
	if code != 17 || e.Error == nil || e.Error.Code != "level_short" ||
		!strings.Contains(e.Error.Message, "critical") || !strings.Contains(e.Error.Message, "L2") {
		t.Fatalf("the same model family on a critical contract refuses level_short naming the tier and the requirement: %d %+v", code, e.Error)
	}
	e, code = m.renderAs(t, "verify", subject, "other/1")
	if code != 0 || e.Result["independence"] != "L2" {
		t.Fatalf("a second model family renders L2: %d %+v", code, e)
	}
	m.land(t, subject, "impl", "pr/4")
	if e, code := runEnv(t, "verdict", "check", "--ledger", m.materialize(t), "--subject", subject,
		"--repo", m.src, "--key", m.keys["verify"]); code != 0 || e.Result["independence"] != "L2" {
		t.Fatalf("the chain records the level the merge was admitted under: %d %+v", code, e)
	}
}

// conformance: plans/os-99829835.md AC7 (fleet) — an executable gated
// critical contract reaches done at L3 with one model family: the
// deterministic-first level comes from the spec and the receipt, and
// a declaration equal to the window's does not lower it.
func TestFleetExecutableCriticalContractReachesDoneAtL3(t *testing.T) {
	m := buildMode(t, fleetPlan(t))
	m.upgradeTo(t, version.Seed4)
	const subject = "c-exec"
	m.criticalContract(t, subject, "verifier", true)
	if got := m.work(t, "implementer", "fable/5.1"); got != subject {
		t.Fatalf("the loop took the critical contract: %s", got)
	}
	e, code := m.renderAs(t, "verifier", subject, "fable/5.1")
	if code != 0 || e.Result["independence"] != "L3" {
		t.Fatalf("an executable gated spec renders L3 under one family: %d %+v", code, e)
	}
	m.land(t, subject, "implementer", "pr/5")
}
