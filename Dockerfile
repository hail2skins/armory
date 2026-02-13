# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
RUN apk add --no-cache make curl
WORKDIR /app
COPY . .
RUN go run github.com/a-h/templ/cmd/templ@v0.3.977 generate ./...
RUN make build-railway

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/app /app/app
COPY --from=builder /app/configs /app/configs
EXPOSE 8080
CMD ["/app/app"]
