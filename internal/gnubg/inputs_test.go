package gnubg

import "testing"

func Test_baseInputs(t *testing.T) {
	type args struct {
		anBoard _TanBoard
		arInput []float32
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "should set base inputs",
			args: args{
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
				arInput: make([]float32, _NUM_INPUTS),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseInputs(tt.args.anBoard, tt.args.arInput)
		})
	}
}
