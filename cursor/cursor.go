package cursor

import (
	"bytes"
	"encoding/base64"
	"github.com/vmihailenco/msgpack/v5"
)

type Type int

const (
	Before Type = 1 << iota
	After
)

type Cursor struct {
	Limit int
	Type  Type
	Value interface{}
}

type Marshaller interface {
	// When input is nil, must return an empty string
	Marshal(input interface{}) ([]byte, error)

	// When encoded is an empty string, return value must be nil
	Unmarshal(encoded []byte) (interface{}, error)
}

type Encoder interface {
	// When input is nil, must return an empty string
	Encode(input []byte) ([]byte, error)

	// When encoded is an empty string, return value must be nil
	Decode(encoded []byte) ([]byte, error)
}

type chain struct {
	m  Marshaller
	es []Encoder
}

func (c chain) Marshal(input interface{}) ([]byte, error) {
	var err error
	s, err := c.m.Marshal(input)
	if err != nil {
		return nil, err
	}

	for _, e := range c.es {
		s, err = e.Encode(s)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (c chain) Unmarshal(encoded []byte) (interface{}, error) {
	s := encoded

	var err error
	for i := len(c.es) - 1; i >= 0; i-- {
		s, err = c.es[i].Decode(s)
		if err != nil {
			return "", err
		}
	}

	return c.m.Unmarshal(s)
}

func Chain(m Marshaller, es ...Encoder) Marshaller {
	return chain{m: m, es: es}
}

func MsgPack() Marshaller {
	return mpack{}
}

type mpack struct{}

func (m mpack) Marshal(input interface{}) ([]byte, error) {
	if input == nil {
		return nil, nil
	}

	s, err := msgpack.Marshal(input)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (m mpack) Unmarshal(s []byte) (interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}

	var data interface{}
	if err := msgpack.Unmarshal(s, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func Base64(encoding *base64.Encoding) Encoder {
	return b64{encoding}
}

type b64 struct {
	*base64.Encoding
}

func (b b64) Encode(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil
	}

	encoded := make([]byte, b.Encoding.EncodedLen(len(input)))
	b.Encoding.Encode(encoded, input)
	return encoded, nil
}

func (b b64) Decode(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil
	}

	decoded := make([]byte, b.Encoding.DecodedLen(len(input)))
	_, err := b.Encoding.Decode(decoded, input)
	if err != nil {
		return nil, err
	}

	decoded = bytes.TrimRightFunc(decoded, func(r rune) bool {
		return r == 0
	})

	return decoded, nil
}
