---
id: os-37fcf7c6
title: 'next: ledger show''s chain_invalid does not stamp its failing position, unlike verify'
state: ready
priority: P3
squad: core
created_at: "2026-09-01T08:55:24Z"
updated_at: "2026-09-03T03:22:34Z"
---

Review finding on #183, deliberately not taken there because plans/os-fa69345e.md D3 decided the opposite in advance ('the drill pins it so a future consistency cleanup does not extend the stamp there'). This card is that cleanup, made deliberate.

The tension is real. next/spec/envelope.md says position is 'the ledger position the response was computed at' and that 'every response that reached the ledger stamps it', reserving null for 'a refusal raised BEFORE a tip was ever read (a malformed invocation, an unreadable ledger directory)'. A show --position scan that read record 0 and then failed parsing record 1 reached the ledger; it simply never reached the tip.

And the tree already has a precedent on the other side: runLedgerVerify stamps fail.Position, so a corrupted chain gets the FAILING position, asserted by TestLedgerVerifyFailureExitCodes ('verification refusals are position-stamped envelopes'). show's chain_invalid is the only chain failure that stamps nothing.

D3's argument was that 'the count it reached is a count of records read before an error, not a statement about the chain'. That is right about a COUNT and wrong about the field: the stamp is documented as where the response was computed, not as the chain's length. Under that reading, verify's behaviour is the conformant one.

The change is small and needs no ledger change. runLedgerShow already tracks count inside the Records callback, and store.Records returns the ParseRecord error BEFORE invoking the callback for that position, so for a failure at position p the callback ran for 0..p-1 and count == p exactly. The failing position is already in scope; it just is not stamped.

Scope: next/cmd/seed/ledger.go (one return), next/cmd/seed/ledger_test.go (invert the D3 assertion added by #183), and a sentence in next/spec/envelope.md if the 'before a tip was ever read' wording needs sharpening to say 'before any position was read'.
