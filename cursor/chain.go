package cursor

func Chain(m Marshaller, es ...Encoder) Marshaller {
	return chain{m: m, es: es}
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
