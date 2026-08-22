# `skills/` — Agent Skills source of truth (D8)

One directory per skill containing `SKILL.md` (agentskills.io format). This
tree is the editable source; `.claude/skills/` and `.agents/skills/` are
generated fan-outs (`seed sync`, Phase 6) — never edit those copies. The v2
manifest/lockfile (`seed.yaml` / `seed.lock`) pins third-party skills.
