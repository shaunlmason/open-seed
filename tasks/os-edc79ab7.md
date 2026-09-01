---
id: os-edc79ab7
title: 'engine: receipt verify surfaces the failing validation command''s output'
state: cancelled
priority: P3
squad: core
created_at: "2026-08-31T21:20:26Z"
updated_at: "2026-09-01T03:13:00Z"
---

When a receipt validation command fails during 'seed receipt verify --run', the runner reports only the exit code ('validation command failed (exit 2): make check'), swallowing the command's output — a CI flake is unattributable from the verify log (seen on open-seed#156: make check failed once, passed twice on the same head, failing test unknown). Capture and emit (or tail) the failing command's output in the failure message.
