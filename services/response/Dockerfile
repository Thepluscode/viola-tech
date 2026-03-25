FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/response/ ./services/response/
WORKDIR /repo/services/response
RUN CGO_ENABLED=0 go build -mod=vendor -o /response ./cmd/response

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /response /response
EXPOSE 8083
ENTRYPOINT ["/response"]
