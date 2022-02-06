package gnubg

import (
	"fmt"
	"io/fs"
)

type TanBoard _TanBoard

type MoveList interface {
	GetMovesNum() int
	GetMove(i int) Move
}

type Move interface {
	GetPlaysNum() int
	GetPlay(i int) [2]int
	GetEvalInfo() EvalInfo
	GetEquity() float32
	GetProbWin() float32
	GetProbWinG() float32
	GetProbWinBG() float32
	GetProbLose() float32
	GetProbLoseG() float32
	GetProbLoseBG() float32
}

type EvalInfo struct {
	Cubeful bool
	Plies   int
}

func Init(dataDir fs.FS) error {
	initMatchEquity(dataDir, "met/Kazaross-XG2.xml")

	if err := evalInitialise(dataDir); err != nil {
		return fmt.Errorf("error in evalInitialise(): %v", err)
	}

	return nil
}

func Destroy() {
	evalShutdown()
}

func FindMoves(board TanBoard, dice [2]int, player int, scoreMoves bool, cubeful bool) (MoveList, error) {

	if scoreMoves {
		var pml = _MoveList{}
		var aamf = &_MOVEFILTER_NORMAL
		var anBoard _TanBoard
		if player == 1 {
			anBoard = _TanBoard{board[1], board[0]}
		} else {
			anBoard = _TanBoard{board[0], board[1]}
		}
		var pci = &_CubeInfo{
			nCube:         1,
			fCubeOwner:    -1,
			fMove:         player,
			nMatchTo:      0,
			anScore:       [2]int{0, 0},
			fCrawford:     false,
			fJacoby:       true,
			fBeavers:      true,
			arGammonPrice: [4]float32{0, 0, 0, 0},
			bgv:           _VARIATION_STANDARD,
		}
		var pec = &_EvalContext{
			fCubeful:       cubeful,
			nPlies:         2,
			fUsePrune:      true,
			fDeterministic: true,
			rNoise:         0,
		}
		if err := findnSaveBestMoves(&pml, dice[0], dice[1], anBoard, nil, 0, pci, pec, aamf); err != nil {
			return nil, err
		}
		return pml, nil
	}

	{
		var tld = _ThreadLocalData{}
		var pml = _MoveList{}
		var anBoard _TanBoard
		if player == 1 {
			anBoard = _TanBoard{board[1], board[0]}
		} else {
			anBoard = _TanBoard{board[0], board[1]}
		}
		generateMoves(&tld, &pml, anBoard, dice[0], dice[1], false)

		return pml, nil
	}
}
