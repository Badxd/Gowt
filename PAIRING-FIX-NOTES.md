# Gowt pairing patch

Changes in pairing.go only: removes global DNS/IPv4 overrides, adds timeout-bound reconnect attempts, serializes pairing requests (including session generator), and improves logs. Existing MongoDB, bot commands, website, Procfile and go.mod are unchanged.

Build: `go mod download && go build ./...` (requires internet and matching Go toolchain). Run: `go run .`.

If `websocket disconnected before info query returned response` persists, inspect transport/protocol errors and WhatsMeow version; this patch cannot guarantee remote WhatsApp server compatibility. Never post SESSION strings publicly.
