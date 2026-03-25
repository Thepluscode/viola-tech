package kafka

import (
  "google.golang.org/protobuf/proto"
)

type Codec[T any] interface {
  Encode(v T) ([]byte, error)
  Decode(b []byte) (T, error)
  SchemaName() string
}

type ProtobufCodec[T proto.Message] struct {
  Schema string
  New    func() T
}

func (c ProtobufCodec[T]) SchemaName() string { return c.Schema }

func (c ProtobufCodec[T]) Encode(v T) ([]byte, error) { return proto.Marshal(v) }

func (c ProtobufCodec[T]) Decode(b []byte) (T, error) {
  msg := c.New()
  // Note: Cannot compare generic T to nil directly in Go
  // If New() returns an invalid value, Unmarshal will fail
  if err := proto.Unmarshal(b, msg); err != nil {
    var zero T
    return zero, err
  }
  return msg, nil
}
