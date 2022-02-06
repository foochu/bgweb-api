package gnubg

import (
	"math"
	"os"
	"reflect"
	"sync"
	"testing"
)

var once sync.Once

func setup() {
	dataDir := os.DirFS("../../cmd/bgweb-api/data")
	initMatchEquity(dataDir, "met/Kazaross-XG2.xml")
	if err := evalInitialise(dataDir); err != nil {
		panic(err)
	}
}

func Test_generateMoves(t *testing.T) {
	once.Do(setup)
	type args struct {
		board    _TanBoard
		n0, n1   int
		fPartial bool
	}
	tests := []struct {
		name string
		args args
		want [][8]int
	}{
		{"should find initial moves for 3-1", args{
			_TanBoard{
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			}, 3, 1, false,
		}, [][8]int{
			{23, 20, 23, 22, -1, 0, 0, 0},
			{23, 20, 20, 19, -1, 0, 0, 0},
			{23, 20, 7, 6, -1, 0, 0, 0},
			{23, 20, 5, 4, -1, 0, 0, 0},
			{12, 9, 23, 22, -1, 0, 0, 0},
			{12, 9, 9, 8, -1, 0, 0, 0},
			{12, 9, 7, 6, -1, 0, 0, 0},
			{12, 9, 5, 4, -1, 0, 0, 0},
			{7, 4, 23, 22, -1, 0, 0, 0},
			{7, 4, 7, 6, -1, 0, 0, 0},
			{7, 4, 5, 4, -1, 0, 0, 0},
			{7, 4, 4, 3, -1, 0, 0, 0},
			{5, 2, 23, 22, -1, 0, 0, 0},
			{5, 2, 7, 6, -1, 0, 0, 0},
			{5, 2, 5, 4, -1, 0, 0, 0},
			{5, 2, 2, 1, -1, 0, 0, 0},
		}},
		{"should find moves for 6-6 after 3-1", args{
			_TanBoard{
				{0, 0, 0, 0, 2, 4, 0, 2, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			}, 6, 6, false,
		}, [][8]int{
			{23, 17, 23, 17, 12, 6, 12, 6},
			{23, 17, 23, 17, 12, 6, 7, 1},
			{23, 17, 23, 17, 7, 1, 7, 1},
			{23, 17, 12, 6, 12, 6, 12, 6},
			{23, 17, 12, 6, 12, 6, 7, 1},
			{23, 17, 12, 6, 7, 1, 7, 1},
			{23, 17, 7, 1, 7, 1, 7, 1},
			{12, 6, 12, 6, 12, 6, 12, 6},
			{12, 6, 12, 6, 12, 6, 7, 1},
			{12, 6, 12, 6, 7, 1, 7, 1},
			{12, 6, 7, 1, 7, 1, 7, 1},
		}},
		{"should find moves on bar for 5-2", args{
			_TanBoard{
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
				{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1},
			}, 5, 2, false,
		}, [][8]int{
			{24, 19, 23, 21, -1, 0, 0, 0},
			{24, 19, 22, 20, -1, 0, 0, 0},
			{24, 19, 19, 17, -1, 0, 0, 0},
			{24, 19, 12, 10, -1, 0, 0, 0},
			{24, 19, 7, 5, -1, 0, 0, 0},
			{24, 19, 5, 3, -1, 0, 0, 0},
			{24, 22, 12, 7, -1, 0, 0, 0},
			{24, 22, 7, 2, -1, 0, 0, 0},
			{24, 22, 5, 0, -1, 0, 0, 0},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tld := _ThreadLocalData{}
			pml := _MoveList{}
			got := generateMoves(&tld, &pml, tt.args.board, tt.args.n0, tt.args.n1, tt.args.fPartial)
			if got != len(tt.want) {
				t.Errorf("generateMoves() = %v, want %v", got, len(tt.want))
			}
			for i := 0; i < got; i++ {
				if pml.amMoves[i].anMove != tt.want[i] {
					t.Errorf("pml.amMoves[%v] = %v, want %v", i, pml.amMoves[i].anMove, tt.want[i])
				}
			}
		})
	}
}

func Test_scoreMoves(t *testing.T) {
	once.Do(setup)
	type args struct {
		moves []_Move
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"should score", args{
			moves: []_Move{{
				anMove: [8]int{23, 20, 23, 22, -1, 0, 0, 0},
				key: _PositionKey{
					data: [7]int{810549248, 372680, 16842752, 810549248, 327680, 536870912, 0},
				},
				cMoves:     2,
				cPips:      4,
				rScore:     0,
				rScore2:    0,
				arEvalMove: [7]float32{0, 0, 0, 0, 0, 0, 0},
				cmark:      _CMARK_NONE,
			}},
		}, false},
	}
	for _, tt := range tests {
		pml := _MoveList{
			cMoves:  len(tt.args.moves),
			amMoves: tt.args.moves,
		}
		pci := _CubeInfo{}
		pec := _EvalContext{fCubeful: true}
		t.Run(tt.name, func(t *testing.T) {
			tld := _ThreadLocalData{}
			err := scoreMoves(&tld, &pml, &pci, &pec, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("scoreMoves() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_findnSaveBestMoves(t *testing.T) {
	once.Do(setup)
	var eval = func(nDice0 int, nDice1 int, anBoard _TanBoard) (*_Move, error) {
		var pml = _MoveList{}
		var pci = _CubeInfo{
			nCube:         1,
			fCubeOwner:    -1,
			fMove:         1,
			nMatchTo:      0,
			anScore:       [2]int{0, 0},
			fCrawford:     false,
			fJacoby:       true,
			fBeavers:      true,
			arGammonPrice: [4]float32{0, 0, 0, 0},
			bgv:           _VARIATION_STANDARD,
		}
		var pec = _EvalContext{
			fCubeful:       true,
			nPlies:         2,
			fUsePrune:      true,
			fDeterministic: true,
			rNoise:         0,
		}
		var aamf = &_MOVEFILTER_NORMAL
		if err := findnSaveBestMoves(&pml, nDice0, nDice1, anBoard, nil, 0, &pci, &pec, aamf); err != nil {
			return nil, err
		}
		bestMove := pml.amMoves[pml.iMoveBest]
		return &bestMove, nil
	}
	type args struct {
		nDice0  int
		nDice1  int
		anBoard _TanBoard
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantMove _Move
	}{
		{
			name: "should find best moves for 3-1",
			args: args{
				nDice0: 3,
				nDice1: 1,
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
			},
			wantMove: _Move{anMove: [8]int{7, 4, 5, 4, -1}, rScore: 0.21818662},
		},
		{
			name: "should find best moves for 6-2",
			args: args{
				nDice0: 6,
				nDice1: 2,
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
			},
			wantMove: _Move{anMove: [8]int{23, 17, 12, 10, -1}, rScore: 0.019569507},
		},
		// {
		// 	name: "should find best moves for 6-3 after 6-5",
		// 	args: args{
		// 		nDice0: 6,
		// 		nDice1: 3,
		// 		anBoard: _TanBoard{
		// 			{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0},
		// 			{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0},
		// 		},
		// 	},
		// 	wantMove: _Move{anMove: [8]int{23, 17, 12, 9, -1}, rScore: -0.018},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			move, err := eval(tt.args.nDice0, tt.args.nDice1, tt.args.anBoard)
			if (err != nil) != tt.wantErr {
				t.Errorf("findnSaveBestMoves() error = %v, wantErr %v", err, tt.wantErr)
			}
			if move.anMove != tt.wantMove.anMove {
				t.Errorf("findnSaveBestMoves() move.anMove = %v, want %v", move.anMove, tt.wantMove.anMove)
			}
			if move.rScore != tt.wantMove.rScore {
				t.Errorf("findnSaveBestMoves() move.rScore = %v, want %v", move.rScore, tt.wantMove.rScore)
			}
		})
	}
}

func Test_evaluatePositionFull(t *testing.T) {
	once.Do(setup)
	type args struct {
		nnStates *[3]_NNState
		anBoard  _TanBoard
		arOutput *[_NUM_OUTPUTS]float32
		pci      *_CubeInfo
		pec      *_EvalContext
		nPlies   int
		pc       _PositionClass
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should evaluate",
			args: args{
				nnStates: &[3]_NNState{},
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
				arOutput: &[_NUM_OUTPUTS]float32{},
				pci:      &_CubeInfo{},
				pec: &_EvalContext{
					nPlies: 2,
				},
				nPlies: 2,
				pc:     _CLASS_RACE,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tld := _ThreadLocalData{}
			if err := evaluatePositionFull(&tld, tt.args.nnStates, tt.args.anBoard, tt.args.arOutput, tt.args.pci, tt.args.pec, tt.args.nPlies, tt.args.pc); (err != nil) != tt.wantErr {
				t.Errorf("evaluatePositionFull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_msb32(t *testing.T) {
	once.Do(setup)
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should find 32",
			args: args{32},
			want: 5,
		},
		{
			name: "should find 4096",
			args: args{4096},
			want: 12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := msb32(tt.args.n); got != tt.want {
				t.Errorf("msb32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_evalContact(t *testing.T) {
	once.Do(setup)
	type args struct {
		anBoard _TanBoard
	}
	tests := []struct {
		name string
		args args
		want [_NUM_OUTPUTS]float32
	}{
		{
			name: "should eval 3-1",
			args: args{
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
			},
			want: [5]float32{0.504503, 0.134615, 0.004168, 0.123002, 0.005468},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arOutput [5]float32
			evalContact(tt.args.anBoard, &arOutput, _VARIATION_STANDARD, nil)
			for i := range arOutput {
				// strip to 6 decimal places
				arOutput[i] = float32(math.Round(float64(arOutput[i]*1000000))) / 1000000
			}
			if !reflect.DeepEqual(arOutput, tt.want) {
				t.Errorf("evalContact() arOutput = %v, want %v", arOutput, tt.want)
			}
		})
	}
}

func Test_calculateContactInputs(t *testing.T) {
	once.Do(setup)
	type args struct {
		anBoard _TanBoard
	}
	tests := []struct {
		name string
		args args
		want []float32
	}{
		{
			name: "should calculate",
			args: args{
				anBoard: _TanBoard{
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0},
					{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				},
			},
			want: []float32{
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 1.000000, 1.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 1.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 1.000000, 1.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				1.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				1.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 1.000000, 1.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 1.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 1.000000, 1.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 1.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000,
				0.000000, 0.000000, 0.000000, 0.910180, 0.958333, 0.958333, 0.166667, 0.240741,
				0.722222, 0.055556, 0.611111, 0.388889, 0.151235, 0.388889, 0.151235, 0.640278,
				0.105000, 0.000000, 0.305556, 0.520000, 0.939394, 0.000000, 0.250000, 0.000000,
				0.527778, 0.000000, 0.000000, 0.000000, 0.976048, 0.916667, 0.500000, 2.000000,
				0.000000, 0.000000, 0.000000, 0.666667, 0.388889, 0.151235, 0.388889, 0.151235,
				0.588611, 0.095000, 0.000000, 0.305556, 0.940000, 0.227273, 0.000000, 0.000000,
				0.000000, 0.500000,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arInput = make([]float32, 250)
			calculateContactInputs(tt.args.anBoard, arInput)
			for i := range arInput {
				// strip to 6 decimal places
				arInput[i] = float32(math.Round(float64(arInput[i]*1000000))) / 1000000
			}
			if !reflect.DeepEqual(arInput, tt.want) {
				t.Errorf("calculateContactInputs() arInput = %v, want %v", arInput, tt.want)
			}
		})
	}
}

func Test_calculateHalfInputs(t *testing.T) {
	once.Do(setup)
	type args struct {
		anBoard    [25]int
		anBoardOpp [25]int
		afInput    []float32
	}
	tests := []struct {
		name string
		args args
		want []float32
	}{
		{
			name: "should calculate",
			args: args{
				anBoard:    [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				anBoardOpp: [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0},
				afInput:    []float32{0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000, 0.000000},
			},
			want: []float32{0.000000, 0.000000, 0.000000, 0.910180, 0.958333, 0.958333, 0.166667, 0.240741, 0.722222, 0.055556, 0.611111, 0.388889, 0.151235, 0.388889, 0.151235, 0.640278, 0.105000, 0.000000, 0.305556, 0.520000, 0.939394, 0.000000, 0.250000, 0.000000, 0.527778},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arInput := tt.args.afInput
			calculateHalfInputs(tt.args.anBoard, tt.args.anBoardOpp, arInput)
			for i := range arInput {
				// strip to 6 decimal places
				arInput[i] = float32(math.Round(float64(arInput[i]*1000000))) / 1000000
			}
			if !reflect.DeepEqual(arInput, tt.want) {
				t.Errorf("calculateContactInputs() arInput = %v, want %v", arInput, tt.want)
			}
		})
	}
}

func Test_escapes(t *testing.T) {
	once.Do(setup)
	type args struct {
		anBoard [25]int
		n       int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "should evaluate",
			args: args{
				anBoard: [25]int{0, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				n:       22,
			},
			want: 22,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapes(tt.args.anBoard, tt.args.n); got != tt.want {
				t.Errorf("escapes() = %v, want %v", got, tt.want)
			}
		})
	}
}
