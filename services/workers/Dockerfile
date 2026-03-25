FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/workers/ ./services/workers/
WORKDIR /repo/services/workers
RUN CGO_ENABLED=0 go build -mod=vendor -o /workers ./cmd/workers

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /workers /workers
ENTRYPOINT ["/workers"]
