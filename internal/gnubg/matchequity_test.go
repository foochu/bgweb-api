package gnubg

import (
	"bgweb-api/internal/gnubg/met"
	"io/fs"
	"os"
	"testing"
)

func Test_readMET(t *testing.T) {
	type args struct {
		met      *met.METData
		dataDir  fs.FS
		filename string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Read Kazaross-XG2",
			args: args{
				met:      &met.METData{},
				dataDir:  os.DirFS("../../cmd/bgweb-api/data"),
				filename: "met/Kazaross-XG2.xml",
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readMET(tt.args.met, tt.args.dataDir, tt.args.filename); got != tt.want {
				t.Errorf("readMET() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initMatchEquity(t *testing.T) {
	type args struct {
		dataDir    fs.FS
		szFileName string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Read Kazaross-XG2",
			args: args{
				dataDir:    os.DirFS("../../cmd/bgweb-api/data"),
				szFileName: "met/Kazaross-XG2.xml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initMatchEquity(tt.args.dataDir, tt.args.szFileName)
			// t.Logf("aafMET: %v", aafMET)
			// t.Logf("aafMETPostCrawford: %v", aafMETPostCrawford)
			// t.Logf("aaaafGammonPrices: %v", aaaafGammonPrices)
			// t.Logf("aaaafGammonPricesPostCrawford: %v", aaaafGammonPricesPostCrawford)
			// t.Logf("miCurrent: %v", miCurrent)
		})
	}
}
