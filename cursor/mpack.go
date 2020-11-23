package cursor

import (
	"github.com/vmihailenco/msgpack/v5"
)

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
