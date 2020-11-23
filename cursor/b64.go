package cursor

import "encoding/base64"

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
	n, err := b.Encoding.Decode(decoded, input)
	if err != nil {
		return nil, err
	}

	decoded = decoded[:n]

	return decoded, nil
}
