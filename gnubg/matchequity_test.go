package gnubg

import (
	"bgweb-api/gnubg/met"
	"testing"
)

func Test_readMET(t *testing.T) {
	type args struct {
		met      *met.METData
		filename string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Read Kazaross-XG2",
			args: args{met: &met.METData{}, filename: "../data/met/Kazaross-XG2.xml"},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readMET(tt.args.met, tt.args.filename); got != tt.want {
				t.Errorf("readMET() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initMatchEquity(t *testing.T) {
	type args struct {
		szFileName string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Read Kazaross-XG2",
			args: args{szFileName: "../data/met/Kazaross-XG2.xml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initMatchEquity(tt.args.szFileName)
			// t.Logf("aafMET: %v", aafMET)
			// t.Logf("aafMETPostCrawford: %v", aafMETPostCrawford)
			// t.Logf("aaaafGammonPrices: %v", aaaafGammonPrices)
			// t.Logf("aaaafGammonPricesPostCrawford: %v", aaaafGammonPricesPostCrawford)
			// t.Logf("miCurrent: %v", miCurrent)
		})
	}
}
