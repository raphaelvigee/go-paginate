package cursor

import (
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

type EncoderDecoder interface {
	// When input is nil, must return an empty string
	Encode(input interface{}) (string, error)

	// When encoded is an empty string, return value must be nil
	Decode(encoded string) (interface{}, error)
}

func MsgPackBase64EncoderDecoder() EncoderDecoder {
	return msgpackBase64{}
}

type msgpackBase64 struct {
}

func (m msgpackBase64) Encode(input interface{}) (string, error) {
	if input == nil {
		return "", nil
	}

	data, err := msgpack.Marshal(input)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (m msgpackBase64) Decode(encoded string) (interface{}, error) {
	if len(encoded) == 0 {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err = msgpack.Unmarshal(decoded, &data); err != nil {
		return nil, err
	}

	return data, nil
}
