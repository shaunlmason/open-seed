// Chain verification: full replay from the empty hash (charter Part II
// section 1; conformance III.A). Every failure names its position and a
// distinct reason, and a HEAD that is merely behind the stream (the healed
// crash window) is reported apart from a HEAD that contradicts it.
package ledger

import (
	"errors"
	"fmt"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

// Reason codes for verification failures.
const (
	ReasonBadParse     = "bad_parse"
	ReasonBadPayload   = "bad_payload"
	ReasonBadSignature = "bad_signature"
	ReasonBadPrev      = "bad_prev"
	ReasonUnknownActor = "unknown_actor"
	ReasonHeadBehind   = "head_behind"
	ReasonHeadWrong    = "head_wrong"
)

// Failure is one verification finding.
type Failure struct {
	Position int
	Reason   string
	Detail   string
}

func (f *Failure) Error() string {
	return fmt.Sprintf("position %d: %s: %s", f.Position, f.Reason, f.Detail)
}

// Report is a successful verification's summary.
type Report struct {
	Count int
	Tip   string
}

// VerifyFromGenesis replays the whole stream: parse every record,
// recompute canonical bytes, check prev linkage from the empty hash,
// verify every signature against the resolver, then compare HEAD. The
// first failure returns as a *Failure. A HEAD that is merely behind the
// stream must still be *consistent*: its tip must equal the chain hash at
// its claimed position, or it is wrong, not recoverable; and a missing
// HEAD over a non-empty stream is itself a behind state, never a pass.
func (s *Store) VerifyFromGenesis(resolve Resolver) (*Report, error) {
	head, headExists, headErr := s.ReadHead()

	tip := event.EmptyHash
	count := 0
	claimedTip := event.EmptyHash
	err := s.scan(func(pos int, segment string, line []byte) error {
		rec, err := event.ParseRecord(line)
		if err != nil {
			return &Failure{Position: pos, Reason: ReasonBadParse, Detail: err.Error()}
		}
		if rec.Event.Prev != tip {
			return &Failure{Position: pos, Reason: ReasonBadPrev,
				Detail: fmt.Sprintf("prev %.12s does not cite tip %.12s", rec.Event.Prev, tip)}
		}
		pub, ok := resolve(rec.Event.Actor)
		if !ok {
			return &Failure{Position: pos, Reason: ReasonUnknownActor, Detail: rec.Event.Actor}
		}
		if err := rec.Verify(pub); err != nil {
			reason := ReasonBadSignature
			if errors.Is(err, event.ErrBadPayload) {
				reason = ReasonBadPayload
			}
			return &Failure{Position: pos, Reason: reason, Detail: err.Error()}
		}
		h, err := rec.Event.Hash()
		if err != nil {
			return &Failure{Position: pos, Reason: ReasonBadPayload, Detail: err.Error()}
		}
		tip = h
		count = pos + 1
		if headExists && count == head.Count {
			claimedTip = h
		}
		return nil
	})
	if err != nil {
		if f, ok := err.(*Failure); ok {
			return nil, f
		}
		return nil, err
	}

	if headErr != nil {
		return nil, &Failure{Position: count, Reason: ReasonHeadWrong, Detail: headErr.Error()}
	}
	switch {
	case !headExists && count == 0:
		// clean empty ledger
	case !headExists:
		return nil, &Failure{Position: count, Reason: ReasonHeadBehind,
			Detail: fmt.Sprintf("HEAD missing over a %d-event stream (healed on next open or append)", count)}
	case head.Tip == tip && head.Count == count:
		// clean
	case head.Count < count && head.Count >= 0 && head.Tip == claimedTip:
		return nil, &Failure{Position: count, Reason: ReasonHeadBehind,
			Detail: fmt.Sprintf("HEAD at count %d behind stream count %d, consistent with its position (healed on next open or append)", head.Count, count)}
	default:
		return nil, &Failure{Position: count, Reason: ReasonHeadWrong,
			Detail: fmt.Sprintf("HEAD claims tip %.12s count %d, stream has tip %.12s count %d", head.Tip, head.Count, tip, count)}
	}
	return &Report{Count: count, Tip: tip}, nil
}
