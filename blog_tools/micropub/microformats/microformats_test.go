package microformats

import (
	"io"
	"reflect"
	"testing"
)

type MockReader struct {
	expectedData  []byte
	expectedError error
}

func (m MockReader) Read(p []byte) (n int, err error) {
	copy(p, m.expectedData)
	return 0, m.expectedError
}

func TestParse(t *testing.T) {

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    MicroFormat
		wantErr bool
	}{
		{
			name: "hello",
			args: args{r: MockReader{expectedData: []byte(`{}`)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
