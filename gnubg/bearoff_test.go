package gnubg

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
)

func Test_bearoffInit(t *testing.T) {
	type args struct {
		dataDir    fs.FS
		szFilename string
		bo         int
	}
	tests := []struct {
		name    string
		args    args
		want    *_BearOffContext
		wantErr bool
	}{
		{
			name: "should init one-sided",
			args: args{
				dataDir:    os.DirFS("../data"),
				szFilename: "gnubg_os0.bd",
				bo:         _BO_MUST_BE_ONE_SIDED,
			},
			want: &_BearOffContext{
				bt:          _BEAROFF_ONESIDED,
				nPoints:     6,
				nChequers:   15,
				fCompressed: true,
				fGammon:     true,
				fND:         false,
				fHeuristic:  false,
				fCubeful:    false,
				szFilename:  "gnubg_os0.bd",
				p:           nil,
			},
			wantErr: false,
		},
		{
			name: "should init two-sided",
			args: args{
				dataDir:    os.DirFS("../data"),
				szFilename: "gnubg_ts0.bd",
				bo:         _BO_MUST_BE_TWO_SIDED,
			},
			want: &_BearOffContext{
				bt:          _BEAROFF_TWOSIDED,
				nPoints:     6,
				nChequers:   6,
				fCompressed: false,
				fGammon:     false,
				fND:         false,
				fHeuristic:  false,
				fCubeful:    true,
				szFilename:  "gnubg_ts0.bd",
				p:           nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bearoffInit(tt.args.dataDir, tt.args.szFilename, tt.args.bo)
			if err != nil {
				bearoffClose(got)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("bearoffInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.p == nil {
				t.Error("bearoffInit() mmap is nil")
			}
			got.p = nil // blank out so can use deep equal for rest
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bearoffInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func Test_isBearoff(t *testing.T) {
// 	type args struct {
// 		pbc     *BearOffContext
// 		anBoard *TanBoard
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := isBearoff(tt.args.pbc, tt.args.anBoard); got != tt.want {
// 				t.Errorf("isBearoff() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func Test_bearoffDist(t *testing.T) {
	dataDir := os.DirFS("../data")
	pbc, err := bearoffInit(dataDir, "gnubg_os0.bd", _BO_MUST_BE_ONE_SIDED)
	if err != nil {
		panic(err)
	}
	defer bearoffClose(pbc)
	type args struct {
		pbc           *_BearOffContext
		nPosID        int
		arProb        *[32]float32
		arGammonProb  *[32]float32
		ar            *[4]float32
		ausProb       *[32]int
		ausGammonProb *[32]int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should get pos 1",
			args: args{
				pbc:     pbc,
				nPosID:  1,
				ausProb: &[32]int{},
			},
			wantErr: false,
		},
		{
			name: "should get pos 68",
			args: args{
				pbc:     pbc,
				nPosID:  68,
				ausProb: &[32]int{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := bearoffDist(tt.args.pbc, tt.args.nPosID, tt.args.arProb, tt.args.arGammonProb, tt.args.ar, tt.args.ausProb, tt.args.ausGammonProb); (err != nil) != tt.wantErr {
				t.Errorf("bearoffDist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_makeInt(t *testing.T) {
	type args struct {
		a byte
		b byte
		c byte
		d byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should make 1 byte",
			args: args{0x1, 0x0, 0x0, 0x0},
			want: 1,
		},
		{
			name: "should make 2 byte",
			args: args{0xc, 0x1, 0x0, 0x0},
			want: 268,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeInt(tt.args.a, tt.args.b, tt.args.c, tt.args.d); got != tt.want {
				t.Errorf("makeInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
