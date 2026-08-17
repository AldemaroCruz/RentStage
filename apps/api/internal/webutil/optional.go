package webutil

import (
	"bytes"
	"encoding/json"
)

// Optional distinguishes an omitted JSON field from a field explicitly set to null.
// Set is false when the field was omitted. Value is nil when the field was supplied as null.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}
