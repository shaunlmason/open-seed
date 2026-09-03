// The ranking projection (plans/os-c7554f18.md D3; next/spec/ranking.md):
// the strongest qualified tuples per capability, derived from the
// verified prefix alone at the tip record's own instant. Input-free
// like every projection but the report: it reads no gold (the gold is
// outside the tree), so its agreement fields are null and it says so.

package project

import (
	"encoding/json"

	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/keyring"
	"github.com/shaunlmason/open-seed/next/internal/ranking"
)

// RankingFile is the projection's one view file.
const RankingFile = "ranking.json"

// Ranking returns the ranking projection.
func Ranking() Projection {
	return Projection{Name: "ranking", Version: "1", Build: buildRanking}
}

// DeriveRanking is the shared derivation: the projection builds from
// it and the doctor reads it at a ledger's tip. The instant is the tip
// record's ts, never a clock, so the same prefix derives the same
// bytes; an empty prefix derives at no instant.
func DeriveRanking(records []*event.Record) (ranking.Ranking, error) {
	ring, _, err := keyring.StateAt(records)
	if err != nil {
		return ranking.Ranking{}, err
	}
	asOf := ""
	if n := len(records); n > 0 && records[n-1] != nil {
		asOf = records[n-1].Event.TS
	}
	return ranking.Derive(ranking.Inputs{Records: records, Ring: ring, AsOf: asOf})
}

func buildRanking(records []*event.Record, _ Inputs) (map[string][]byte, error) {
	r, err := DeriveRanking(records)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{RankingFile: append(b, '\n')}, nil
}
