FROM golang:1.24-alpine AS builder
WORKDIR /repo
COPY shared/go/ ./shared/go/
COPY services/intel/ ./services/intel/
WORKDIR /repo/services/intel
RUN CGO_ENABLED=0 go build -mod=vendor -o /intel ./cmd/intel

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /intel /intel
EXPOSE 8082
ENTRYPOINT ["/intel"]
