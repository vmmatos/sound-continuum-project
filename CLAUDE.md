# CLAUDE.md

Instructions for Claude Code sessions working in this repository.

## Read before working

1. [`docs/manifesto.md`](docs/manifesto.md) before making any editorial
   decision.
2. [`docs/memory/README.md`](docs/memory/README.md).
3. [`docs/memory/project-context.md`](docs/memory/project-context.md).
4. [`docs/memory/current-state.md`](docs/memory/current-state.md).
5. [`docs/memory/decisions.md`](docs/memory/decisions.md) before proposing
   any architectural change.
6. [`docs/memory/roadmap.md`](docs/memory/roadmap.md) when planning future
   work.

## Rules

- Never silently contradict or modify the manifesto. It is canonical for
  editorial decisions; technical decisions follow it, they don't redefine
  it.
- Never introduce significant architectural complexity (new service, new
  datastore, new infra, new framework) without recording the decision in
  `docs/memory/decisions.md`.
- Update `docs/memory/current-state.md` after meaningful implementation
  work — keep it a snapshot, not a changelog.
- Update `docs/memory/decisions.md` when a meaningful architectural or
  product decision is made.
- Don't create documentation for trivial changes.
- Don't assume prior conversation context exists — the repo docs are the
  persistent project context.
- Keep `TODO.md` current: what's done, what's in progress, what's planned.
- Write clear, specific commit messages (why, not just what).
- Keep Claude Code skills/harness in mind — prefer existing skills, hooks,
  and conventions over reinventing workflow tooling.
- **Always work on a branch and open a PR for review. Do not push directly
  to `main`, and do not merge the PR yourself — leave it for the user to
  review.**

## MVP constraints (current milestone)

No Spotify integration, no recommendation engine, no AI-driven editorial
decisions, no microservices, no Kubernetes, no Redis (unless a concrete
need appears), no authentication, no user management, no complex database
infra. See `docs/memory/decisions.md` for the full reasoning.

## Stack

- Backend: Go, standard library (`net/http`), no framework.
- Frontend: Vue 3 + Vite + Pinia.
