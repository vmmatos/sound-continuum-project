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
