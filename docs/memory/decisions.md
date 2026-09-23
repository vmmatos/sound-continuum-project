# Decisions

Record only meaningful architectural and product decisions here, not
trivial implementation details. Each entry: Decision, Context, Reason,
Consequences.

---

**Decision:** Keep the MVP intentionally simple.

**Context:** Sound Continuum is a new, experimental project.

**Reason:** There is no guarantee the concept will gain traction. The first
goal is validating the musical curation concept, workflow, and audience
response — not building for scale that may never be needed.

**Consequences:** Every other decision below follows from this one.

---

**Decision:** Use a simple Go backend.

**Context:** MVP needs an HTTP API surface for the frontend to eventually
call.

**Reason:** Go's standard library is sufficient for a small API; no need
for a framework at this stage.

**Consequences:** `net/http` only, no framework, minimal dependencies.

---

**Decision:** Use Vue 3 + Vite + Pinia for the frontend.

**Context:** MVP needs a frontend, not yet built as of Card 15.

**Reason:** Lightweight, conventional, fast dev loop.

**Consequences:** Frontend work (later cards) should follow this stack.

---

**Decision:** Avoid microservices for the MVP.

**Reason:** No requirement yet justifies the operational overhead.

---

**Decision:** Avoid Kubernetes.

**Reason:** Nothing to orchestrate yet; adds infrastructure the MVP doesn't
need.

---

**Decision:** Avoid Redis unless a concrete requirement appears.

**Reason:** No caching or pub/sub need has been identified.

---

**Decision:** Avoid complex authentication.

**Reason:** MVP has no user accounts or protected actions yet.

---

**Decision:** Avoid unnecessary infrastructure (e.g. Dockerfile deferred as
of Card 15).

**Reason:** `go run` / `go build` is sufficient for local dev; nothing is
being deployed yet that requires a container.

**Superseded (Card 18):** Dockerfiles + Docker Compose were introduced for
local dev consistency — see the Card 18 decision below. `go run` / `go
build` remain fine outside Docker; this entry no longer reflects current
state.

---

**Decision:** Do not make the final editorial decision automatically.

**Reason:** Per the manifesto, editorial judgment remains human.

---

**Decision:** AI must not become a dependency of the editorial workflow.

**Context/Note:** This does not mean "AI can never be used." AI may assist
development or research. But the Sound Continuum MVP must remain fully
understandable and functional without an AI agent making the final
curation decision.

---

**Decision:** Prefer simple solutions until a real requirement proves
additional complexity necessary.

**Reason:** Guiding principle for all future architectural choices in this
project.

---

**Decision:** Frontend lives at `frontend/` at repo root; the Go backend is
not moved into a `backend/` directory.

**Context:** Card 16 introduced the Vue 3 + Vite frontend. The Go backend
(`cmd/`, `internal/`, `go.mod`) already lived at repo root from Card 15,
un-namespaced.

**Reason:** Wrapping the existing backend in a `backend/` directory purely
for symmetry with `frontend/` would be an unrelated restructuring of
working code, with no functional benefit, done as a side effect of an
unrelated card. `frontend/` alone is enough to make the physical
frontend/backend separation explicit.

**Consequences:** Repo root mixes backend files (`cmd/`, `internal/`,
`go.mod`, `Makefile`) with `frontend/` and `docs/`. If this becomes
confusing later, moving the backend into `backend/` is a small, reversible
follow-up — not a blocker now.

---

**Decision:** Add a Docker Compose development environment; persist SQLite
through a named volume mounted into the backend container, not a separate
database service.

**Context:** Card 18 needs frontend and backend to start together locally
with a consistent, reproducible environment, and to establish where the
future SQLite database will live.

**Reason:** SQLite is an embedded database — it runs inside the process
that uses it. A separate `sqlite` container/service would misrepresent the
architecture and add orchestration complexity (networking, startup
ordering) that an embedded database doesn't need. A named volume mounted at
`/data` in the backend container gives persistence without a database
process of its own. The backend `Dockerfile` lives at repo root (build
context `.`), matching the existing decision to keep the Go module at repo
root rather than under `backend/`.

**Consequences:** `docker compose up --build` starts both services;
`docker compose down` preserves the `sqlite_data` volume, `docker compose
down -v` removes it. No SQLite application/repository code was added — the
volume only reserves `/data` as the future database location. No reverse
proxy, healthcheck orchestration, or production deployment concerns were
introduced.
