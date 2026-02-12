# syntax=docker/dockerfile:1

FROM ghcr.io/a-h/templ:latest AS templ-generate
WORKDIR /app
COPY --chown=65532:65532 . .
RUN ["templ", "generate", "./..."]

FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY --from=templ-generate /app /app
RUN go build -mod=vendor -tags netgo -ldflags='-s -w' -o /app/out ./cmd/api/main.go

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/out /app/out
COPY --from=builder /app/cmd/web/assets /app/cmd/web/assets
COPY --from=builder /app/configs /app/configs
EXPOSE 8080
CMD ["/app/out"]
