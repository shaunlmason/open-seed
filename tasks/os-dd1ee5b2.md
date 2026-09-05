---
id: os-dd1ee5b2
title: 'next: seed serve drops explicit id:null requests as notifications, but JSON-RPC 2.0 requires a response'
state: backlog
priority: P2
squad: core
created_at: "2026-09-05T19:47:08Z"
---

Found by a read-only review of the tree at bfa01638 (2026-09-05). handleRequest (next/cmd/seed/serve.go:109-115) computes notification := len(req.ID) == 0 || string(req.ID) == "null". JSON-RPC 2.0 defines a notification by the ABSENCE of the id member; an explicit null id is a valid request id that requires a response carrying id null. Repro: echo '{"jsonrpc":"2.0","id":null,"method":"version"}' | seed serve emits no response, so a conforming caller waits indefinitely. This card: treat only an absent id as a notification (track presence, not value) and add a drill pinning the id:null response. Tier: standard. Plan-first.
