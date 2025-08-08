package microformats

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
)

type MockReader struct {
	expectedData  []byte
	expectedError error
}

func (m MockReader) Read(p []byte) (n int, err error) {
	fmt.Printf("\nexpectedData :: %v", len(m.expectedData))
	fmt.Printf("\nlen p :: %v", len(p))
	copy(p, m.expectedData)
	return len(m.expectedData), io.EOF
}

func TestParse(t *testing.T) {

	type args struct {
		mfFileName string
		r          io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    HugoPost
		wantErr bool
	}{
		{
			name: "hello",
			args: args{
				mfFileName: "photo.json",
			},
			want: HugoPost{
				Date: "2024-05-01T11:03:17+01:00",
				Tags: []string{"ldw", "hike"},
				Params: HugoPostParams{
					Photo:   "https://media.funabashi.co.uk/lg_20240501_110317_e70b8d3dbde23d49cfd88e5b251871d5.jpg",
					Caption: "Grimspound is a late Bronze Age settlement",
					Location: HugoPostLocation{
						Locality: "Yelverton",
						Region:   "Devon",
						Country:  "United Kingdom",
						Lat:      "50.611904",
						Lon:      "-3.836313",
					},
				},
			},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			mfData, err := os.ReadFile(tt.args.mfFileName)
			if err != nil {
				t.Errorf("failed to read file %s :: %v", tt.args.mfFileName, err)
				return
			}

			mockReader := &MockReader{expectedData: mfData}
			got, err := Parse(mockReader)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
