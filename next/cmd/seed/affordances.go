package main

// The envelope's advisory fields (plans/os-f5551001.md; charter
// §II.10): every append-path response carries the verbs currently
// legal for its signing actor on its subject, and the budget block
// on budget-active subjects, computed at the response's tip by the
// same rule set admission enforces. Stamping is advisory and never
// breaks the verb: any failure to compute degrades to the empty
// list and the null block.

import (
	"crypto/ed25519"

	"github.com/shaunlmason/open-seed/next/internal/admit"
	"github.com/shaunlmason/open-seed/next/internal/envelope"
	"github.com/shaunlmason/open-seed/next/internal/ledger"
)

// stampAffordances fills env.Affordances and env.Budget for the
// signing key's actor on the subject, at the ledger directory's
// current tip. Responses lacking any of the three inputs (the
// remote transport's path included, which holds no local ledger
// directory) keep their fields as built.
func stampAffordances(env *envelope.Envelope, dir string, key ed25519.PrivateKey, subject string) *envelope.Envelope {
	if env == nil || dir == "" || subject == "" || len(key) != ed25519.PrivateKeySize {
		return env
	}
	store, err := ledger.Open(dir)
	if err != nil {
		return env
	}
	ctx, err := admit.ContextAt(store)
	if err != nil {
		return env
	}
	env.Affordances = admit.Affordances(ctx, key, subject)
	if reserved, remaining, ok := admit.BudgetBlock(ctx, subject); ok {
		env.Budget = &envelope.Budget{Reserved: reserved, Remaining: remaining}
	}
	return env
}
