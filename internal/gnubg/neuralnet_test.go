package gnubg

import (
	"os"
	"testing"
)

func openFile(filename string) *os.File {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	if err = verifyWeights(f, filename); err != nil {
		panic(err)
	}
	return f
}

func Test_neuralNetLoad(t *testing.T) {
	pf := openFile("../../cmd/bgweb-api/data/gnubg.weights")
	defer pf.Close()
	type args struct {
		pnn *_NeuralNet
		pf  *os.File
	}
	type bounds struct {
		len   int
		first float32
		last  float32
	}
	tests := []struct {
		name                string
		args                args
		wantErr             bool
		wantHiddenWeight    bounds
		wantOutputWeight    bounds
		wantHiddenThreshold bounds
		wantOutputThreshold bounds
	}{
		{
			name: "should load",
			args: args{
				pnn: &_NeuralNet{},
				pf:  pf,
			},
			wantErr:             false,
			wantHiddenWeight:    bounds{32000, -3.2585607, -0.872228},
			wantOutputWeight:    bounds{640, 7.094037, 27.94653},
			wantHiddenThreshold: bounds{128, -23.102942, -39.628853},
			wantOutputThreshold: bounds{5, 0.0657336, -1.7618835},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := neuralNetLoad(tt.args.pnn, tt.args.pf)
			if (err != nil) != tt.wantErr {
				t.Errorf("neuralNetLoad() error = %v, wantErr %v", err, tt.wantErr)
			}
			check := func(ar []float32, s bounds, ctx string) {
				var len, first, last = len(ar), ar[0], ar[len(ar)-1]
				if len != s.len {
					t.Errorf("neuralNetLoad() [%v] len: %v, wantLen: %v", ctx, len, s.len)
				}
				if first != s.first {
					t.Errorf("neuralNetLoad() [%v] first: %v, wantFirst: %v", ctx, first, s.first)
				}
				if last != s.last {
					t.Errorf("neuralNetLoad() [%v] last: %v, wantLast: %v", ctx, last, s.last)
				}
			}
			check(tt.args.pnn.arHiddenWeight, tt.wantHiddenWeight, "arHiddenWeight")
			check(tt.args.pnn.arOutputWeight, tt.wantOutputWeight, "arOutputWeight")
			check(tt.args.pnn.arHiddenThreshold, tt.wantHiddenThreshold, "arHiddenThreshold")
			check(tt.args.pnn.arOutputThreshold, tt.wantOutputThreshold, "arOutputThreshold")
		})
	}
}

func Test_neuralNetEvaluate(t *testing.T) {
	pf := openFile("../../cmd/bgweb-api/data/gnubg.weights")
	defer pf.Close()
	pnn := _NeuralNet{}
	if err := neuralNetLoad(&pnn, pf); err != nil {
		panic(err)
	}
	pnState := _NNState{}
	type args struct {
		pnn      *_NeuralNet
		arInput  *[_NUM_INPUTS]float32
		arOutput *[_NUM_OUTPUTS]float32
		pnState  *_NNState
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should evaluate stateless",
			args: args{
				pnn:      &pnn,
				arInput:  &[_NUM_INPUTS]float32{0.0},
				arOutput: &[_NUM_OUTPUTS]float32{0.0},
				pnState:  nil,
			},
			wantErr: false,
		},
		{
			name: "should evaluate initial state",
			args: args{
				pnn:      &pnn,
				arInput:  &[_NUM_INPUTS]float32{0.0},
				arOutput: &[_NUM_OUTPUTS]float32{0.0},
				pnState:  &pnState,
			},
			wantErr: false,
		},
		{
			name: "should evaluate saved state",
			args: args{
				pnn:      &pnn,
				arInput:  &[_NUM_INPUTS]float32{0.0},
				arOutput: &[_NUM_OUTPUTS]float32{0.0},
				pnState:  &pnState,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := neuralNetEvaluate(tt.args.pnn, tt.args.arInput, tt.args.arOutput, tt.args.pnState); (err != nil) != tt.wantErr {
				t.Errorf("neuralNetEvaluate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
