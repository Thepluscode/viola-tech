FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/gateway-api/ ./services/gateway-api/
WORKDIR /repo/services/gateway-api
RUN CGO_ENABLED=0 go build -mod=vendor -o /gateway-api ./cmd/gateway-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /gateway-api /gateway-api
EXPOSE 8080
ENTRYPOINT ["/gateway-api"]
