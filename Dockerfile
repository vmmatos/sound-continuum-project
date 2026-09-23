# Backend (Go) — development image.
# Build context is the repo root: the Go module lives at repo root
# (cmd/, internal/, go.mod), not under a backend/ directory.

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /server ./cmd/server

FROM alpine:latest
COPY --from=build /server /server
EXPOSE 8080
CMD ["/server"]
