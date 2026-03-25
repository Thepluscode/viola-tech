FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/ingestion/ ./services/ingestion/
WORKDIR /repo/services/ingestion
RUN CGO_ENABLED=0 go build -mod=vendor -o /ingestion ./cmd/ingestion

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /ingestion /ingestion
ENTRYPOINT ["/ingestion"]
