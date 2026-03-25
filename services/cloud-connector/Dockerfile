FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/cloud-connector/ ./services/cloud-connector/
WORKDIR /repo/services/cloud-connector
RUN CGO_ENABLED=0 go build -o /cloud-connector ./cmd/cloud-connector

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-connector /cloud-connector
EXPOSE 8084
ENTRYPOINT ["/cloud-connector"]
