FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/detection/ ./services/detection/
WORKDIR /repo/services/detection
RUN CGO_ENABLED=0 go build -mod=vendor -o /detection ./cmd/detection

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /detection /detection
COPY services/detection/rules/ /rules/
EXPOSE 9090
ENTRYPOINT ["/detection"]
