# `receipts/` — evidence records (D4.5)

One JSON receipt per task at `receipts/<task-id>.json`: the merge-base plan's
sha256, the diff summary (excluding `receipts/**`), validation-command
results, and run metadata. Above L1 the CI verify check regenerates the
receipt and is the author of record — a locally forged receipt is overwritten
and the mismatch fails the check (R11). Cards closed via the no-PR path carry
the D7 exemption: their evidence record is the server-attributed dispatch
artifact instead.
