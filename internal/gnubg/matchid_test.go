package gnubg

import (
	"testing"
)

func Test_logCube(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"Should calculate", args{0}, 0},
		{"Should calculate", args{2}, 1},
		{"Should calculate", args{4}, 2},
		{"Should calculate", args{8}, 3},
		{"Should calculate", args{16}, 4},
		{"Should calculate", args{32}, 5},
		{"Should calculate", args{64}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logCube(tt.args.n); got != tt.want {
				t.Errorf("logCube() = %v, want %v", got, tt.want)
			}
		})
	}
}
