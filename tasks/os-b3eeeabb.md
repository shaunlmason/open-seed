---
id: os-b3eeeabb
title: 'next: mint the deployment''s operator identity and re-sign the capability card (operator act, os-f11601e0 D2)'
state: backlog
priority: P2
squad: core
created_at: "2026-09-07T12:59:12Z"
---

OPERATOR ACT, filed by os-f11601e0 (plan #368 D2) rather than performed. An implementer must not mint a deployment's identity on the operator's behalf, so this card records the work and its constraint and waits for the operator.

WHY IT IS OWED. next/docs/decisions.md records, under "CI diffs content, not signatures": "The fixture deployment's card is signed by a throwaway key kept out of the tree." A public key cannot be derived from a SHA-256 fingerprint or from an ed25519 signature, so the signature on next/boundary/card.json is unverifiable by anyone, including its own repository. os-f11601e0 made the reader's half reachable (seed boundary verify) and made the canonical form RFC 8785, so the mechanism is sound end to end; what is missing is a key someone actually holds.

WHAT THE OPERATOR DOES. Generate the deployment's operator keypair; re-sign the card with it (seed boundary card --config fixtures/deployment/seed.json --key <the new key> --name open-seed --out boundary/card.json); check in the public half; keep the private half out of the tree.

THE CONSTRAINT THAT BINDS THE CHECK-IN (os-f11601e0 D1, next/spec/boundary.md "A key pinned in-tree is a trust anchor or it is nothing"). The public key file joins the declaration's `protected` list AND CODEOWNERS in the same change that introduces it. next/boundary/ is on neither list today. A key on neither can be swapped together with the card in one non-owner change, and a gate reading it would still pass: it would verify that the card matches whichever key the last committer supplied, which is no property at all. A key pinned without both entries is worse than no key, because it reads as a trust anchor and is not one.

ONLY THEN is a Makefile change worth making: with a protected key in the tree, `make check` can pass --pubkey-file to `seed boundary check` and the repository's gate becomes a signature gate rather than the content gate it is today. Makefile is a protected path, so that change carries owner review on its own account.

NOT THIS CARD. The canonical form (os-f11601e0, done: RFC 8785 through jcs.Transform). The reader's verb (os-f11601e0, done: seed boundary verify). The honesty pass on the content gate (os-f11601e0, done: the envelope note and next/spec/boundary.md). Nothing here changes behavior that an agent can change alone.
