package api

import (
	"bgweb-api/internal/gnubg"
	"bgweb-api/internal/openapi"
	"fmt"
	"math"
	"strconv"
)

func GetMoves(args openapi.MoveArgs) ([]openapi.Move, error) {
	var board = gnubg.TanBoard{
		layoutToGNU(args.Board.X),
		layoutToGNU(args.Board.O),
	}
	var dice = args.Dice

	var player int = 1

	if args.Player == "o" {
		player = 0
	}

	var maxMoves = fromPtr(args.MaxMoves, 9999)
	var scoreMoves = fromPtr(args.ScoreMoves, true)
	var cubeful = fromPtr(args.Cubeful, false)

	var pml, err = gnubg.FindMoves(board, [2]int{dice[0], dice[1]}, player, scoreMoves, cubeful)

	if err != nil {
		return nil, fmt.Errorf("error in gnubg.FindMoves(): %v", err)
	}

	var movesNum int = int(math.Min(float64(pml.GetMovesNum()), float64(maxMoves)))

	var ret = make([]openapi.Move, 0, movesNum)

	var topMove gnubg.Move

	for i := 0; i < movesNum; i++ {
		var move = pml.GetMove(i)
		if i == 0 {
			topMove = move
		}

		// add return value
		if scoreMoves {
			evalInfo := move.GetEvalInfo()

			ret = append(ret, openapi.Move{
				Play: toPtr(playFromMove(move)),
				Evaluation: &openapi.Evaluation{
					Info: &openapi.EvalInfo{
						Cubeful: evalInfo.Cubeful,
						Plies:   evalInfo.Plies + 1,
					},
					Eq:   outputEquity(move.GetEquity()),
					Diff: outputEquityDiff(move.GetEquity(), topMove.GetEquity()),
					Probability: &openapi.Probability{
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
			ret = append(ret, openapi.Move{
				Play: toPtr(playFromMove(move)),
			})
		}
	}

	return ret, nil
}

func layoutToGNU(layout openapi.CheckerLayout) [25]int {
	return [25]int{
		fromPtr(layout.N1, 0),
		fromPtr(layout.N2, 0),
		fromPtr(layout.N3, 0),
		fromPtr(layout.N4, 0),
		fromPtr(layout.N5, 0),
		fromPtr(layout.N6, 0),
		fromPtr(layout.N7, 0),
		fromPtr(layout.N8, 0),
		fromPtr(layout.N9, 0),
		fromPtr(layout.N10, 0),
		fromPtr(layout.N11, 0),
		fromPtr(layout.N12, 0),
		fromPtr(layout.N13, 0),
		fromPtr(layout.N14, 0),
		fromPtr(layout.N15, 0),
		fromPtr(layout.N16, 0),
		fromPtr(layout.N17, 0),
		fromPtr(layout.N18, 0),
		fromPtr(layout.N19, 0),
		fromPtr(layout.N20, 0),
		fromPtr(layout.N21, 0),
		fromPtr(layout.N22, 0),
		fromPtr(layout.N23, 0),
		fromPtr(layout.N24, 0),
		fromPtr(layout.Bar, 0),
	}
}

func playFromMove(move gnubg.Move) []openapi.CheckerPlay {
	var play = make([]openapi.CheckerPlay, 0, 4)
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
		play = append(play, openapi.CheckerPlay{
			From: openapi.CheckerPlayFrom(from),
			To:   openapi.CheckerPlayTo(to),
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

func toPtr[T any](val T) *T {
	return &val
}

func fromPtr[T any](val *T, def T) T {
	if val != nil {
		return *val
	}
	return def
}
