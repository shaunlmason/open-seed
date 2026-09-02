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
	"github.com/shaunlmason/open-seed/next/internal/transition"
)

// KnowledgeFile is the projection's one view file.
const KnowledgeFile = "knowledge.json"

// KnowledgeView is knowledge.json.
type KnowledgeView struct {
	// Stages counts the pipeline's stages: observations (dead ends
	// recorded), hypotheses proposed, promoted and contested, lessons,
	// and the unbound promotions.
	Stages     KnowledgeStages                   `json:"stages"`
	DeadEnds   map[string][]curation.DeadEndFact `json:"dead_ends"`
	Hypotheses []curation.HypothesisFact         `json:"hypotheses"`
	Contests   map[string][]curation.ContestFact `json:"contests"`
	Lessons    []KnowledgeLesson                 `json:"lessons"`
	Unbound    []curation.LessonFact             `json:"unbound"`
	Anomalies  int                               `json:"anomalies"`
}

// KnowledgeLesson is a promotion with whether it surfaces from the
// record alone (promoted, not contested) and why not; the repository
// half of surfacing (the anchor's ancestry and the digest) is the
// reader's, since a projection build holds no repository.
type KnowledgeLesson struct {
	curation.LessonFact
	Surfaces bool   `json:"surfaces"`
	Reason   string `json:"reason,omitempty"`
}

// KnowledgeStages is the count per stage, the report's section too.
type KnowledgeStages struct {
	Observations int `json:"observations"`
	Hypotheses   int `json:"hypotheses"`
	Promoted     int `json:"promoted"`
	Contested    int `json:"contested"`
	Lessons      int `json:"lessons"`
	Unbound      int `json:"unbound"`
}

// Knowledge returns the knowledge projection. Version "2" added the
// contested stage, the single-actor note and per-lesson surfacing
// (plans/os-96850e5a.md D7).
func Knowledge() Projection {
	return Projection{Name: "knowledge", Version: "2", Build: buildKnowledge}
}

// DeriveKnowledge renders the view from the prefix: the shared
// derivation the projection builds from and the CLI's show renders.
func DeriveKnowledge(records []*event.Record) KnowledgeView {
	st := curation.Fold(records)
	view := KnowledgeView{DeadEnds: st.DeadEnds, Hypotheses: []curation.HypothesisFact{}, Contests: st.Contests,
		Lessons: []KnowledgeLesson{}, Unbound: []curation.LessonFact{}, Anomalies: st.Anomalies}
	if view.DeadEnds == nil {
		view.DeadEnds = map[string][]curation.DeadEndFact{}
	}
	if view.Contests == nil {
		view.Contests = map[string][]curation.ContestFact{}
	}
	var table *transition.Table
	if t, err := transition.Default(); err == nil {
		table = t
	}
	for _, id := range st.HypothesisIDs() {
		h, _ := st.Hypothesis(id)
		fact := *h
		if table != nil {
			fact.SingleActorFamily = st.SingleActorFamily(records, table, id)
		}
		view.Hypotheses = append(view.Hypotheses, fact)
		switch h.Stage {
		case curation.StagePromoted:
			view.Stages.Promoted++
		case curation.StageContested:
			view.Stages.Contested++
		}
	}
	for _, l := range st.Lessons {
		row := KnowledgeLesson{LessonFact: l, Surfaces: true}
		if c, _ := curation.ParseCitation(l.Hypothesis); st.Contested(c.Contract) {
			row.Surfaces, row.Reason = false, "the hypothesis stands contested"
		}
		view.Lessons = append(view.Lessons, row)
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
