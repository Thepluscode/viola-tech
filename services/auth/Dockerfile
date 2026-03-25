FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/auth/ ./services/auth/
WORKDIR /repo/services/auth
RUN CGO_ENABLED=0 go build -mod=vendor -o /auth ./cmd/auth

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /auth /auth
EXPOSE 8081
ENTRYPOINT ["/auth"]
