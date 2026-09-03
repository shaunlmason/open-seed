package importer

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/event"
	"github.com/shaunlmason/open-seed/next/internal/genesis"
	"github.com/shaunlmason/open-seed/next/internal/transition"
	"github.com/shaunlmason/open-seed/next/internal/version"
)

// signer is a key the importer signs with: the importing operator's,
// or one it generated for a v1 actor name.
type signer struct {
	fp   string
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

func newSigner(priv ed25519.PrivateKey) (*signer, error) {
	pub := priv.Public().(ed25519.PublicKey)
	fp, err := event.Fingerprint(pub)
	if err != nil {
		return nil, err
	}
	return &signer{fp: fp, priv: priv, pub: pub}, nil
}

// chain is the import's in-memory ledger: every record admitted
// through admit.Check against the context of everything before it,
// which is the same judgment the boundary passes on a live append.
// Nothing is written to disk until the whole history admits.
type chain struct {
	records []*event.Record
	ctx     *admit.Context
	keys    map[string]ed25519.PublicKey
}

// RefusedError is a synthesized record the boundary refused: the
// import is not performed by loosening the rules, so this is fatal and
// names the record.
type RefusedError struct {
	Position int
	Verb     string
	Subject  string
	Record   string // the export record the event was synthesized from
	Err      error
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("position %d %s on %s (from %s) refused by admission: %v", e.Position, e.Verb, e.Subject, e.Record, e.Err)
}

func (e *RefusedError) Unwrap() error { return e.Err }

func newChain(operator *signer, now time.Time) (*chain, error) {
	g, err := genesis.Build(operator.priv, nil, now)
	if err != nil {
		return nil, err
	}
	c := &chain{keys: map[string]ed25519.PublicKey{operator.fp: operator.pub}}
	c.records = append(c.records, g)
	if err := c.rebuild(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *chain) rebuild() error {
	ctx, err := admit.ContextOver(c.records)
	if err != nil {
		return err
	}
	c.ctx = ctx
	return nil
}

func (c *chain) resolve(fp string) (ed25519.PublicKey, bool) {
	pub, ok := c.keys[fp]
	return pub, ok
}

// append signs and admits one record; the position is len(records)
// before the append.
func (c *chain) append(s *signer, verb, subject string, payload []byte, ts time.Time, from string) (int, error) {
	if payload == nil {
		payload = []byte(`{}`)
	}
	if !json.Valid(payload) {
		return 0, fmt.Errorf("%s payload is not JSON", verb)
	}
	e := event.Event{
		V:       c.ctx.Active,
		TS:      ts.UTC().Format(time.RFC3339),
		Actor:   s.fp,
		Verb:    verb,
		Subject: subject,
		Payload: json.RawMessage(payload),
		Prev:    c.ctx.Tip,
	}
	rec, err := event.Sign(e, s.priv)
	if err != nil {
		return 0, err
	}
	pos := len(c.records)
	if err := admit.Check(c.ctx, rec); err != nil {
		return 0, &RefusedError{Position: pos, Verb: verb, Subject: subject, Record: from, Err: err}
	}
	c.records = append(c.records, rec)
	if err := c.rebuild(); err != nil {
		return 0, err
	}
	return pos, nil
}

// state is the subject's folded lifecycle state, or ok=false for a
// subject the lifecycle never created.
func (c *chain) state(subject string) (transition.SubjectState, bool) {
	if c.ctx.Lifecycle == nil {
		return transition.SubjectState{}, false
	}
	return c.ctx.Lifecycle.State(subject)
}

// upgradeTo appends the protocol upgrades from genesis's version to
// the target, one version at a time.
func (c *chain) upgradeTo(operator *signer, target string, ts time.Time) error {
	for _, v := range version.Supported() {
		if v == version.Protocol {
			continue
		}
		if _, err := c.append(operator, "system.protocol.upgraded", "system", []byte(`{"to": "`+v+`"}`), ts, "importer"); err != nil {
			return err
		}
		if v == target {
			return nil
		}
	}
	return fmt.Errorf("version %s is not one this build registers", target)
}
