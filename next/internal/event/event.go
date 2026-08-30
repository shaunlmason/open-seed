// Package event implements the Seed canonical event form: the wire-level
// core of the ledger (charter Part II section 1, Appendix B). Encodings are
// normative in next/spec/protocol.md: JCS (RFC 8785) canonical bytes,
// lowercase-hex SHA-256 chain hashes, lowercase-hex Ed25519 signatures, and
// the {"event": ..., "sig": ...} ledger record wrapper. Verifiers recompute
// canonical bytes from parsed events; nothing trusts stored byte sequences.
package event

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gowebpki/jcs"
)

// EmptyHash is the genesis prev: the SHA-256 of zero bytes
// (next/spec/protocol.md, "Canonical event form").
const EmptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// Typed refusal reasons, mapped to envelope error codes by later phases.
var (
	ErrBadSignature   = errors.New("signature does not verify over the canonical form")
	ErrBadSigEncoding = errors.New("sig is not 128 lowercase hex characters")
	ErrBadPayload     = errors.New("payload is not a JSON object")
)

// Event carries exactly the canonical fields of next/spec/protocol.md.
// Payload stays raw JSON: verb-specific schemas land with their phases, and
// canonicalization must not depend on Go-side typing.
type Event struct {
	V       string          `json:"v"`
	TS      string          `json:"ts"`
	Actor   string          `json:"actor"`
	Verb    string          `json:"verb"`
	Subject string          `json:"subject"`
	Payload json.RawMessage `json:"payload"`
	Prev    string          `json:"prev"`
}

// Canonical returns the RFC 8785 (JCS) bytes of the event object including
// prev. These are the signed bytes and the hashed bytes; they are computed
// fresh on every call, never cached from storage.
func (e *Event) Canonical() ([]byte, error) {
	if !payloadIsObject(e.Payload) {
		return nil, ErrBadPayload
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return jcs.Transform(b)
}

func payloadIsObject(p json.RawMessage) bool {
	for _, c := range p {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// Hash is the chain hash: lowercase-hex SHA-256 of the canonical bytes. The
// successor event's prev cites it.
func (e *Event) Hash() (string, error) {
	b, err := e.Canonical()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Record is the ledger record wrapper {"event": ..., "sig": ...}. Wrapper
// field order and whitespace are irrelevant to identity: only the inner
// event's canonical bytes are signed and hashed.
type Record struct {
	Event Event  `json:"event"`
	Sig   string `json:"sig"`
}

// Sign returns a Record for e signed by priv: the Ed25519 signature over
// the canonical bytes, encoded as 128 lowercase hex characters.
func Sign(e Event, priv ed25519.PrivateKey) (*Record, error) {
	b, err := e.Canonical()
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(priv, b)
	return &Record{Event: e, Sig: hex.EncodeToString(sig)}, nil
}

// Verify recomputes the canonical bytes from the parsed event and checks
// the record's signature against pub.
func (r *Record) Verify(pub ed25519.PublicKey) error {
	sig, err := hex.DecodeString(r.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrBadSigEncoding
	}
	b, err := r.Event.Canonical()
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, b, sig) {
		return ErrBadSignature
	}
	return nil
}

// Marshal renders the record as one JSON line for JSONL segment storage.
func (r *Record) Marshal() ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// ParseRecord parses one ledger record line. It tolerates any wrapper field
// order and whitespace; identity comes from recomputation, not bytes.
func ParseRecord(line []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(line, &r); err != nil {
		return nil, fmt.Errorf("ledger record does not parse: %w", err)
	}
	return &r, nil
}
