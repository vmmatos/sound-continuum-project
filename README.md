# Sound Continuum

A weekly music curation project connecting timeless classics, contemporary
music, and emerging artists through intentional musical bridges — a
continuous musical journey rather than disconnected weekly playlists.

Editorial philosophy: see [`docs/manifesto.md`](docs/manifesto.md).

## Status

MVP under development. M1 (concept) is complete; M2 (technical foundation)
is in progress. No production system exists yet. See
[`docs/memory/current-state.md`](docs/memory/current-state.md) for details
and [`docs/memory/roadmap.md`](docs/memory/roadmap.md) for direction.

## Project structure

```
cmd/server/       HTTP server entrypoint
internal/health/  health-check handler
docs/manifesto.md  editorial philosophy (canonical)
docs/memory/       persistent project memory for future sessions
```

## Local development

Requires Go 1.24+.

```
make run    # start the server on :8080 (or $PORT)
make test   # run tests
make build  # build a binary to bin/server
```

Health check:

```
curl localhost:8080/health
# {"status":"ok"}
```

Environment variables: see [`.env.example`](.env.example).
