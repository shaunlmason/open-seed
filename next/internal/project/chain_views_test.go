package project_test

// The reconciliation-chain view drills (plans/os-6cdc15be.md):
// contracts v6 carries the verdict, requested, and merged facts;
// report v3 carries the record-derivable divergence classes with the
// evidence-grade surface named; a clean full chain yields no findings.

import (
	"strings"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
	"github.com/shaunlmason/open-seed/next/internal/project"
	"github.com/shaunlmason/open-seed/next/internal/reconcile"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

func TestReconciliationViews(t *testing.T) {
	root, worker := pKey(t, 1), pKey(t, 2)
	dir, resolve, add := fixtureChain(t, root, worker)
	add(root, version.Protocol, ledger.UpgradeVerb, "system", `{"to": "`+version.Seed1+`"}`)
	add(root, version.Seed1, keyring.VerbEnrolled, pFP(t, worker), enrollJSON(t, worker, "agent", "worker"))
	packet := `{"acceptance": ["done means done"], "decisions": [], "base": "1234567..1234567", "refs": [], "findings": []}`
	sha := strings.Repeat("ef", 20)
	receipt := strings.Repeat("ab", 32)

	// c-F walks the full chain: submission at 6, pass verdict at 7,
	// request at 8 citing it, observation at 9.
	add(root, version.Seed1, "intent.filed", "c-F", `{"intent": "ship", "tier": "standard", "budget": "s", "routing": "core"}`)
	add(root, version.Seed1, "contract.specified", "c-F", `{"acceptance": {"ref": "specs/f.md @ abc1234", "executable": false}}`)
	add(worker, version.Seed1, "claim.taken", "c-F", `{}`)
	add(worker, version.Seed1, "submission.made", "c-F", `{"fence": "5", "packet": `+packet+`}`)
	add(worker, version.Seed1, "verdict.rendered", "c-F", `{"verdict": "pass", "receipt": "`+receipt+`", "submission": "6", "independence": "L1"}`)
	add(worker, version.Seed1, "merge.requested", "c-F", `{"verdict": "7"}`)
	add(root, version.Seed1, "merge.observed", "c-F", `{"merged": "`+sha+`", "pr": "pr/1"}`)

	// c-G reaches done through a raw-pushed observation with no
	// verdict at all: merge_without_verdict.
	add(root, version.Seed1, "intent.filed", "c-G", `{"intent": "slip", "tier": "standard", "budget": "s", "routing": "core"}`)
	add(root, version.Seed1, "contract.specified", "c-G", `{"acceptance": {"ref": "specs/g.md @ abc1234", "executable": false}}`)
	add(worker, version.Seed1, "claim.taken", "c-G", `{}`)
	add(worker, version.Seed1, "submission.made", "c-G", `{"fence": "12", "packet": `+packet+`}`)
	add(root, version.Seed1, "merge.observed", "c-G", `{"merged": "`+sha+`", "pr": "pr/2"}`)

	// c-H holds a pass verdict with no merge yet: unreconciled,
	// neutrally.
	add(root, version.Seed1, "intent.filed", "c-H", `{"intent": "wait", "tier": "standard", "budget": "s", "routing": "core"}`)
	add(root, version.Seed1, "contract.specified", "c-H", `{"acceptance": {"ref": "specs/h.md @ abc1234", "executable": false}}`)
	add(worker, version.Seed1, "claim.taken", "c-H", `{}`)
	add(worker, version.Seed1, "submission.made", "c-H", `{"fence": "17", "packet": `+packet+`}`)
	add(worker, version.Seed1, "verdict.rendered", "c-H", `{"verdict": "pass", "receipt": "`+receipt+`", "submission": "18", "independence": "L1"}`)

	out := rebuildAll(t, dir, resolve)

	var entries []project.ContractEntry
	readView(t, out, "contracts", project.ContractsFile, &entries)
	byID := map[string]project.ContractEntry{}
	for _, e := range entries {
		byID[e.Subject] = e
	}
	f := byID["c-F"]
	if f.Verdict == nil || f.Verdict.Position != "7" || f.Verdict.Verdict != "pass" || f.Verdict.Receipt != receipt {
		t.Fatalf("c-F must carry the verdict fact: %+v", f.Verdict)
	}
	if f.Requested == nil || *f.Requested != "8" {
		t.Fatalf("c-F must carry the request position: %+v", f.Requested)
	}
	if f.Merged == nil || f.Merged.Position != "9" || f.Merged.SHA != sha {
		t.Fatalf("c-F must carry the merged fact: %+v", f.Merged)
	}
	g := byID["c-G"]
	if g.State == nil || *g.State != "done" || g.Verdict != nil || g.Merged == nil || g.Anomalies == 0 {
		t.Fatalf("c-G is done with a recorded merge, no verdict, and a visible anomaly: %+v", g)
	}

	var rep project.ReportView
	readView(t, out, "report", project.ReportFile, &rep)
	if rep.Reconciliation == nil {
		t.Fatal("the report carries the reconciliation section when work subjects exist")
	}
	by := rep.Reconciliation.ByClass
	if by[reconcile.ClassMergeWithoutVerdict] != 1 || by[reconcile.ClassUnreconciled] != 1 || by[reconcile.ClassChainSkipped] != 0 {
		t.Fatalf("the record-derivable classes must count c-G and c-H only: %+v", by)
	}
	for _, fnd := range rep.Reconciliation.Findings {
		if fnd.Subject == "c-F" {
			t.Fatalf("a clean full chain yields no findings: %+v", fnd)
		}
	}
	if !strings.Contains(rep.Reconciliation.EvidenceGrade, "seed reconcile") {
		t.Fatalf("the evidence-grade surface is named: %q", rep.Reconciliation.EvidenceGrade)
	}
}
