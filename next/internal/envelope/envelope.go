// Package envelope renders the affordance envelope: the versioned,
// schema-stable JSON shape every verb response uses (charter Part II
// section 10 and Appendix B; next/spec/envelope.md is the normative field
// and exit-code table). Exit codes reuse v1 semantics where they match
// (build-plan fixed default); new codes are allocated in the spec table,
// never ad hoc in code.
package envelope

import (
	"encoding/json"
	"io"
)

// V is the envelope schema version carried in every response.
const V = "seed-envelope/0"

// Exit codes: the v1-inherited allocations. The table in
// next/spec/envelope.md is authoritative; these constants mirror it.
const (
	ExitOK                = 0
	ExitContention        = 2
	ExitInvalidTransition = 3
	ExitNotFound          = 4
	ExitUnavailable       = 5
	ExitFenced            = 6
	ExitHalted            = 7
	ExitChainInvalid      = 8
	ExitClassificationRef = 9
	ExitVersionMismatch   = 10
	ExitRemoteRejected    = 11
	ExitHeadRegression    = 12
	ExitPostureInvalid    = 13
	ExitOutOfGrant        = 14
	ExitStale             = 15
	ExitPlanRequired      = 16
	ExitNotIndependent    = 17
	ExitUngated           = 18
	ExitSpecUnrunnable    = 19
	ExitChecksRed         = 20
	ExitReceiptMismatch   = 21
	// The sealed-checks refusals (plans/os-3128535a.md;
	// next/spec/sealed-checks.md): a broken seal (missing ciphertext,
	// commitment mismatch, or an empty-checks envelope), an identity
	// outside the recipient set (rotation lag), and an above-trivial
	// subject with no commitment at the verifier boundary.
	ExitSealBroken   = 22
	ExitNotRecipient = 23
	ExitUnsealed     = 24
	// The red-verdict lockout (plans/os-d2497eb7.md): rendering pass
	// over a submission an authenticated fail already judged refuses
	// until a new submission.
	ExitRedLocked = 25
	// The lane-validation refusal (plans/os-cf1c9688.md;
	// next/spec/lanes.md): a checked-in lane manifest makes a claim
	// the tables refuse — a grant outside the vocabulary, an act whose
	// accepted capabilities the lane does not hold, a liveness source
	// that is not a work step, a missing fragment. Distinct from
	// posture_invalid, which judges the deployment's posture
	// declaration rather than a role definition.
	ExitLaneInvalid = 26
	ExitUsage       = 64
	ExitUnreadable  = 66
)

// Error is the machine-branchable half of a refusal: a stable code to
// branch on and a human message to read.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Budget mirrors the reservation block. It stays nil until budget
// reservations land (build plan Phase 7).
type Budget struct {
	Reserved  string `json:"reserved"`
	Remaining string `json:"remaining"`
}

// Envelope is the one response shape (charter Appendix B). Result and Error
// are mutually exclusive. Position is the ledger position the response was
// computed at: null until the ledger lands (Phase 1), then always stamped.
type Envelope struct {
	V           string         `json:"v"`
	OK          bool           `json:"ok"`
	Result      map[string]any `json:"result"`
	Error       *Error         `json:"error"`
	Position    *string        `json:"position"`
	Affordances []string       `json:"affordances"`
	Budget      *Budget        `json:"budget"`
	Exit        int            `json:"exit"`
}

// OK builds a success envelope for result.
func OK(result map[string]any) *Envelope {
	return &Envelope{V: V, OK: true, Result: result, Affordances: []string{}, Exit: ExitOK}
}

// Fail builds a refusal envelope with the given exit code and error code.
func Fail(exit int, code, message string) *Envelope {
	return &Envelope{V: V, OK: false, Error: &Error{Code: code, Message: message}, Affordances: []string{}, Exit: exit}
}

// Render writes the envelope as a single JSON line.
func (e *Envelope) Render(w io.Writer) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}
