package microformats

import (
	"io"
	"os"
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
		mfFileName string
		r          io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    Microformat
		wantErr bool
	}{
		{
			name: "hello",
			args: args{
				mfFileName: "photo.json",
			},
			want: Microformat{
				Type: []string{"h-entry"},
			},
		},
	}
	for _, tt := range tests {

		mfData, err := os.ReadFile(tt.args.mfFileName)
		if err != nil {
			t.Logf("failed to read file %s", err)
			t.FailNow()
		}
		mockReader := MockReader{expectedData: mfData}

		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(mockReader)
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
