# Project Memory

This directory is a persistent, file-based memory system for Sound
Continuum. It exists so that a future Claude Code session — or any new
contributor — can understand the project without relying on prior chat
history or tribal knowledge.

## Read in this order

1. [`docs/manifesto.md`](../manifesto.md) — canonical editorial philosophy.
2. [`project-context.md`](project-context.md) — stable purpose and scope.
3. [`current-state.md`](current-state.md) — what exists right now.
4. [`decisions.md`](decisions.md) — why the project is built the way it is.
5. [`roadmap.md`](roadmap.md) — where the project is headed.

## What belongs where

- **`project-context.md`** — facts that rarely change: what the project is,
  its core idea, and pointers to editorial principles. Not a changelog.
- **`current-state.md`** — a snapshot of implementation progress. Update it
  after meaningful work; keep it a snapshot, not a detailed changelog.
- **`decisions.md`** — architectural and product decisions with their
  reasoning, in Decision / Context / Reason / Consequences form. Only
  meaningful decisions, not trivial implementation details.
- **`roadmap.md`** — high-level milestone direction. Not a detailed spec of
  future work.

## Ground rules

- `docs/manifesto.md` is canonical for editorial decisions. Nothing in this
  directory may contradict it.
- Technical decisions must follow the manifesto, never redefine it.
- If a change here would contradict the manifesto, that's a signal to stop
  and raise it, not to quietly resolve it in code.
