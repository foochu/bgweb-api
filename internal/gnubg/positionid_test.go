package gnubg

import (
	"reflect"
	"testing"
)

func Test_positionBearoff(t *testing.T) {
	type args struct {
		anBoard   [6]int
		nPoints   int
		nChequers int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should get 100000",
			args: args{
				anBoard:   [6]int{1, 0, 0, 0, 0, 0},
				nPoints:   1,
				nChequers: 1,
			},
			want: 1,
		},
		{
			name: "should get 222000",
			args: args{
				anBoard:   [6]int{2, 2, 2, 0, 0, 0},
				nPoints:   3,
				nChequers: 6,
			},
			want: 68,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := positionBearoff(tt.args.anBoard, tt.args.nPoints, tt.args.nChequers); got != tt.want {
				t.Errorf("positionBearoff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_PositionKey_fromBoard(t *testing.T) {
	type args struct {
		anBoard _TanBoard
	}
	tests := []struct {
		name string
		args args
		want _PositionKey
	}{
		{
			name: "should get initial",
			args: args{_TanBoard{
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			}},
			want: _PositionKey{[7]int{810549248, 327680, 536870912, 810549248, 327680, 536870912, 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := _PositionKey{}
			tr.fromBoard(tt.args.anBoard)
			if !reflect.DeepEqual(tr, tt.want) {
				t.Errorf("PositionKey_fromBoard() = %v, want %v", tr, tt.want)
			}
		})
	}
}
