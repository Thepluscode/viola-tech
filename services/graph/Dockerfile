FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/graph/ ./services/graph/
WORKDIR /repo/services/graph
RUN CGO_ENABLED=0 go build -mod=vendor -o /graph ./cmd/graph

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /graph /graph
ENTRYPOINT ["/graph"]
