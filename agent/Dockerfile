FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY agent/ ./agent/
WORKDIR /repo/agent
RUN CGO_ENABLED=0 go build -o /viola-agent ./cmd/viola-agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /viola-agent /viola-agent
ENTRYPOINT ["/viola-agent"]
