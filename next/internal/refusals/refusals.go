// Package refusals is the attempts journal behind the report's
// refusal-rate metric (plans/os-edf73d66.md; SEED-NEXT.md §II.10,
// charter III.I row 4). Refusals never reach the chain — admission
// refuses before anything is written — so the metric's source is a
// local, append-only JSONL journal of admission-boundary ATTEMPTS,
// both outcomes, kept beside the ledger: one population for the
// rate's numerator and denominator alike. The journal is operator
// telemetry — never synced by gitref, never part of verification —
// and it enters the report only as a declared, digest-covered input
// (the observations pattern).
package refusals

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gowebpki/jcs"
)

// File is the journal's name inside the ledger directory.
const File = "attempts.jsonl"

// Outcomes an attempt line may carry.
const (
	OutcomeAdmitted = "admitted"
	OutcomeRefused  = "refused"
)

// Entry is one admission-boundary attempt: the instant, the
// tip-ordinal position the response envelope stamped, the signing
// actor, the verb and subject attempted, the outcome, and — on
// refusals only — the envelope's machine code.
type Entry struct {
	TS       string `json:"ts"`
	Position string `json:"position"`
	Actor    string `json:"actor"`
	Verb     string `json:"verb"`
	Subject  string `json:"subject"`
	Outcome  string `json:"outcome"`
	Code     string `json:"code,omitempty"`
}

// Note appends one attempt line to the journal in dir, best-effort:
// journaling must never fail or slow the verb it rides, so every
// error — an unwritable directory, a full disk, a marshal failure —
// is swallowed, exactly the affordance-stamping posture.
func Note(dir string, e Entry) {
	if dir == "" {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}

// Journal is a loaded attempts journal.
type Journal struct {
	Entries []Entry
}

// Load parses the journal at path. Inputs are declared, so garbage
// is the declarer's error, never silently skipped telemetry: every
// line must decode strictly to the entry shape, carry a known
// outcome and a numeric position, and pair code with outcome
// (refusals carry one, admissions never do). Errors name the line.
func Load(path string) (*Journal, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	j := &Journal{Entries: []Entry{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var e Entry
		dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("line %d does not parse as an attempt: %v", line, err)
		}
		if dec.More() {
			return nil, fmt.Errorf("line %d carries trailing data", line)
		}
		if err := e.check(); err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		j.Entries = append(j.Entries, e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return j, nil
}

func (e Entry) check() error {
	switch e.Outcome {
	case OutcomeAdmitted:
		if e.Code != "" {
			return fmt.Errorf("admitted attempts carry no code, got %q", e.Code)
		}
	case OutcomeRefused:
		if e.Code == "" {
			return errors.New("refused attempts carry the envelope's machine code")
		}
	default:
		return fmt.Errorf("outcome must be %q or %q, got %q", OutcomeAdmitted, OutcomeRefused, e.Outcome)
	}
	if _, err := strconv.Atoi(e.Position); err != nil {
		return fmt.Errorf("position must be the envelope's decimal tip-ordinal stamp, got %q", e.Position)
	}
	if e.Verb == "" || e.Subject == "" || e.Actor == "" {
		return errors.New("attempts name actor, verb, and subject")
	}
	return nil
}

// Digest is the journal's declared-input identity: the RFC 8785
// digest over the entry list, so any change to any line rekeys the
// build that consumed it.
func (j *Journal) Digest() (string, error) {
	b, err := json.Marshal(j.Entries)
	if err != nil {
		return "", err
	}
	canonical, err := jcs.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
