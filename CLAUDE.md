# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Smart Translator is a real-time speech-to-speech translation broadcasting platform. A host speaks into a microphone → audio is sent via WebSocket → Deepgram transcribes speech → Groq LLM translates text to English → OpenAI TTS (or Fish Audio TTS) synthesizes translated audio → audio is broadcast to listeners via chunked HTTP streaming.

The repo is a monorepo with two active components:
- `smarttranslator_api/` — Go backend (Gin HTTP server, REST + WebSocket)
- `smarttranslator_web/` — Vue 3 + Vite + TypeScript frontend
- `vox_web_old/` — Legacy React frontend, unused

## Commands

### Backend (`smarttranslator_api/`)

```bash
go build ./...
go test ./internal/... ./pkg/...
go test -tags=integration ./tests/...
go test -tags=integration -run TestPublishHandler_HappyPath ./tests/integration/
golangci-lint run
swag init -g cmd/smarttranslator/production/main.go   # regenerate Swagger docs
```

### Frontend (`smarttranslator_web/`)

```bash
npm run dev        # dev server (Vite)
npm run build      # production build → dist/
npm run typecheck  # vue-tsc --noEmit
npm run lint       # eslint src --max-warnings 0
```

## Architecture

### Backend

**Entry point**: `cmd/smarttranslator/production/main.go` — reads config from Docker secrets (files pointed to by env vars), sets up the pgx connection pool with exponential-backoff retry, initializes multi-sink zap logger (stdout, Loki, info file, error file), and calls `internal.NewRouter`.

**Router** (`internal/router.go`): Registers all routes with middleware chain: zap request logger → panic recovery → rate limiter (100 req/min in-memory) → security headers → CORS (only `smarttranslator.store`). The active router uses flat route registration — the commented-out grouped version is the old style.

**Package structure**:
- `internal/auth` — JWT access/refresh token pair (15-min access, random hex refresh stored as SHA-256 hash), Argon2id password hashing, OAuth2 (GitHub + Google)
- `internal/hub` — Core broadcast engine (see below)
- `internal/user` — User info handler
- `internal/user/voice` — Voice reference CRUD (upload audio samples for TTS cloning)
- `internal/admin/logs` — Dynamic zap log level endpoint
- `pkg/models` — Shared `Config`, `Pool`, error constants, `GetLogger`/`GetWSUpgrader` context helpers

**Hub broadcast pipeline** (`internal/hub/`):
1. `PublishHandler` upgrades the connection to WebSocket, verifies hub ownership via in-memory `HostAndHubs` cache, then launches four concurrent goroutines via `errgroup`:
   - WebSocket reader → writes raw audio bytes to an `io.Pipe`
   - `Deepgram.do` — streams audio to Deepgram STT via WebSocket callback, pushes transcripts to `StringChan`
   - `Groq.do` — reads transcripts, calls Groq/OpenAI-compatible chat completion for translation, pushes translated tokens to `StringChan`
   - `VoiceAgent.Do` — reads translated tokens, synthesizes audio via TTS, calls `hub.Publish(chunk)` for each audio chunk
2. `Hub` is a fan-out router: it has a `broadcast` channel (capacity 1024) and a `consumers` map. `run()` goroutine reads from broadcast and non-blockingly sends to each consumer's `Send` channel (capacity 128). Full consumer channels silently drop chunks.
3. `ListenHandler` registers a `Consumer`, streams audio chunks as `audio/mpeg` chunked transfer encoding.
4. `Manager` is a thread-safe map of `hubID → *Hub`. `HostAndHubs` is a thread-safe map of `userID → []hubID` (in-memory ownership, lost on restart).

**TTS abstraction**: `VoiceAgent` / `VoiceAgentBuilder` interfaces allow swapping TTS backends. Current implementations: `OpenAIBuilder`/`OpenAI` (uses OpenAI TTS API), `BuildHolder`/`FishHolder` (Fish Audio WebSocket streaming, currently not wired in the active router — `OpenAISDK` middleware is used instead).

**Gin context keys** injected by middleware:
- `"logger"` → `*zap.Logger`
- `"user_id"` → `string`
- `"hub"` → `*hub.Hub`
- `"host_and_hub_cache"` → `*hub.HostAndHubs`
- `"voice_agent_builder"` → `hub.VoiceAgentBuilder`
- `"ws_upgrader"` → `*websocket.Upgrader`

**Migrations**: SQL files in `db/migrations/`, run by goose (embedded via `//go:embed`). Deployed as a one-shot `migrate` service in Docker Swarm.

**Integration tests** (`tests/integration/`): Use `testcontainers-go` to spin up a real `postgres:16` container. Build tag: `integration`. Test utilities live in `tests/utils/{db,helpers,mocks,vars}`. `NewPublishRouterFull` uses `zap.NewNop()` (not `zaptest.NewLogger`) to avoid a data race with the Deepgram SDK's internal goroutine outliving the test.

### Frontend

**Stack**: Vue 3 + Vite + TypeScript. `vite.config.ts` sets `base: '/'`.

**API**: `src/api.ts` — axios client with `baseURL: https://smarttranslator.store/api` and `withCredentials: true` (cookie auth).

**Auth state**: `src/store.ts` — module-level `reactive({ user, loading })`. Hydrated in `main.ts` via `/user/info` before mount. Router guard in `router.ts` redirects unauthenticated users to `/`.

**Routes**: `/` (login+signup+OAuth), `/host` (hub dashboard), `/host/:id/broadcast`, `/listen/:id?` (no auth required), `/profile`.

**Broadcast flow**: `BroadcastView` — opens WebSocket to `/hub/{id}/publish?lang={lang}&file_id={fileId}`, pipes `MediaRecorder` chunks (250 ms intervals, webm/opus) as binary WebSocket messages.

**Listener flow**: `ListenView` — fetches `/hub/{id}/listen` (chunked `audio/mpeg`), feeds chunks through `MediaSource` API into an `<audio>` element.
