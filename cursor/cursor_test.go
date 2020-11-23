package cursor

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testRoundTrip(t *testing.T, marshaller Marshaller, input interface{}, check func(interface{})) {
	mdata, err := marshaller.Marshal(input)
	assert.NoError(t, err)

	t.Logf("%v: %v (%[2]s)\n", input, mdata)

	umdata, err := marshaller.Unmarshal(mdata)
	assert.NoError(t, err)

	check(umdata)
}

type spec struct {
	input interface{}
	check func(interface{})
}

func testRoutine(t *testing.T, marshaller Marshaller) {
	for _, s := range specs(t) {
		testRoundTrip(t, marshaller, s.input, s.check)
	}
}

func specs(t *testing.T) []spec {
	return []spec{
		{
			input: nil,
			check: func(i interface{}) {
				assert.Nil(t, i)
			},
		},
		{
			input: 42,
			check: func(i interface{}) {
				assert.Equal(t, int8(42), i)
			},
		},
		{
			input: "hey",
			check: func(i interface{}) {
				assert.Equal(t, "hey", i)
			},
		},
		{
			input: []interface{}{"1", 2, 3.0},
			check: func(i interface{}) {
				ia := i.([]interface{})
				assert.Len(t, ia, 3)
				assert.Equal(t, "1", ia[0])
				assert.Equal(t, int8(2), ia[1])
				assert.Equal(t, 3.0, ia[2])
			},
		},
	}
}

func TestMsgPack(t *testing.T) {
	testRoutine(t, MsgPack())
}

func TestChainMsgPack(t *testing.T) {
	testRoutine(t, Chain(MsgPack()))
}

func TestChainMsgPackBase64(t *testing.T) {
	testRoutine(t, Chain(MsgPack(), Base64(base64.StdEncoding)))
}

type reverse struct{}

func (reverse) reverse(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	out := make([]byte, len(in))
	copy(out, in)

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (r reverse) Encode(input []byte) ([]byte, error) {
	return r.reverse(input), nil
}

func (r reverse) Decode(encoded []byte) ([]byte, error) {
	return r.reverse(encoded), nil
}

func TestReverse(t *testing.T) {
	encoded, _ := reverse{}.Encode([]byte{1, 2, 3, 4, 5})
	assert.Equal(t, []byte{5, 4, 3, 2, 1}, encoded)

	decoded, _ := reverse{}.Decode(encoded)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, decoded)
}

// Test encoding/decoding order
func TestChainMsgPackBase64Reverse(t *testing.T) {
	testRoutine(t, Chain(MsgPack(), Base64(base64.StdEncoding), reverse{}))
}

func TestChainMsgPackReverseBase64(t *testing.T) {
	testRoutine(t, Chain(MsgPack(), reverse{}, Base64(base64.StdEncoding)))
}
