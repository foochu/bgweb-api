package api

import (
	"bgweb-api/internal/gnubg"
	"fmt"
	"math"
	"strconv"
)

type MoveArgs struct {
	Board      Board  `json:"board"`
	Dice       [2]int `json:"dice" swaggertype:"array,integer" example:"3,1"`
	Player     string `json:"player" enums:"x, o" default:"x" example:"x"`
	MaxMoves   int    `json:"max-moves" minimum:"0" default:"0"`
	ScoreMoves bool   `json:"score-moves" example:"true" default:"true"`
	Cubeful    bool
}

type Board struct {
	X CheckerLayout `json:"x"`
	O CheckerLayout `json:"o"`
}

type CheckerLayout struct {
	P1  int `json:"1,omitempty"`
	P2  int `json:"2,omitempty"`
	P3  int `json:"3,omitempty"`
	P4  int `json:"4,omitempty"`
	P5  int `json:"5,omitempty"`
	P6  int `json:"6,omitempty" example:"5"`
	P7  int `json:"7,omitempty"`
	P8  int `json:"8,omitempty" example:"3"`
	P9  int `json:"9,omitempty"`
	P10 int `json:"10,omitempty"`
	P11 int `json:"11,omitempty"`
	P12 int `json:"12,omitempty"`
	P13 int `json:"13,omitempty" example:"5"`
	P14 int `json:"14,omitempty"`
	P15 int `json:"15,omitempty"`
	P16 int `json:"16,omitempty"`
	P17 int `json:"17,omitempty"`
	P18 int `json:"18,omitempty"`
	P19 int `json:"19,omitempty"`
	P20 int `json:"20,omitempty"`
	P21 int `json:"21,omitempty"`
	P22 int `json:"22,omitempty"`
	P23 int `json:"23,omitempty"`
	P24 int `json:"24,omitempty" example:"2"`
	Bar int `json:"bar,omitempty"`
}

type CheckerPlay struct {
	From string `json:"from" enums:"1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,bar"`
	To   string `json:"to" enums:"1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,off"`
}

type Move struct {
	Play       []CheckerPlay `json:"play"`
	Evaluation *Evaluation   `json:"evaluation,omitempty"`
}

type Evaluation struct {
	Info       EvalInfo   `json:"info"`
	Equity     float32    `json:"eq"`
	EquityDiff float32    `json:"diff"`
	Probablity Probablity `json:"probability"`
}

type EvalInfo struct {
	Cubeful bool `json:"cubeful"`
	Plies   int  `json:"plies"`
}

type Probablity struct {
	Win    float32 `json:"win"`
	WinG   float32 `json:"winG"`
	WinBG  float32 `json:"winBG"`
	Lose   float32 `json:"lose"`
	LoseG  float32 `json:"loseG"`
	LoseBG float32 `json:"loseBG"`
}

func GetMoves(args MoveArgs) ([]Move, error) {
	var board = gnubg.TanBoard{
		layoutToGNU(args.Board.X),
		layoutToGNU(args.Board.O),
	}
	var dice = args.Dice

	var player int = 1

	if args.Player == "o" {
		player = 0
	}

	var maxMoves int = 9999

	if args.MaxMoves > 0 {
		maxMoves = args.MaxMoves
	}

	var pml, err = gnubg.FindMoves(board, dice, player, args.ScoreMoves, args.Cubeful)

	if err != nil {
		return nil, fmt.Errorf("error in gnubg.FindMoves(): %v", err)
	}

	var movesNum int = int(math.Min(float64(pml.GetMovesNum()), float64(maxMoves)))

	var ret = make([]Move, 0, movesNum)

	var topMove gnubg.Move

	for i := 0; i < movesNum; i++ {
		var move = pml.GetMove(i)
		if i == 0 {
			topMove = move
		}

		// add return value
		if args.ScoreMoves {
			evalInfo := move.GetEvalInfo()

			ret = append(ret, Move{
				Play: playFromMove(move),
				Evaluation: &Evaluation{
					Info: EvalInfo{
						Cubeful: evalInfo.Cubeful,
						Plies:   evalInfo.Plies + 1,
					},
					Equity:     outputEquity(move.GetEquity()),
					EquityDiff: outputEquityDiff(move.GetEquity(), topMove.GetEquity()),
					Probablity: Probablity{
						Win:    fformat(move.GetProbWin()),
						WinG:   fformat(move.GetProbWinG()),
						WinBG:  fformat(move.GetProbWinBG()),
						Lose:   fformat(move.GetProbLose()),
						LoseG:  fformat(move.GetProbLoseG()),
						LoseBG: fformat(move.GetProbLoseBG()),
					},
				},
			})
		} else {
			ret = append(ret, Move{
				Play: playFromMove(move),
			})
		}
	}

	return ret, nil
}

func layoutToGNU(layout CheckerLayout) [25]int {
	return [25]int{
		layout.P1,
		layout.P2,
		layout.P3,
		layout.P4,
		layout.P5,
		layout.P6,
		layout.P7,
		layout.P8,
		layout.P9,
		layout.P10,
		layout.P11,
		layout.P12,
		layout.P13,
		layout.P14,
		layout.P15,
		layout.P16,
		layout.P17,
		layout.P18,
		layout.P19,
		layout.P20,
		layout.P21,
		layout.P22,
		layout.P23,
		layout.P24,
		layout.Bar,
	}
}

func playFromMove(move gnubg.Move) []CheckerPlay {
	var play = make([]CheckerPlay, 0, 4)
	for j := 0; j < move.GetPlaysNum(); j++ {
		var ar = move.GetPlay(j)
		var from = strconv.Itoa(ar[0] + 1)
		var to = strconv.Itoa(ar[1] + 1)
		if from == "25" {
			from = "bar"
		}
		if to == "0" {
			to = "off"
		}
		play = append(play, CheckerPlay{
			From: from,
			To:   to,
		})
	}
	return play
}

func outputEquity(score float32) float32 {
	return fformat(score)
}

func outputEquityDiff(score float32, topScore float32) float32 {
	return fformat(score - topScore)
}

func fformat(f float32) float32 {
	return float32(math.Round(float64(f*1000))) / 1000
}
