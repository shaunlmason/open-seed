// The ready-queue projection (plans/os-fecfb3f7.md step 2): the
// registered claimable-work surface, schema fixed now. The v0
// derivation is "none": the Phase 4 vocabulary defines no claimable
// states, so ready is empty by definition and the derivation field says
// so machine-readably; a consumer can refuse to treat an underived
// queue as meaning "nothing to do". Phase 5 item 1 replaces the
// derivation (and its marker) with the transition table's; the
// eligibility filter follows later, per the build plan.

package project

import (
	"encoding/json"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

// QueueFile is the ready-queue projection's one view file.
const QueueFile = "queue.json"

// QueueSchemaVersion is the queue view's schema version; entries'
// field set is Phase 5's to extend.
const QueueSchemaVersion = "1"

// QueueDerivationNone marks the v0 derivation: no readiness vocabulary
// exists before the transition table.
const QueueDerivationNone = "none"

// QueueEntry is one claimable subject. The field set is minimal by
// design; Phase 5 extends it with what the transition table derives.
type QueueEntry struct {
	Subject       string `json:"subject"`
	SincePosition int    `json:"since_position"`
}

// QueueView is the queue.json shape.
type QueueView struct {
	SchemaVersion string       `json:"schema_version"`
	Derivation    string       `json:"derivation"`
	Ready         []QueueEntry `json:"ready"`
}

// Queue returns the ready-queue projection.
func Queue() Projection {
	return Projection{Name: "queue", Build: buildQueue}
}

func buildQueue(_ []*event.Record) (map[string][]byte, error) {
	view := QueueView{
		SchemaVersion: QueueSchemaVersion,
		Derivation:    QueueDerivationNone,
		Ready:         []QueueEntry{},
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{QueueFile: append(b, '\n')}, nil
}
