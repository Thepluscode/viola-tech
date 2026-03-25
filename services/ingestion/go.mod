module github.com/viola/ingestion

go 1.24.0

require github.com/viola/shared v0.0.0

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/viola/shared => ../../shared/go
