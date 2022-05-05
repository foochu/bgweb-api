package api

import (
	"bgweb-api/internal/gnubg"
	"bgweb-api/internal/openapi"
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
		args openapi.MoveArgs
	}
	tests := []struct {
		name    string
		args    args
		want    []openapi.Move
		wantErr bool
	}{
		{
			name: "should get 3-1 with scores",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					X: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(3), N13: toPtr(5), N24: toPtr(2)},
					O: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(3), N13: toPtr(5), N24: toPtr(2)},
				},
				Dice:       []int{3, 1},
				Player:     "x",
				MaxMoves:   toPtr(3),
				ScoreMoves: toPtr(true),
				Cubeful:    toPtr(true),
			}},
			want: []openapi.Move{
				{
					Play: &[]openapi.CheckerPlay{{From: "8", To: "5"}, {From: "6", To: "5"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: true, Plies: 1},
						Eq:          0.218,
						Diff:        0,
						Probability: &openapi.Probability{Win: 0.551, WinG: 0.174, WinBG: 0.013, Lose: 0.449, LoseG: 0.124, LoseBG: 0.005},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "13", To: "10"}, {From: "24", To: "23"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: true, Plies: 1},
						Eq:          -0.012,
						Diff:        -0.23,
						Probability: &openapi.Probability{Win: 0.497, WinG: 0.137, WinBG: 0.008, Lose: 0.503, LoseG: 0.14, LoseBG: 0.007},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "24", To: "21"}, {From: "21", To: "20"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: true, Plies: 1},
						Eq:          -0.020,
						Diff:        -0.239,
						Probability: &openapi.Probability{Win: 0.497, WinG: 0.125, WinBG: 0.005, Lose: 0.503, LoseG: 0.135, LoseBG: 0.004},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should get 3-1 without scores",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					O: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(3), N13: toPtr(5), N24: toPtr(2)},
					X: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(3), N13: toPtr(5), N24: toPtr(2)},
				},
				Dice:       []int{3, 1},
				Player:     "x",
				MaxMoves:   toPtr(3),
				ScoreMoves: toPtr(false),
			}},
			want: []openapi.Move{
				{Play: &[]openapi.CheckerPlay{{From: "24", To: "21"}, {From: "24", To: "23"}}},
				{Play: &[]openapi.CheckerPlay{{From: "24", To: "21"}, {From: "21", To: "20"}}},
				{Play: &[]openapi.CheckerPlay{{From: "24", To: "21"}, {From: "8", To: "7"}}},
			},
			wantErr: false,
		},
		{
			name: "should consider player on bar",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					O: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(4), N13: toPtr(4), N15: toPtr(1), N24: toPtr(1)},
					X: openapi.CheckerLayout{N6: toPtr(5), N7: toPtr(2), N8: toPtr(3), N13: toPtr(2), N24: toPtr(2), Bar: toPtr(1)},
				},
				Dice:       []int{6, 1},
				Player:     "x",
				MaxMoves:   toPtr(9),
				ScoreMoves: toPtr(false),
			}},
			want: []openapi.Move{
				{Play: &[]openapi.CheckerPlay{{From: "bar", To: "24"}, {From: "24", To: "18"}}},
				{Play: &[]openapi.CheckerPlay{{From: "bar", To: "24"}, {From: "13", To: "7"}}},
				{Play: &[]openapi.CheckerPlay{{From: "bar", To: "24"}, {From: "8", To: "2"}}},
				{Play: &[]openapi.CheckerPlay{{From: "bar", To: "24"}, {From: "7", To: "1"}}},
			},
			wantErr: false,
		},
		{
			name: "should bear off",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					X: openapi.CheckerLayout{N1: toPtr(1)},
					O: openapi.CheckerLayout{N2: toPtr(1)},
				},
				Dice:       []int{6, 1},
				Player:     "x",
				MaxMoves:   toPtr(9),
				ScoreMoves: toPtr(false),
			}},
			want: []openapi.Move{
				{Play: &[]openapi.CheckerPlay{{From: "1", To: "off"}}},
			},
			wantErr: false,
		},
		{
			name: "should save gammon",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					X: openapi.CheckerLayout{N1: toPtr(1)},
					O: openapi.CheckerLayout{N1: toPtr(4), N2: toPtr(3), N3: toPtr(1), N4: toPtr(2), N5: toPtr(2), N6: toPtr(3)},
				},
				Dice:       []int{4, 1},
				Player:     "o",
				MaxMoves:   toPtr(3),
				ScoreMoves: toPtr(true),
			}},
			want: []openapi.Move{
				{
					Play: &[]openapi.CheckerPlay{{From: "4", To: "off"}, {From: "3", To: "2"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          -1,
						Diff:        0,
						Probability: &openapi.Probability{Win: 0, WinG: 0, WinBG: 0, Lose: 1, LoseG: 0, LoseBG: 0},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "4", To: "off"}, {From: "6", To: "5"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          -1,
						Diff:        0,
						Probability: &openapi.Probability{Win: 0, WinG: 0, WinBG: 0, Lose: 1, LoseG: 0, LoseBG: 0},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "6", To: "2"}, {From: "1", To: "off"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          -1,
						Diff:        0,
						Probability: &openapi.Probability{Win: 0, WinG: 0, WinBG: 0, Lose: 1, LoseG: 0, LoseBG: 0},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should score x",
			args: args{openapi.MoveArgs{
				Board: openapi.Board{
					X: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(4), N13: toPtr(4), N23: toPtr(1), N24: toPtr(1)},
					O: openapi.CheckerLayout{N6: toPtr(5), N8: toPtr(3), N13: toPtr(5), N21: toPtr(1), N24: toPtr(1)},
				},
				Dice:       []int{6, 5},
				Player:     "x",
				MaxMoves:   toPtr(3),
				ScoreMoves: toPtr(true),
			}},
			want: []openapi.Move{
				{
					Play: &[]openapi.CheckerPlay{{From: "24", To: "18"}, {From: "23", To: "18"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          0.159,
						Diff:        0,
						Probability: &openapi.Probability{Win: 0.564, WinG: 0.108, WinBG: 0.003, Lose: 0.436, LoseG: 0.078, LoseBG: 0.001},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "24", To: "18"}, {From: "18", To: "13"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          0.044,
						Diff:        -0.115,
						Probability: &openapi.Probability{Win: 0.527, WinG: 0.109, WinBG: 0.002, Lose: 0.473, LoseG: 0.118, LoseBG: 0.003},
					},
				},
				{
					Play: &[]openapi.CheckerPlay{{From: "24", To: "18"}, {From: "13", To: "8"}},
					Evaluation: &openapi.Evaluation{
						Info:        &openapi.EvalInfo{Cubeful: false, Plies: 3},
						Eq:          -0.013,
						Diff:        -0.172,
						Probability: &openapi.Probability{Win: 0.5, WinG: 0.124, WinBG: 0.005, Lose: 0.5, LoseG: 0.136, LoseBG: 0.004},
					},
				},
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
				if !reflect.DeepEqual(got[i].Play, m.Play) {
					t.Errorf("GetMoves(%d).Play = %v, want %v}", i, got[i].Play, m.Play)
				}
				if !reflect.DeepEqual(got[i].Evaluation, m.Evaluation) {
					t.Errorf("GetMoves(%d).Evaluation = %v, want %v", i, got[i].Evaluation, m.Evaluation)
				}
			}
		})
	}
}
