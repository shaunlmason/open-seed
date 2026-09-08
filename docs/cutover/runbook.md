# The cutover runbook

The operator's step-by-step for the self-hosting cutover, at the
`cooperative` posture [`decisions/0005`](../../decisions/0005-cooperative-posture.md)
records. It exists because the flip is the one moment where a wrong command
costs a spent root key: **every command below has been run**, against this
repository's real v1 state, in the rehearsal
[`next/docs/promotion.md`](../../next/docs/promotion.md) section 3 records.

**Division of labour.** Part 1 is yours alone and never enters an agent
session: it mints the keys. From Part 2 on, an agent can drive, given the
**fingerprints** from Part 1. A fingerprint is a public value; the private keys
stay with you.

Three things the written order got wrong before the rehearsal, now corrected
here and in the packet: `--ledger` must name a path that does **not** exist,
`actor.enrolled` takes the fingerprint as subject with the public key in the
*payload*, and the transform table must be refreshed against live state first.

---

## Part 1 — the keys (operator only)

Mint the governance root and one key per acting lane. Five lanes act in this
deployment; add others later with the same two commands.

```sh
mkdir -p ~/seed-keys && cd ~/seed-keys
for k in root implementer dispatcher verifier maintenance observer; do
  ssh-keygen -t ed25519 -N '' -C "seed-$k" -f "$k"
done
chmod 600 ~/seed-keys/*
```

**Back these up now.** The root key is the governance anchor: losing it means
the chain can never admit another actor change.

For each key, read off the two public values the flip needs.

```sh
# The FINGERPRINT (safe to share; this is what the agent needs).
# The throwaway ledger path must not already exist.
seed init --ledger /tmp/fp-$$ --key ~/seed-keys/root >/dev/null 2>&1
seed init --ledger /tmp/fp2-$$ --key ~/seed-keys/root | \
  python3 -c 'import json,sys; print(json.load(sys.stdin)["result"]["governance_root"][0])'
rm -rf /tmp/fp-$$ /tmp/fp2-$$

# The RAW PUBLIC KEY in hex (also safe; needed to enrol a lane).
awk '{print $2}' ~/seed-keys/implementer.pub | base64 -d | tail -c 32 | od -An -tx1 | tr -d ' \n'; echo
```

The fingerprint and the public key hex are **different values**, and swapping
them is the mistake `actor.enrolled` refuses on. Record both per key.

Hand over: the root fingerprint, and for each lane its fingerprint and public
key hex. Nothing else.

---

## Part 2 — the flip

Run from a clean checkout of `main` with the cutover branch merged up to but
**not including** the `AGENTS.md` move.

### 1. Declare the deployment

Copy the block from the packet's "The deployment" into `seed.json` at the
repository root, substituting the root fingerprint for
`<the root key's fingerprint>`. That is the only substitution.

### 2. Anchor and export v1

```sh
scripts/seed state anchor          # tags the state head and pushes the tag
scripts/seed state export > /tmp/export.json
```

### 3. Refresh the transform table against live state, and rehearse

**Do not skip this.** CI proves the migration against a snapshot and is
structurally blind to a v1 run-log verb added since. The rehearsal caught
exactly that: `exempt-plan`, `mail-send` and `mail-ack` had no rows, with three,
one and one occurrences in the whole log, while every drill was green.

```sh
make fixture-import                       # regenerate at the anchor just made
cd next && go test ./internal/importer/   # the drills, against the fresh fixture
```

Then rehearse the whole of Part 2 under a **throwaway** key, into throwaway
directories. If the import refuses `import_unmapped`, add a row per verb to
`next/spec/import-open-seed.json` and `next/internal/importer/table.json` (they
must stay byte-identical) and repeat. Only proceed when the rehearsal reaches
step 8 clean.

### 4. Import (the genesis transform)

`--ledger` must name a path that does **not** exist. A pre-created empty
directory refuses `unavailable`.

```sh
seed import --from-open-seed /tmp/export.json \
  --source . --repo . \
  --ledger /srv/seed-ledger --artifacts /srv/seed-artifacts \
  --key ~/seed-keys/root
```

Expect `ok: true` with `counts.records` matching the export and
`counts.dispositions` equal to it: every record gets a disposition, and a drop
is a disposition.

### 5. Apply the declaration over the imported chain

```sh
seed init --ledger /srv/seed-ledger --key ~/seed-keys/root \
  --preseed seed.json --lanes next/lanes
```

Expect the protocol activations the declaration names beyond the import's
(`seed/6` and `seed/7` today). Run it twice: the second run must report
`"unchanged": true` and append nothing.

### 6. Check

```sh
seed preseed check --config seed.json --ledger /srv/seed-ledger --lanes next/lanes
```

Expect `ok: true` and `"pending": []`.

### 7. Enrol and grant the lanes

The subject is the **fingerprint**; the **public key hex** goes in the payload.

```sh
enrol() {  # $1 lane name, $2 fingerprint, $3 public key hex, $4.. grants
  seed ledger append --ledger /srv/seed-ledger --key ~/seed-keys/root \
    --verb actor.enrolled --subject "$2" \
    --payload "{\"key\":\"$3\",\"kind\":\"agent\",\"name\":\"$1\"}"
  shift 3
  for cap in "$@"; do
    seed ledger append --ledger /srv/seed-ledger --key ~/seed-keys/root \
      --verb actor.granted --subject "$2" --payload "{\"capability\":\"$cap\"}"
  done
}
```

The grants each lane needs, from `next/lanes/*.json`:

| lane | grants |
|---|---|
| implementer | `claim` |
| planner | `claim` |
| dispatcher | `dispatch` |
| verifier | `verdict`, `sealer` |
| curator | `curate` |
| maintenance | `maintenance`, `operator` |
| supervisor | `supervise` |
| observer | `observer` |

`sealer` must stay disjoint from `claim` and `operator` in both directions, so
the verifier identity must not also hold either. The boundary refuses a grant
that breaks this.

### 8. Verify, then push

```sh
seed ledger verify --ledger /srv/seed-ledger   # verifies from genesis
seed ledger audit  --ledger /srv/seed-ledger   # the five bars, all empty
```

Commit the ledger directory (`HEAD` and `segments/*.jsonl`) as the tree of
`refs/seed/ledger` and push it once.

### 9. Merge the cutover pull request

`git mv docs/cutover/AGENTS.md AGENTS.md`, delete its staging comment, retire
`scripts/seed task` from the remaining role files and workflows, and merge.
**That merge is the flip**, and it is the escalated decision build plan §5
reserves. Freeze `seed-state` at its final anchor: read-only, never written
again.

---

## Part 3 — the audit, immediately after the flip

[`decisions/0004`](../../decisions/0004-shadow-run-substitution.md) binds one act
after the flip. It was to run at day 7;
[`decisions/0007`](../../decisions/0007-no-day-7-wait.md) dropped the wait and
kept the check, so this runs as the last step of the flip rather than a return
visit a week later:

```sh
seed ledger audit --ledger /srv/seed-ledger
```

Append the reading to the packet's divergence log. **A red bar is a defect card
and blocks stage 4, not a note.** Read minutes after the flip this catches a
broken import, a bad linkage or an unreserved spend in the imported history,
which is the bulk of what the flip risks; it cannot catch a defect that only
appears under real load, because there has been none. `0007` records that trade.

## Part 4 — retirement

Only once the audit reads clean: [`docs/v1-retirement.md`](../v1-retirement.md)
stages 4 and 5. Stage 4 deletes the inventory that
`next/internal/retirement` holds to the tree; stage 5 archives
`open-seed-engine`, read-only, never deleted. Stage 4 is one commit and is
revertible, and the frozen `seed-state` ref and its anchor tags are kept
permanently, so the history survives the deletion either way.
