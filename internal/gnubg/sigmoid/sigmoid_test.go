package sigmoid

import "testing"

func Test_Sigmoid(t *testing.T) {
	type args struct {
		xin float32
	}
	tests := []struct {
		name string
		args args
		want float32
	}{
		// function produces reverse sigmoid
		{"should calculate -10", args{-10}, 0.9999498},
		{"should calculate -1", args{-1}, 0.7310586},
		{"should calculate 0", args{0}, 0.5},
		{"should calculate 1", args{1}, 0.26894143},
		{"should calculate 10", args{10}, 5.0172166e-05},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sigmoid(tt.args.xin); got != tt.want {
				t.Errorf("Sigmoid() = %v, want %v", got, tt.want)
			}
		})
	}
}
