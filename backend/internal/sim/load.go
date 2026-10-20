package sim

import (
	"bytes"
	"errors"
	"io"

	"go.yaml.in/yaml/v3"
)

// Decode parses one topology file with strict YAML (unknown fields are errors). It does not validate.
func Decode(data []byte) (*Topology, error) {
	var t Topology
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&t); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("file is empty")
		}
		return nil, err
	}
	return &t, nil
}
