package api

import (
	"bgweb-api/internal/gnubg"
	"os"
	"reflect"
	"sync"
	"testing"
)

var once sync.Once

func setup() {
	if err := gnubg.Init(os.DirFS("../../cmd/bgweb-api/data")); err != nil {
		panic(err)
	}
}

func TestGetMoves(t *testing.T) {
	once.Do(setup)
	type args struct {
		args MoveArgs
	}
	tests := []struct {
		name    string
		args    args
		want    []Move
		wantErr bool
	}{
		{
			name: "should get 3-1 with scores",
			args: args{MoveArgs{
				Board: Board{
					X: CheckerLayout{P6: 5, P8: 3, P13: 5, P24: 2},
					O: CheckerLayout{P6: 5, P8: 3, P13: 5, P24: 2},
				},
				Dice:       [2]int{3, 1},
				Player:     "x",
				MaxMoves:   9,
				ScoreMoves: true,
				Cubeful:    true,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"8", "5"}, {"6", "5"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: 0.218, EquityDiff: 0, Probablity: Probablity{0.551, 0.174, 0.013, 0.449, 0.124, 0.005}}},
				{Play: []CheckerPlay{{"13", "10"}, {"24", "23"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.012, EquityDiff: -0.23, Probablity: Probablity{0.497, 0.137, 0.008, 0.503, 0.14, 0.007}}},
				{Play: []CheckerPlay{{"24", "21"}, {"21", "20"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.020, EquityDiff: -0.239, Probablity: Probablity{0.497, 0.125, 0.005, 0.503, 0.135, 0.004}}},
				{Play: []CheckerPlay{{"24", "21"}, {"24", "23"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.026, EquityDiff: -0.244, Probablity: Probablity{0.495, 0.123, 0.005, 0.505, 0.135, 0.004}}},
				{Play: []CheckerPlay{{"13", "10"}, {"10", "9"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.026, EquityDiff: -0.244, Probablity: Probablity{0.491, 0.143, 0.009, 0.509, 0.143, 0.008}}},
				{Play: []CheckerPlay{{"24", "21"}, {"6", "5"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.049, EquityDiff: -0.267, Probablity: Probablity{0.493, 0.125, 0.007, 0.507, 0.146, 0.007}}},
				{Play: []CheckerPlay{{"13", "10"}, {"6", "5"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.053, EquityDiff: -0.271, Probablity: Probablity{0.489, 0.138, 0.01, 0.511, 0.154, 0.012}}},
				{Play: []CheckerPlay{{"6", "3"}, {"24", "23"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.108, EquityDiff: -0.326, Probablity: Probablity{0.477, 0.125, 0.007, 0.523, 0.157, 0.008}}},
				{Play: []CheckerPlay{{"8", "5"}, {"24", "23"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: true, Plies: 1}, Equity: -0.109, EquityDiff: -0.328, Probablity: Probablity{0.477, 0.124, 0.007, 0.523, 0.156, 0.009}}},
			},
			wantErr: false,
		},
		{
			name: "should get 3-1 without scores",
			args: args{MoveArgs{
				Board: Board{
					O: CheckerLayout{P6: 5, P8: 3, P13: 5, P24: 2},
					X: CheckerLayout{P6: 5, P8: 3, P13: 5, P24: 2},
				},
				Dice:       [2]int{3, 1},
				Player:     "x",
				MaxMoves:   9,
				ScoreMoves: false,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"24", "21"}, {"24", "23"}}},
				{Play: []CheckerPlay{{"24", "21"}, {"21", "20"}}},
				{Play: []CheckerPlay{{"24", "21"}, {"8", "7"}}},
				{Play: []CheckerPlay{{"24", "21"}, {"6", "5"}}},
				{Play: []CheckerPlay{{"13", "10"}, {"24", "23"}}},
				{Play: []CheckerPlay{{"13", "10"}, {"10", "9"}}},
				{Play: []CheckerPlay{{"13", "10"}, {"8", "7"}}},
				{Play: []CheckerPlay{{"13", "10"}, {"6", "5"}}},
				{Play: []CheckerPlay{{"8", "5"}, {"24", "23"}}},
			},
			wantErr: false,
		},
		{
			name: "should consider player on bar",
			args: args{MoveArgs{
				Board: Board{
					O: CheckerLayout{P6: 5, P8: 4, P13: 4, P15: 1, P24: 1},
					X: CheckerLayout{P6: 5, P7: 2, P8: 3, P13: 2, P24: 2, Bar: 1},
				},
				Dice:       [2]int{6, 1},
				Player:     "x",
				MaxMoves:   9,
				ScoreMoves: false,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"bar", "24"}, {"24", "18"}}},
				{Play: []CheckerPlay{{"bar", "24"}, {"13", "7"}}},
				{Play: []CheckerPlay{{"bar", "24"}, {"8", "2"}}},
				{Play: []CheckerPlay{{"bar", "24"}, {"7", "1"}}},
			},
			wantErr: false,
		},
		{
			name: "should bear off",
			args: args{MoveArgs{
				Board: Board{
					X: CheckerLayout{P1: 1},
					O: CheckerLayout{P2: 1},
				},
				Dice:       [2]int{6, 1},
				Player:     "x",
				MaxMoves:   9,
				ScoreMoves: false,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"1", "off"}}},
			},
			wantErr: false,
		},
		{
			name: "should save gammon",
			args: args{MoveArgs{
				Board: Board{
					X: CheckerLayout{P1: 1},
					O: CheckerLayout{P1: 4, P2: 3, P3: 1, P4: 2, P5: 2, P6: 3},
				},
				Dice:       [2]int{4, 1},
				Player:     "o",
				MaxMoves:   9,
				ScoreMoves: true,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"4", "off"}, {"3", "2"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"4", "off"}, {"6", "5"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"6", "2"}, {"1", "off"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"4", "off"}, {"2", "1"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"5", "1"}, {"1", "off"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"4", "off"}, {"1", "off"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"4", "off"}, {"4", "3"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -1, EquityDiff: 0, Probablity: Probablity{0, 0, 0, 1, 0, 0}}},
				{Play: []CheckerPlay{{"6", "2"}, {"4", "3"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -2, EquityDiff: -1, Probablity: Probablity{0, 0, 0, 1, 1, 0}}},
				{Play: []CheckerPlay{{"6", "2"}, {"2", "1"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -2, EquityDiff: -1, Probablity: Probablity{0, 0, 0, 1, 1, 0}}},
			},
			wantErr: false,
		},
		{
			name: "should score x",
			args: args{MoveArgs{
				Board: Board{
					X: CheckerLayout{P6: 5, P8: 4, P13: 4, P23: 1, P24: 1},
					O: CheckerLayout{P6: 5, P8: 3, P13: 5, P21: 1, P24: 1},
				},
				Dice:       [2]int{6, 5},
				Player:     "x",
				MaxMoves:   9,
				ScoreMoves: true,
			}},
			want: []Move{
				{Play: []CheckerPlay{{"24", "18"}, {"23", "18"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: 0.159, EquityDiff: 0, Probablity: Probablity{0.564, 0.108, 0.003, 0.436, 0.078, 0.001}}},
				{Play: []CheckerPlay{{"24", "18"}, {"18", "13"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: 0.044, EquityDiff: -0.115, Probablity: Probablity{0.527, 0.109, 0.002, 0.473, 0.118, 0.003}}},
				{Play: []CheckerPlay{{"24", "18"}, {"13", "8"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 3}, Equity: -0.013, EquityDiff: -0.172, Probablity: Probablity{0.5, 0.124, 0.005, 0.5, 0.136, 0.004}}},
				{Play: []CheckerPlay{{"24", "18"}, {"6", "1"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.056, EquityDiff: -0.215, Probablity: Probablity{0.483, 0.116, 0.003, 0.517, 0.136, 0.006}}},
				{Play: []CheckerPlay{{"13", "7"}, {"6", "1"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.057, EquityDiff: -0.216, Probablity: Probablity{0.476, 0.128, 0.005, 0.524, 0.134, 0.008}}},
				{Play: []CheckerPlay{{"13", "7"}, {"7", "2"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.086, EquityDiff: -0.245, Probablity: Probablity{0.466, 0.126, 0.005, 0.534, 0.141, 0.007}}},
				{Play: []CheckerPlay{{"8", "2"}, {"23", "18"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.105, EquityDiff: -0.264, Probablity: Probablity{0.466, 0.116, 0.004, 0.534, 0.151, 0.007}}},
				{Play: []CheckerPlay{{"13", "7"}, {"23", "18"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.106, EquityDiff: -0.265, Probablity: Probablity{0.465, 0.115, 0.004, 0.535, 0.148, 0.007}}},
				{Play: []CheckerPlay{{"24", "18"}, {"8", "3"}}, Evaluation: &Evaluation{Info: EvalInfo{Cubeful: false, Plies: 1}, Equity: -0.108, EquityDiff: -0.267, Probablity: Probablity{0.466, 0.116, 0.004, 0.534, 0.152, 0.007}}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMoves(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMoves() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("GetMoves() = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, m := range tt.want {
				if !reflect.DeepEqual(got[i], m) {
					t.Errorf("GetMoves(%d) = %v (%v), want %v (%v)", i, got[i], got[i].Evaluation, m, m.Evaluation)
				}
			}
		})
	}
}
