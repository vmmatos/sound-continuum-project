# Backend (Go) — development image.
# Build context is backend/ — the Go module lives here (cmd/, internal/,
# go.mod), separate from frontend/.

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /server ./cmd/server

FROM alpine:latest
COPY --from=build /server /server
EXPOSE 8080
CMD ["/server"]
