module github.com/viola/tests/integration

go 1.24.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/segmentio/kafka-go v0.4.47
	github.com/viola/shared v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
)

replace github.com/viola/shared => ../../shared/go
