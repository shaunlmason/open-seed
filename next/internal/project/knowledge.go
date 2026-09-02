// The knowledge projection (plans/os-f30ee0d3.md D3; build plan Phase
// 11 item 1): the curation stages rendered from the verified prefix,
// input-free like every projection but the report. Dead ends by
// contract, hypotheses with their stage, lessons by path, and the
// promotions that cite no admitted hypothesis, which are unbound
// rather than lessons.

package project

import (
	"encoding/json"
	"sort"

	"github.com/shaunlmason/open-seed/next/internal/curation"
	"github.com/shaunlmason/open-seed/next/internal/event"
)

// KnowledgeFile is the projection's one view file.
const KnowledgeFile = "knowledge.json"

// KnowledgeView is knowledge.json.
type KnowledgeView struct {
	// Stages counts the pipeline's stages: observations (dead ends
	// recorded), hypotheses proposed, lessons promoted, and the
	// unbound promotions.
	Stages     KnowledgeStages                   `json:"stages"`
	DeadEnds   map[string][]curation.DeadEndFact `json:"dead_ends"`
	Hypotheses []curation.HypothesisFact         `json:"hypotheses"`
	Lessons    []curation.LessonFact             `json:"lessons"`
	Unbound    []curation.LessonFact             `json:"unbound"`
	Anomalies  int                               `json:"anomalies"`
}

// KnowledgeStages is the count per stage, the report's section too.
type KnowledgeStages struct {
	Observations int `json:"observations"`
	Hypotheses   int `json:"hypotheses"`
	Promoted     int `json:"promoted"`
	Lessons      int `json:"lessons"`
	Unbound      int `json:"unbound"`
}

// Knowledge returns the knowledge projection.
func Knowledge() Projection {
	return Projection{Name: "knowledge", Version: "1", Build: buildKnowledge}
}

// DeriveKnowledge renders the view from the prefix: the shared
// derivation the projection builds from and the CLI's show renders.
func DeriveKnowledge(records []*event.Record) KnowledgeView {
	st := curation.Fold(records)
	view := KnowledgeView{DeadEnds: st.DeadEnds, Hypotheses: []curation.HypothesisFact{},
		Lessons: []curation.LessonFact{}, Unbound: []curation.LessonFact{}, Anomalies: st.Anomalies}
	if view.DeadEnds == nil {
		view.DeadEnds = map[string][]curation.DeadEndFact{}
	}
	for _, id := range st.HypothesisIDs() {
		h, _ := st.Hypothesis(id)
		view.Hypotheses = append(view.Hypotheses, *h)
		if h.Stage == curation.StagePromoted {
			view.Stages.Promoted++
		}
	}
	for _, l := range st.Lessons {
		view.Lessons = append(view.Lessons, l)
	}
	sort.Slice(view.Lessons, func(i, j int) bool { return view.Lessons[i].Pos < view.Lessons[j].Pos })
	if st.Unbound != nil {
		view.Unbound = st.Unbound
	}
	for _, ds := range view.DeadEnds {
		view.Stages.Observations += len(ds)
	}
	view.Stages.Hypotheses = len(view.Hypotheses)
	view.Stages.Lessons = len(view.Lessons)
	view.Stages.Unbound = len(view.Unbound)
	return view
}

func buildKnowledge(records []*event.Record, _ Inputs) (map[string][]byte, error) {
	b, err := json.MarshalIndent(DeriveKnowledge(records), "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{KnowledgeFile: append(b, '\n')}, nil
}
