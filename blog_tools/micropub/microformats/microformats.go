package microformats

import "io"

type MicroFormat struct {
}

func Parse(r io.Reader) (MicroFormat, error) {
	mf := MicroFormat{}
	return mf,  nil
}
