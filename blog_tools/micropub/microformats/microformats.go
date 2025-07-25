package microformats

import "io"

type Microformat struct {
	Type       []string         `json:"type"`
	Properties map[string][]any `json:"properties"`
}

func Parse(r io.Reader) (Microformat, error) {
	mf := Microformat{}
	return mf, nil
}
