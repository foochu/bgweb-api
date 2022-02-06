package gnubg

import (
	"bgweb-api/gnubg/math32"
	"crypto/md5"
	"fmt"
	"io/fs"
	"math/bits"
	"sort"
)

type _ThreadLocalData struct {
	// id       int
	aMoves   [_MAX_INCOMPLETE_MOVES]_Move
	pnnState [3]_NNState
}

type _BGVariation int

const (
	_VARIATION_STANDARD      _BGVariation = iota /* standard backgammon */
	_VARIATION_NACKGAMMON                        /* standard backgammon with nackgammon starting position */
	_VARIATION_HYPERGAMMON_1                     /* 1-chequer hypergammon */
	_VARIATION_HYPERGAMMON_2                     /* 2-chequer hypergammon */
	_VARIATION_HYPERGAMMON_3                     /* 3-chequer hypergammon */
	_NUM_VARIATIONS
)

const _NUM_OUTPUTS = 5

// const _NUM_CUBEFUL_OUTPUTS = 4
const _NUM_ROLLOUT_OUTPUTS = 7

const _OUTPUT_WIN = 0
const _OUTPUT_WINGAMMON = 1
const _OUTPUT_WINBACKGAMMON = 2
const _OUTPUT_LOSEGAMMON = 3
const _OUTPUT_LOSEBACKGAMMON = 4
const _OUTPUT_EQUITY = 5 /* NB: neural nets do not output equity, only rollouts do. */
const _OUTPUT_CUBEFUL_EQUITY = 6

const _MAX_INCOMPLETE_MOVES = 3875

// const _MAX_MOVES = 3060

/* Evaluation cache size is 2^SIZE entries */
const _CACHE_SIZE_DEFAULT = 19

// const _CACHE_SIZE_GUIMAX = 23

type _CMark int

const (
	_CMARK_NONE _CMark = iota
	_CMARK_ROLLOUT
)

type _TanBoard [2][25]int

type _CubeInfo struct {
	/*
	 * nCube: the current value of the cube,
	 * fCubeOwner: the owner of the cube,
	 * fMove: the player for which we are
	 *        calculating equity for,
	 * fCrawford, fJacoby, fBeavers: optional rules in effect,
	 * arGammonPrice: the gammon prices;
	 *   [ 0 ] = gammon price for player 0,
	 *   [ 1 ] = gammon price for player 1,
	 *   [ 2 ] = backgammon price for player 0,
	 *   [ 3 ] = backgammon price for player 1.
	 *
	 */
	nCube, fCubeOwner, fMove, nMatchTo int
	anScore                            [2]int
	fCrawford, fJacoby, fBeavers       bool
	arGammonPrice                      [4]float32
	bgv                                _BGVariation
}

type _EvalSetup struct {
	et _EvalType
	ec _EvalContext
	// rc _RolloutContext
}

type _EvalType int

const (
	_EVAL_NONE _EvalType = iota
	_EVAL_EVAL
	_EVAL_ROLLOUT
)

type _EvalContext struct {
	/* FIXME expand this... e.g. different settings for different position
	 * classes */
	fCubeful       bool /* cubeful evaluation */
	nPlies         int
	fUsePrune      bool
	fDeterministic bool
	// unsigned int :25;		/* padding */
	rNoise float32 /* standard deviation */
}

type _Move struct {
	anMove        [8]int
	key           _PositionKey
	cMoves, cPips int
	/* scores for this move */
	rScore, rScore2 float32
	/* evaluation for this move */
	arEvalMove [_NUM_ROLLOUT_OUTPUTS]float32
	//arEvalStdDev [NUM_ROLLOUT_OUTPUTS]float32
	esMove _EvalSetup
	cmark  _CMark
}

func (t _Move) GetPlaysNum() int {
	return t.cMoves
}

func (t _Move) GetPlay(i int) [2]int {
	var ret [2]int
	ret[0] = t.anMove[i*2]
	ret[1] = t.anMove[i*2+1]
	return ret
}

func (t _Move) GetEvalInfo() EvalInfo {
	return EvalInfo{
		Cubeful: t.esMove.ec.fCubeful,
		Plies:   t.esMove.ec.nPlies,
	}
}

func (t _Move) GetEquity() float32 {
	return t.rScore
}

func (t _Move) GetProbWin() float32 {
	return t.arEvalMove[_OUTPUT_WIN]
}

func (t _Move) GetProbWinG() float32 {
	return t.arEvalMove[_OUTPUT_WINGAMMON]
}

func (t _Move) GetProbWinBG() float32 {
	return t.arEvalMove[_OUTPUT_WINBACKGAMMON]
}

func (t _Move) GetProbLose() float32 {
	return 1 - t.arEvalMove[_OUTPUT_WIN]
}

func (t _Move) GetProbLoseG() float32 {
	return t.arEvalMove[_OUTPUT_LOSEGAMMON]
}

func (t _Move) GetProbLoseBG() float32 {
	return t.arEvalMove[_OUTPUT_LOSEBACKGAMMON]
}

type _MoveList struct {
	cMoves              int /* and current move when building list */
	cMaxMoves, cMaxPips int
	iMoveBest           int
	rBestScore          float32
	amMoves             []_Move
}

func (t _MoveList) GetMovesNum() int {
	return t.cMoves
}

func (t _MoveList) GetMove(i int) Move {
	return t.amMoves[i]
}

type _PositionClass int

const (
	_CLASS_OVER         _PositionClass = iota /* Game already finished */
	_CLASS_HYPERGAMMON1                       /* hypergammon with 1 chequers */
	_CLASS_HYPERGAMMON2                       /* hypergammon with 2 chequers */
	_CLASS_HYPERGAMMON3                       /* hypergammon with 3 chequers */
	_CLASS_BEAROFF2                           /* Two-sided bearoff database (in memory) */
	_CLASS_BEAROFF_TS                         /* Two-sided bearoff database (on disk) */
	_CLASS_BEAROFF1                           /* One-sided bearoff database (in memory) */
	_CLASS_BEAROFF_OS                         /* One-sided bearoff database /on disk) */
	_CLASS_RACE                               /* Race neural network */
	_CLASS_CRASHED                            /* Contact, one side has less than 7 active checkers */
	_CLASS_CONTACT                            /* Contact neural network */
)

const _N_CLASSES = (_CLASS_CONTACT + 1)

const _CLASS_PERFECT = _CLASS_BEAROFF_TS
const _CLASS_GOOD = _CLASS_BEAROFF_OS /* Good enough to not need SanityCheck */

type classEvalFunc func(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error

/* Race inputs */
const (
	/* In a race position, bar and the 24 points are always empty, so only */
	/* 23*4 (92) are needed */

	/* (0 <= k < 14), RI_OFF + k = */
	/*                       1 if exactly k+1 checkers are off, 0 otherwise */

	_RI_OFF int = 92

	/* Number of cross-overs by outside checkers */

	_RI_NCROSS = 92 + 14

	_HALF_RACE_INPUTS = _RI_NCROSS + 1
)

/* Contact inputs -- see Berliner for most of these */
const (
	/* n - number of checkers off
	 *
	 * off1 -  1         n >= 5
	 * n/5       otherwise
	 *
	 * off2 -  1         n >= 10
	 * (n-5)/5   n < 5 < 10
	 * 0         otherwise
	 *
	 * off3 -  (n-10)/5  n > 10
	 * 0         otherwise
	 */

	_I_OFF1 int = iota
	_I_OFF2
	_I_OFF3

	/* Minimum number of pips required to break contact.
	 *
	 * For each checker x, N(x) is checker location,
	 * C(x) is max({forall o : N(x) - N(o)}, 0)
	 *
	 * Break Contact : (sum over x of C(x)) / 152
	 *
	 * 152 is dgree of contact of start position.
	 */
	_I_BREAK_CONTACT

	/* Location of back checker (Normalized to [01])
	 */
	_I_BACK_CHEQUER

	/* Location of most backward anchor.  (Normalized to [01])
	 */
	_I_BACK_ANCHOR

	/* Forward anchor in opponents home.
	 *
	 * Normalized in the following way:  If there is an anchor in opponents
	 * home at point k (1 <= k <= 6), value is k/6. Otherwise, if there is an
	 * anchor in points (7 <= k <= 12), take k/6 as well. Otherwise set to 2.
	 *
	 * This is an attempt for some continuity, since a 0 would be the "same" as
	 * a forward anchor at the bar.
	 */
	_I_FORWARD_ANCHOR

	/* Average number of pips opponent loses from hits.
	 *
	 * Some heuristics are required to estimate it, since we have no idea what
	 * the best move actually is.
	 *
	 * 1. If board is weak (less than 3 anchors), don't consider hitting on
	 * points 22 and 23.
	 * 2. Don't break anchors inside home to hit.
	 */
	_I_PIPLOSS

	/* Number of rolls that hit at least one checker.
	 */
	_I_P1

	/* Number of rolls that hit at least two checkers.
	 */
	_I_P2

	/* How many rolls permit the back checker to escape (Normalized to [01])
	 */
	_I_BACKESCAPES

	/* Maximum containment of opponent checkers, from our points 9 to op back
	 * checker.
	 *
	 * Value is (1 - n/36), where n is number of rolls to escape.
	 */
	_I_ACONTAIN

	/* Above squared */
	_I_ACONTAIN2

	/* Maximum containment, from our point 9 to home.
	 * Value is (1 - n/36), where n is number of rolls to escape.
	 */
	_I_CONTAIN

	/* Above squared */
	_I_CONTAIN2

	/* For all checkers out of home,
	 * sum (Number of rolls that let x escape * distance from home)
	 *
	 * Normalized by dividing by 3600.
	 */
	_I_MOBILITY

	/* One sided moment.
	 * Let A be the point of weighted average:
	 * A = sum of N(x) for all x) / nCheckers.
	 *
	 * Then for all x : A < N(x), M = (average (N(X) - A)^2)
	 *
	 * Diveded by 400 to normalize.
	 */
	_I_MOMENT2

	/* Average number of pips lost when on the bar.
	 * Normalized to [01]
	 */
	_I_ENTER

	/* Probablity of one checker not entering from bar.
	 * 1 - (1 - n/6)^2, where n is number of closed points in op home.
	 */
	_I_ENTER2

	_I_TIMING

	_I_BACKBONE

	_I_BACKG

	_I_BACKG1

	_I_FREEPIP

	_I_BACKRESCAPES

	_MORE_INPUTS
)

const _MINPPERPOINT = 4
const _NUM_INPUTS = ((25*_MINPPERPOINT + _MORE_INPUTS) * 2)
const _NUM_RACE_INPUTS = (_HALF_RACE_INPUTS * 2)
const _NUM_PRUNING_INPUTS = (25 * _MINPPERPOINT * 2)

var anEscapes [0x1000]int
var anEscapes1 [0x1000]int

var anPoint = [16]int{0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

var nnContact, nnRace, nnCrashed _NeuralNet
var nnpContact, nnpRace, nnpCrashed _NeuralNet

var pbcOS *_BearOffContext
var pbcTS *_BearOffContext
var pbc1 *_BearOffContext
var pbc2 *_BearOffContext
var apbcHyper [3]*_BearOffContext

var cEval _EvalCache
var cpEval _EvalCache
var cCache int
var fInterrupt bool = false

var anChequers = [_NUM_VARIATIONS]int{15, 15, 1, 2, 3}

const (
	/* gammon possible by side on roll */
	_G_POSSIBLE = 0x1
	/* backgammon possible by side on roll */
	_BG_POSSIBLE = 0x2
	/* gammon possible by side not on roll */
	_OG_POSSIBLE = 0x4
	/* backgammon possible by side not on roll */
	_OBG_POSSIBLE = 0x8
)

type _MoveFilter struct {
	accept int /* always allow this many moves. 0 means don't use this */
	/* level, since at least 1 is needed when used. */
	extra     int     /* and add up to this many more... */
	threshold float32 /* ...if they are within this equity difference */
}

const _MAX_FILTER_PLIES = 4

var ecBasic = _EvalContext{false, 0, false, false, 0.0}

var defaultFilters [_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter = _MOVEFILTER_NORMAL

/* parameters for EvalEfficiency */

const rTSCubeX float32 = 0.6 /* for match play only */
const rOSCubeX float32 = 0.6
const rRaceFactorX float32 = 0.00125
const rRaceCoefficientX float32 = 0.55
const rRaceMax float32 = 0.7
const rRaceMin float32 = 0.6
const rCrashedX float32 = 0.68
const rContactX float32 = 0.68

// type _LuckType int

// const (
// 	_LUCK_VERYBAD _LuckType = iota
// 	_LUCK_BAD
// 	_LUCK_NONE
// 	_LUCK_GOOD
// 	_LUCK_VERYGOOD
// )

// type _SkillType int

// const (
// 	_SKILL_VERYBAD _SkillType = iota
// 	_SKILL_BAD
// 	_SKILL_DOUBTFUL
// 	_SKILL_NONE
// )

// var arLuckLevel = []float32{
// 	0.6, /* LUCK_VERYBAD */
// 	0.3, /* LUCK_BAD */
// 	0,   /* LUCK_NONE */
// 	0.3, /* LUCK_GOOD */
// 	0.6, /* LUCK_VERYGOOD */
// }

// var arSkillLevel = []float32{
// 	0.12, /* SKILL_VERYBAD */
// 	0.06, /* SKILL_BAD */
// 	0.03, /* SKILL_DOUBTFUL */
// 	0,    /* SKILL_NONE */
// }

func msb32(n int) int {
	return 31 - bits.LeadingZeros32(uint32(n))
}

func evalInitialise(dataDir fs.FS) error {
	cCache = 0x1 << _CACHE_SIZE_DEFAULT
	if err := cacheCreate(&cEval, cCache); err != nil {
		return fmt.Errorf("error while creating cache: %v", err)
	}

	if err := cacheCreate(&cpEval, 0x1<<16); err != nil {
		return fmt.Errorf("error while creating cache: %v", err)
	}

	computeTable()

	//         rc.randrsl[0] = (ub4) time(NULL);
	//         for (i = 0; i < RANDSIZ; i++)
	//             rc.randrsl[i] = rc.randrsl[0];
	//         irandinit(&rc, TRUE);

	var err error

	if pbc1 == nil {
		pbc1, err = bearoffInit(dataDir, "gnubg_os0.bd", _BO_MUST_BE_ONE_SIDED)
		if err != nil {
			// logWarningf("creating a heuristic bearoff database as a fallback, reason: %v", err)
			// pbc1, err = bearoffInit(dataDir, "", _BO_HEURISTIC)
			// if err != nil {
			// 	return fmt.Errorf("unable to create any type of bearoff database: %v", err)
			// }
			return fmt.Errorf("error while reading bearoff database: %v", err)
		}
	}

	/* read two-sided db from gnubg.bd */
	pbc2, err = bearoffInit(dataDir, "gnubg_ts0.bd", _BO_MUST_BE_TWO_SIDED)
	if err != nil {
		logWarningf("will not use the two-sided bearoff database: %v", err)
	}
	/* init one-sided db */
	pbcOS, _ = bearoffInit(dataDir, "gnubg_os.bd", _BO_MUST_BE_ONE_SIDED)

	/* init two-sided db */
	pbcTS, _ = bearoffInit(dataDir, "gnubg_ts.bd", _BO_MUST_BE_TWO_SIDED)

	/* hyper-gammon databases */

	for i := 0; i < 3; i++ {
		fn := fmt.Sprintf("hyper%1d.bd", i+1)
		apbcHyper[i], _ = bearoffInit(dataDir, fn, _BO_NONE)
	}

	weightsFile := "gnubg.weights"
	pfWeights, err := dataDir.Open(weightsFile)
	if err != nil {
		return fmt.Errorf("error while opening weights file %v: %v", weightsFile, err)
	}
	defer pfWeights.Close()

	if err = verifyWeights(pfWeights, weightsFile); err != nil {
		return fmt.Errorf("error while verifying weights file: %v", err)
	}

	if err := neuralNetLoad(&nnContact, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnContact: %v", err)
	}
	if err := neuralNetLoad(&nnRace, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnRace: %v", err)
	}
	if err := neuralNetLoad(&nnCrashed, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnCrashed: %v", err)
	}
	if err := neuralNetLoad(&nnpContact, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnpContact: %v", err)
	}
	if err := neuralNetLoad(&nnpCrashed, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnpCrashed: %v", err)
	}
	if err := neuralNetLoad(&nnpRace, pfWeights); err != nil {
		return fmt.Errorf("error while loading nnpRace: %v", err)
	}

	if nnContact.cInput != _NUM_INPUTS || nnContact.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnContact")
	}
	if nnCrashed.cInput != _NUM_INPUTS || nnCrashed.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnCrashed")
	}
	if nnRace.cInput != _NUM_RACE_INPUTS || nnRace.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnRace")
	}
	if nnpContact.cInput != _NUM_PRUNING_INPUTS || nnpContact.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnpContact")
	}
	if nnpCrashed.cInput != _NUM_PRUNING_INPUTS || nnpCrashed.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnpCrashed")
	}
	if nnpRace.cInput != _NUM_PRUNING_INPUTS || nnpRace.cOutput != _NUM_OUTPUTS {
		return fmt.Errorf("invalid nnpRace")
	}

	return nil
}

func evalShutdown() {
	/* close bearoff databases */
	bearoffClose(pbc1)
	bearoffClose(pbc2)
	bearoffClose(pbcOS)
	bearoffClose(pbcTS)
	for i := 0; i < 3; i++ {
		bearoffClose(apbcHyper[i])
	}

	/* destroy neural nets */
	destroyWeights()

	/* destroy cache */
	cacheDestroy(&cEval)
	cacheDestroy(&cpEval)
}

func destroyWeights() {
	neuralNetDestroy(&nnContact)
	neuralNetDestroy(&nnCrashed)
	neuralNetDestroy(&nnRace)

	neuralNetDestroy(&nnpContact)
	neuralNetDestroy(&nnpCrashed)
	neuralNetDestroy(&nnpRace)
}

func generateMoves(tld *_ThreadLocalData, pml *_MoveList, anBoard _TanBoard, n0 int, n1 int, fPartial bool) int {
	var anRoll [4]int
	var anMoves [8]int
	anRoll[0] = n0
	anRoll[1] = n1

	if n0 == n1 {
		anRoll[2] = n0
		anRoll[3] = n0
	}

	pml.cMoves = 0
	pml.cMaxMoves = 0
	pml.cMaxPips = 0
	pml.iMoveBest = 0
	pml.amMoves = tld.aMoves[:]
	generateMovesSub(pml, anRoll, 0, 23, 0, anBoard, &anMoves, fPartial)

	if anRoll[0] != anRoll[1] {
		swap(&anRoll[0], &anRoll[1])

		generateMovesSub(pml, anRoll, 0, 23, 0, anBoard, &anMoves, fPartial)
	}

	return pml.cMoves
}

func swap(p0 *int, p1 *int) {
	n := *p0
	*p0 = *p1
	*p1 = n
}

func generateMovesSub(pml *_MoveList, anRoll [4]int, nMoveDepth int, iPip int, cPip int, anBoard _TanBoard, anMoves *[8]int, fPartial bool) bool {
	var fUsed int
	var anBoardNew _TanBoard

	if nMoveDepth > 3 || anRoll[nMoveDepth] == 0 {
		return true
	}

	if anBoard[1][24] > 0 { /* on bar */
		if anBoard[0][anRoll[nMoveDepth]-1] >= 2 {
			return true
		}

		anMoves[nMoveDepth*2] = 24
		anMoves[nMoveDepth*2+1] = 24 - anRoll[nMoveDepth]

		for i := 0; i < 25; i++ {
			anBoardNew[0][i] = anBoard[0][i]
			anBoardNew[1][i] = anBoard[1][i]
		}

		applySubMove(&anBoardNew, 24, anRoll[nMoveDepth], true)

		if generateMovesSub(pml, anRoll, nMoveDepth+1, 23, cPip+anRoll[nMoveDepth], anBoardNew, anMoves, fPartial) {
			saveMoves(pml, nMoveDepth+1, cPip+anRoll[nMoveDepth], *anMoves, anBoardNew, fPartial)
		}

		return fPartial
	} else {
		for i := iPip; i >= 0; i-- {
			if anBoard[1][i] > 0 && legalMove(anBoard, i, anRoll[nMoveDepth]) {
				anMoves[nMoveDepth*2] = i
				anMoves[nMoveDepth*2+1] = i - anRoll[nMoveDepth]

				copy(anBoardNew[:][:], anBoard[:][:])

				applySubMove(&anBoardNew, i, anRoll[nMoveDepth], true)

				var iPipNew int
				if anRoll[0] == anRoll[1] {
					iPipNew = i
				} else {
					iPipNew = 23
				}
				if generateMovesSub(pml, anRoll, nMoveDepth+1, iPipNew, cPip+anRoll[nMoveDepth], anBoardNew, anMoves, fPartial) {
					saveMoves(pml, nMoveDepth+1, cPip+anRoll[nMoveDepth], *anMoves, anBoardNew, fPartial)
				}

				fUsed = 1
			}
		}
	}

	return fUsed == 0 || fPartial
}

func legalMove(anBoard _TanBoard, iSrc int, nPips int) bool {
	var nBack int
	iDest := iSrc - nPips

	if iDest >= 0 { /* Here we can do the Chris rule check */
		return anBoard[0][23-iDest] < 2
	}
	/* otherwise, attempting to bear off */

	for nBack = 24; nBack > 0; nBack-- {
		if anBoard[1][nBack] > 0 {
			break
		}
	}

	return (nBack <= 5 && (iSrc == nBack || iDest == -1))
}

func applySubMove(anBoard *_TanBoard, iSrc int, nRoll int, fCheckLegal bool) error {
	iDest := iSrc - nRoll

	if fCheckLegal && (nRoll < 1 || nRoll > 6) {
		return fmt.Errorf("invalid dice roll")
	}

	if iSrc < 0 || iSrc > 24 || iDest >= iSrc || anBoard[1][iSrc] < 1 {
		return fmt.Errorf("invalid point number, or source is empty")
	}

	anBoard[1][iSrc]--

	if iDest < 0 {
		return nil
	}

	if anBoard[0][23-iDest] > 0 {
		if anBoard[0][23-iDest] > 1 {
			return fmt.Errorf("trying to move to a point already made by the opponent")
		}
		anBoard[1][iDest] = 1
		anBoard[0][23-iDest] = 0
		anBoard[0][24]++
	} else {
		anBoard[1][iDest]++
	}

	return nil
}

func saveMoves(pml *_MoveList, cMoves int, cPip int, anMoves [8]int, anBoard _TanBoard, fPartial bool) {
	var pm *_Move
	var key _PositionKey

	if fPartial {
		/* Save all moves, even incomplete ones */
		if cMoves > pml.cMaxMoves {
			pml.cMaxMoves = cMoves
		}
		if cPip > pml.cMaxPips {
			pml.cMaxPips = cPip
		}
	} else {
		/* Save only legal moves: if the current move moves plays less
		 * chequers or pips than those already found, it is illegal; if
		 * it plays more, the old moves are illegal. */
		if cMoves < pml.cMaxMoves || cPip < pml.cMaxPips {
			return
		}

		if cMoves > pml.cMaxMoves || cPip > pml.cMaxPips {
			pml.cMoves = 0
		}

		pml.cMaxMoves = cMoves
		pml.cMaxPips = cPip
	}
	key.fromBoard(anBoard)

	for i := 0; i < pml.cMoves; i++ {

		pm = &pml.amMoves[i]

		if key.equals(pm.key) {
			if cMoves > pm.cMoves || cPip > pm.cPips {
				for j := 0; j < cMoves*2; j++ {
					if anMoves[j] > -1 {
						pm.anMove[j] = anMoves[j]
					} else {
						pm.anMove[j] = -1
					}
				}

				if cMoves < 4 {
					pm.anMove[cMoves*2] = -1
				}

				pm.cMoves = cMoves
				pm.cPips = cPip
			}

			return
		}
	}

	pm = &pml.amMoves[pml.cMoves]

	for i := 0; i < cMoves*2; i++ {
		if anMoves[i] > -1 {
			pm.anMove[i] = anMoves[i]
		} else {
			pm.anMove[i] = -1
		}
	}

	if cMoves < 4 {
		pm.anMove[cMoves*2] = -1
	}

	pm.key.copyFrom(key)

	pm.cMoves = cMoves
	pm.cPips = cPip
	pm.cmark = _CMARK_NONE

	for i := 0; i < _NUM_OUTPUTS; i++ {
		pm.arEvalMove[i] = 0.0
	}

	pml.cMoves++

	if pml.cMoves >= _MAX_INCOMPLETE_MOVES {
		panic("pml.cMoves >= MAX_INCOMPLETE_MOVES")
	}
}

func scoreMoves(tld *_ThreadLocalData, pml *_MoveList, pci *_CubeInfo, pec *_EvalContext, nPlies int) error {
	nnStates := &tld.pnnState

	pml.rBestScore = -99999.9

	if nPlies == 0 {
		/* start incremental evaluations */
		nnStates[0].state = _NNSTATE_INCREMENTAL
		nnStates[1].state = _NNSTATE_INCREMENTAL
		nnStates[2].state = _NNSTATE_INCREMENTAL

		defer func() {
			/* reset to none */
			nnStates[0].state = _NNSTATE_NONE
			nnStates[1].state = _NNSTATE_NONE
			nnStates[2].state = _NNSTATE_NONE
		}()
	}

	for i := 0; i < pml.cMoves; i++ {
		if err := scoreMove(tld, nnStates, &pml.amMoves[i], pci, pec, nPlies); err != nil {
			return fmt.Errorf("error in scoreMove: %v", err)
		}

		if (pml.amMoves[i].rScore > pml.rBestScore) || ((pml.amMoves[i].rScore == pml.rBestScore) && (pml.amMoves[i].rScore2 > pml.amMoves[pml.iMoveBest].rScore2)) {
			pml.iMoveBest = i
			pml.rBestScore = pml.amMoves[i].rScore
		}
	}

	return nil
}

func scoreMovesPruned(tld *_ThreadLocalData, pml *_MoveList, pci *_CubeInfo, pec *_EvalContext, bmovesi *[_MAX_PRUNE_MOVES]int, prune_moves int) error {
	nnStates := &tld.pnnState

	pml.rBestScore = -99999.9

	/* start incremental evaluations */
	nnStates[0].state = _NNSTATE_INCREMENTAL
	nnStates[1].state = _NNSTATE_INCREMENTAL
	nnStates[2].state = _NNSTATE_INCREMENTAL

	defer func() {
		nnStates[0].state = _NNSTATE_NONE
		nnStates[1].state = _NNSTATE_NONE
		nnStates[2].state = _NNSTATE_NONE
	}()

	for j := 0; j < prune_moves; j++ {
		i := bmovesi[j]

		if err := scoreMove(tld, nnStates, &pml.amMoves[i], pci, pec, 0); err != nil {
			return fmt.Errorf("error in scoreMove: %v", err)
		}

		if (pml.amMoves[i].rScore > pml.rBestScore) || ((pml.amMoves[i].rScore == pml.rBestScore) && (pml.amMoves[i].rScore2 > pml.amMoves[pml.iMoveBest].rScore2)) {
			pml.iMoveBest = i
			pml.rBestScore = pml.amMoves[i].rScore
		}
	}

	return nil
}

func scoreMove(tld *_ThreadLocalData, nnStates *[3]_NNState, pm *_Move, pci *_CubeInfo, pec *_EvalContext, nPlies int) error {
	var anBoardTemp _TanBoard
	var arEval [_NUM_ROLLOUT_OUTPUTS]float32
	var ci _CubeInfo

	pm.key.toBoard(&anBoardTemp)
	swapSides(&anBoardTemp)

	ci = *pci
	ci.fMove ^= 1

	if err := generalEvaluationEPlied(tld, nnStates, &arEval, anBoardTemp, &ci, pec, nPlies); err != nil {
		return err
	}

	invertEvaluationR(&arEval, &ci)

	if ci.nMatchTo > 0 {
		arEval[_OUTPUT_CUBEFUL_EQUITY] = mwc2eq(arEval[_OUTPUT_CUBEFUL_EQUITY], pci)
	}

	/* Save evaluations */
	pm.arEvalMove = arEval

	/* Save evaluation setup */
	pm.esMove.et = _EVAL_EVAL
	pm.esMove.ec = *pec
	pm.esMove.ec.nPlies = nPlies

	/* Score for move:
	 * rScore is the primary score (cubeful/cubeless)
	 * rScore2 is the secondary score (cubeless) */
	if pec.fCubeful {
		pm.rScore = arEval[_OUTPUT_CUBEFUL_EQUITY]
	} else {
		pm.rScore = arEval[_OUTPUT_EQUITY]
	}
	pm.rScore2 = arEval[_OUTPUT_EQUITY]

	// fmt.Printf("=== ScoreMove(), pm->anMove: %d, %d, %d, %d, %d, %d, %d, %d, pm->rScore: %f, pm->rScore2 %f\n", pm.anMove[0], pm.anMove[1], pm.anMove[2], pm.anMove[3], pm.anMove[4], pm.anMove[5], pm.anMove[6], pm.anMove[7], pm.rScore, pm.rScore2)

	return nil
}

func invertEvaluationR(ar *[_NUM_ROLLOUT_OUTPUTS]float32, pci *_CubeInfo) {
	/* invert win, gammon etc. */

	var arOutputTmp [_NUM_OUTPUTS]float32

	copy(arOutputTmp[:], ar[:])

	invertEvaluation(&arOutputTmp)

	copy(ar[:], arOutputTmp[:])

	/* invert equities */

	ar[_OUTPUT_EQUITY] = -ar[_OUTPUT_EQUITY]

	if pci.nMatchTo > 0 {
		ar[_OUTPUT_CUBEFUL_EQUITY] = 1.0 - ar[_OUTPUT_CUBEFUL_EQUITY]
	} else {
		ar[_OUTPUT_CUBEFUL_EQUITY] = -ar[_OUTPUT_CUBEFUL_EQUITY]
	}

}

func invertEvaluation(ar *[_NUM_OUTPUTS]float32) {
	var r float32

	ar[_OUTPUT_WIN] = 1.0 - ar[_OUTPUT_WIN]

	r = ar[_OUTPUT_WINGAMMON]
	ar[_OUTPUT_WINGAMMON] = ar[_OUTPUT_LOSEGAMMON]
	ar[_OUTPUT_LOSEGAMMON] = r

	r = ar[_OUTPUT_WINBACKGAMMON]
	ar[_OUTPUT_WINBACKGAMMON] = ar[_OUTPUT_LOSEBACKGAMMON]
	ar[_OUTPUT_LOSEBACKGAMMON] = r
}

func generalEvaluationEPlied(tld *_ThreadLocalData, nnStates *[3]_NNState, arOutput *[_NUM_ROLLOUT_OUTPUTS]float32, anBoard _TanBoard, pci *_CubeInfo, pec *_EvalContext, nPlies int) error {

	// fmt.Printf("=== GeneralEvaluationEPlied()\n")
	// fmt.Printf(" anBoard[0]: %v\n", anBoard[0])
	// fmt.Printf(" anBoard[1]: %v\n", anBoard[1])

	if pec.fCubeful {
		if err := generalEvaluationEPliedCubeful(tld, nnStates, arOutput, anBoard, pci, pec, nPlies); err != nil {
			return err
		}
	} else {
		var arOutputTmp [_NUM_OUTPUTS]float32

		copy(arOutputTmp[:], (*arOutput)[:])

		if err := evaluatePositionCache(tld, nnStates, anBoard, &arOutputTmp, pci, pec, nPlies, classifyPosition(anBoard, pci.bgv)); err != nil {
			return fmt.Errorf("error in evaluatePositionCache: %v", err)
		}

		copy((*arOutput)[:], arOutputTmp[:])

		arOutput[_OUTPUT_EQUITY] = utilityME(&arOutputTmp, pci)
		arOutput[_OUTPUT_CUBEFUL_EQUITY] = 0.0
	}

	// fmt.Printf(" arOutput: %v\n", arOutput)

	// if 1 == 1 {
	// 	os.Exit(0)
	// }

	return nil
}

func generalEvaluationEPliedCubeful(tld *_ThreadLocalData, nnStates *[3]_NNState, arOutput *[_NUM_ROLLOUT_OUTPUTS]float32, anBoard _TanBoard, pci *_CubeInfo, pec *_EvalContext, nPlies int) error {
	rCubeful := make([]float32, 1)

	aciCubePos := []_CubeInfo{*pci}

	var arOutputTmp [_NUM_OUTPUTS]float32

	copy(arOutputTmp[:], (*arOutput)[:])

	if err := evaluatePositionCubeful3(tld, nnStates, anBoard, &arOutputTmp, rCubeful, aciCubePos, 1, pci, pec, nPlies, false); err != nil {
		return err
	}

	copy((*arOutput)[:], arOutputTmp[:])

	arOutput[_OUTPUT_EQUITY] = utilityME(&arOutputTmp, pci)
	arOutput[_OUTPUT_CUBEFUL_EQUITY] = rCubeful[0]

	return nil
}

func evaluatePositionCubeful3(tld *_ThreadLocalData, nnStates *[3]_NNState, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, arCubeful []float32, aciCubePos []_CubeInfo, cci int, pciMove *_CubeInfo, pec *_EvalContext, nPlies int, fTop bool) error {

	// var ici int
	// var fAll bool = TRUE
	// var ec EvalCache

	// if !cCache || pec.rNoise != 0.0 {
	/* non-deterministic evaluation; never cache */
	return evaluatePositionCubeful4(tld, nnStates, anBoard, arOutput, arCubeful, aciCubePos, cci, pciMove, pec, nPlies, fTop)
	// }

	// PositionKey(anBoard, &ec.key);

	// /* check cache for existence for earlier calculation */

	// fAll = !fTop;               /* FIXME: fTop should be a part of EvalKey */

	// for (ici = 0; ici < cci && fAll; ++ici) {

	// if (aciCubePos[ici].nCube < 0) {
	// continue;
	// }

	// ec.nEvalContext = EvalKey(pec, nPlies, &aciCubePos[ici], TRUE);

	// if (CacheLookup(&cEval, &ec, arOutput, arCubeful + ici) != CACHEHIT) {
	// fAll = FALSE;
	// }
	// }

	// /* get equities */

	// if (!fAll) {

	// /* cache miss */
	// if (EvaluatePositionCubeful4(nnStates, anBoard, arOutput, arCubeful,
	// 				aciCubePos, cci, pciMove, pec, nPlies, fTop))
	// return -1;

	// /* add to cache */

	// if (!fTop) {

	// for (ici = 0; ici < cci; ++ici) {
	// if (aciCubePos[ici].nCube < 0)
	// continue;

	// memcpy(ec.ar, arOutput, sizeof(float) * NUM_OUTPUTS);
	// ec.ar[5] = arCubeful[ici];      /* Cubeful equity stored in slot 5 */
	// ec.nEvalContext = EvalKey(pec, nPlies, &aciCubePos[ici], TRUE);

	// CacheAdd(&cEval, &ec, GetHashKey(cEval.hashMask, &ec));

	// }
	// }
	// }

	// return nil

}

func evaluatePositionCubeful4(tld *_ThreadLocalData, nnStates *[3]_NNState, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, arCubeful []float32, aciCubePos []_CubeInfo, cci int, pciMove *_CubeInfo, pec *_EvalContext, nPlies int, fTop bool) error {
	/* calculate cubeful equity */

	// int i;
	var pc _PositionClass
	var ar [_NUM_OUTPUTS]float32
	var arEquity [4]float32

	var ciMoveOpp _CubeInfo

	arCf := make([]float32, 2*cci)
	arCfTemp := make([]float32, 2*cci)
	aci := make([]_CubeInfo, 2*cci)

	// var s string
	// s += fmt.Sprintf("=== evaluatePositionCubeful4()\n")
	// s += fmt.Sprintf(" anBoard[0]: %v\n", anBoard[0])
	// s += fmt.Sprintf(" anBoard[1]: %v\n", anBoard[1])

	pc = classifyPosition(anBoard, pciMove.bgv)

	if pc > _CLASS_OVER && nPlies > 0 && !(pc <= _CLASS_PERFECT && pciMove.nMatchTo == 0) {
		/* internal node; recurse */

		var anBoardNew _TanBoard
		var n0, n1 int
		var r float32

		usePrune := pec.fUsePrune && pec.rNoise == 0.0 && pciMove.bgv == _VARIATION_STANDARD

		for i := 0; i < _NUM_OUTPUTS; i++ {
			arOutput[i] = 0.0
		}

		for i := 0; i < 2*cci; i++ {
			arCf[i] = 0.0
		}

		/* construct next level cube positions */

		makeCubePos(aciCubePos, cci, fTop, aci, true)

		/* loop over rolls */

		for n0 = 1; n0 <= 6; n0++ {
			for n1 = 1; n1 <= n0; n1++ {
				var w float32
				if n0 == n1 {
					w = 1.0
				} else {
					w = 2.0
				}

				for i := 0; i < 25; i++ {
					anBoardNew[0][i] = anBoard[0][i]
					anBoardNew[1][i] = anBoard[1][i]
				}

				if fInterrupt {
					return fmt.Errorf("evaluation interrupted")
				}

				if usePrune {
					findBestMoveInEval(tld, nnStates, n0, n1, anBoard, &anBoardNew, pciMove, pec)
				} else {
					if _, err := findBestMovePlied(nil, n0, n1, &anBoardNew, pciMove, pec, 0, &defaultFilters); err != nil {
						logWarningf("error in findBestMovePlied: %v", err)
					}
				}

				swapSides(&anBoardNew)

				setCubeInfo(&ciMoveOpp, pciMove.nCube, pciMove.fCubeOwner, 1-pciMove.fMove, pciMove.nMatchTo, pciMove.anScore, pciMove.fCrawford, pciMove.fJacoby, pciMove.fBeavers, pciMove.bgv)

				/* Evaluate at 0-ply */
				if err := evaluatePositionCubeful3(tld, nnStates, anBoardNew, &ar, arCfTemp, aci, 2*cci, &ciMoveOpp, pec, nPlies-1, false); err != nil {
					return fmt.Errorf("erron in evaluatePositionCubeful3: %v", err)
				}
				/* Sum up cubeless winning chances and cubeful equities */
				for i := 0; i < _NUM_OUTPUTS; i++ {
					arOutput[i] += w * ar[i]
				}
				for i := 0; i < 2*cci; i++ {
					arCf[i] += w * arCfTemp[i]
				}
			}
		}

		/* Flip evals */
		const sumW = 36

		arOutput[_OUTPUT_WIN] = 1.0 - arOutput[_OUTPUT_WIN]/sumW

		r = arOutput[_OUTPUT_WINGAMMON] / sumW
		arOutput[_OUTPUT_WINGAMMON] = arOutput[_OUTPUT_LOSEGAMMON] / sumW
		arOutput[_OUTPUT_LOSEGAMMON] = r

		r = arOutput[_OUTPUT_WINBACKGAMMON] / sumW
		arOutput[_OUTPUT_WINBACKGAMMON] = arOutput[_OUTPUT_LOSEBACKGAMMON] / sumW
		arOutput[_OUTPUT_LOSEBACKGAMMON] = r

		for i := 0; i < 2*cci; i++ {
			if pciMove.nMatchTo > 0 {
				arCf[i] = 1.0 - arCf[i]/sumW
			} else {
				arCf[i] = -arCf[i] / sumW
			}
		}

		/* invert fMove */
		/* Remember than fMove was inverted in the call to MakeCubePos */

		for i := 0; i < 2*cci; i++ {
			aci[i].fMove = 1 - aci[i].fMove
		}

		/* get cubeful equities */
		getECF3(arCubeful, cci, arCf, aci)

		// s += fmt.Sprintf(" arOutput: %v\n", arOutput)

		// if nPlies == 2 {
		// 	fmt.Printf("%v", s)
		// 	os.Exit(0)
		// }

	} else {
		/* at leaf node; use static evaluation */

		if pc == _CLASS_HYPERGAMMON1 || pc == _CLASS_HYPERGAMMON2 || pc == _CLASS_HYPERGAMMON3 {
			pbc := apbcHyper[pc-_CLASS_HYPERGAMMON1]
			var nUs, nThem, iPos int
			var n int

			if pbc != nil {
				return fmt.Errorf("pbc not supplied")
			}

			nUs = positionBearoff(anBoard.getHomeBoard(1), pbc.nPoints, pbc.nChequers)
			nThem = positionBearoff(anBoard.getHomeBoard(0), pbc.nPoints, pbc.nChequers)
			n = combination(pbc.nPoints+pbc.nChequers, pbc.nPoints)
			iPos = nUs*n + nThem

			if err := bearoffHyper(apbcHyper[pc-_CLASS_HYPERGAMMON1], iPos, arOutput, &arEquity); err != nil {
				return fmt.Errorf("error in bearoffHyper: %v", err)
			}
		} else if pc > _CLASS_OVER && pc <= _CLASS_PERFECT /* && ! pciMove->nMatchTo */ {
			if err := evaluatePerfectCubeful(anBoard, &arEquity, pciMove.bgv); err != nil {
				return fmt.Errorf("error in evaluatePerfectCubeful: %v", err)
			}

			arOutput[_OUTPUT_WIN] = (arEquity[0] + 1.0) / 2.0
			arOutput[_OUTPUT_WINGAMMON] = 0.0
			arOutput[_OUTPUT_WINBACKGAMMON] = 0.0
			arOutput[_OUTPUT_LOSEGAMMON] = 0.0
			arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0
		} else {
			/* evaluate with neural net */
			if err := evaluatePosition(tld, nnStates, anBoard, arOutput, pciMove, nil); err != nil {
				return fmt.Errorf("error in evaluatePosition: %v", err)
			}

			if pec.rNoise > 0.0 && pc != _CLASS_OVER {
				for i := 0; i < _NUM_OUTPUTS; i++ {
					arOutput[i] += noise(pec, anBoard, i)
					arOutput[i] = math32.Max(arOutput[i], 0.0)
					arOutput[i] = math32.Min(arOutput[i], 1.0)
				}
			}

			if pc > _CLASS_GOOD || pec.rNoise > 0.0 {
				/* no sanity check needed for accurate evaluations */
				sanityCheck(anBoard, arOutput)
			}
		}

		/* Calculate cube efficiency */

		var rCubeX float32 = evalEfficiency(anBoard, pc)

		/* Build all possible cube positions */

		makeCubePos(aciCubePos, cci, fTop, aci, false)

		/* Calculate cubeful equity for each possible cube position */

		for ici := 0; ici < 2*cci; ici++ {
			if aci[ici].nCube > 0 {
				// /* cube available */
				if aci[ici].nMatchTo == 0 {
					/* money play */

					switch pc {
					case _CLASS_HYPERGAMMON1, _CLASS_HYPERGAMMON2, _CLASS_HYPERGAMMON3:
						/* exact bearoff equities & contact */
						arCf[ici] = _CFHYPER(arEquity, &aci[ici])

					case _CLASS_BEAROFF2, _CLASS_BEAROFF_TS:
						/* exact bearoff equities */
						arCf[ici] = _CFMONEY(arEquity, &aci[ici])

					case _CLASS_OVER, _CLASS_RACE, _CLASS_CRASHED, _CLASS_CONTACT, _CLASS_BEAROFF1, _CLASS_BEAROFF_OS:
						/* approximate using Janowski's formulae */
						arCf[ici] = _Cl2CfMoney(arOutput, &aci[ici], rCubeX)

					}

				} else {

					var rCl, rCf, rCfMoney float32
					var X float32 = rCubeX
					var ciMoney _CubeInfo

					/* match play */

					switch pc {
					case _CLASS_HYPERGAMMON1, _CLASS_HYPERGAMMON2, _CLASS_HYPERGAMMON3:
						/* use exact money equities to guess cube efficiency */

						setCubeInfoMoney(&ciMoney, 1, aci[ici].fCubeOwner, aci[ici].fMove, false, false, aci[ici].bgv)

						rCl = utility(arOutput, &ciMoney)
						rCubeX = 1.0
						rCf = _Cl2CfMoney(arOutput, &ciMoney, rCubeX)
						rCfMoney = _CFHYPER(arEquity, &ciMoney)

						if math32.Fabsf(rCl-rCf) > 0.0001 {
							rCubeX = (rCfMoney - rCl) / (rCf - rCl)
						}

						arCf[ici] = _Cl2CfMatch(arOutput, &aci[ici], rCubeX)

						rCubeX = X

					case _CLASS_BEAROFF2, _CLASS_BEAROFF_TS:
						/* use exact money equities to guess cube efficiency */

						setCubeInfoMoney(&ciMoney, 1, aci[ici].fCubeOwner, aci[ici].fMove, false, false, aci[ici].bgv)

						rCl = arEquity[0]
						rCubeX = 1.0
						rCf = _Cl2CfMoney(arOutput, &ciMoney, rCubeX)
						rCfMoney = _CFMONEY(arEquity, &ciMoney)

						if math32.Fabsf(rCl-rCf) > 0.0001 {
							rCubeX = (rCfMoney - rCl) / (rCf - rCl)
						} else {
							rCubeX = X
						}

						/* fabsf(...) > 0.0001 above is not enough. We still get some
						* nutty values for rCubeX and need more sanity checking */

						if rCubeX < 0.0 {
							rCubeX = 0.0
						}
						if rCubeX > X {
							rCubeX = X
						}

						arCf[ici] = _Cl2CfMatch(arOutput, &aci[ici], rCubeX)

						rCubeX = X

					case _CLASS_OVER, _CLASS_RACE, _CLASS_CRASHED, _CLASS_CONTACT, _CLASS_BEAROFF1, _CLASS_BEAROFF_OS:
						/* approximate using Joern's generalisation of
						* Janowski's formulae */

						arCf[ici] = _Cl2CfMatch(arOutput, &aci[ici], rCubeX)

					}

				}

			}
		}
		/* find optimal of "no double" and "double" */

		getECF3(arCubeful, cci, arCf, aci)

		// s += fmt.Sprintf(" arOutput: %v\n", arOutput)

		// if nPlies == 2 {
		// 	fmt.Printf("%v", s)
		// 	os.Exit(0)
		// }

	}

	return nil

}

func getECF3(arCubeful []float32, cci int, arCf []float32, aci []_CubeInfo) {
	var rND, rDT, rDP float32

	for ici, i := 0, 0; ici < cci; ici, i = ici+1, i+2 {
		if aci[i+1].nCube > 0 {
			/* cube available */
			rND = arCf[i]

			if aci[0].nMatchTo > 0 {
				rDT = arCf[i+1]
			} else {
				rDT = 2.0 * arCf[i+1]
			}

			getDPEq(nil, &rDP, aci[i])

			if rDT >= rND && rDP >= rND {
				/* double */
				if rDT >= rDP {
					/* pass */
					arCubeful[ici] = rDP
				} else {
					/* take */
					arCubeful[ici] = rDT
				}
			} else {
				/* no double */
				arCubeful[ici] = rND
			}
		} else {
			/* no cube available: always no double */
			arCubeful[ici] = arCf[i]
		}
	}
}

/*
 * The pruning nets select the best MIN_PRUNE_MOVES +
 * floor(log2(number of legal moves)) moves instead of 10 as they used
 * to do.  A value of 5 for MIN_PRUNE_MOVES brings a small speed-up
 * and, according to the Depreli benchmark, an insignificant strength
 * improvement.  Using a lower value causes a measurable degradation
 * of play. Using a higher one doesn't significantly improve it.
 */
const _MIN_PRUNE_MOVES = 5
const _MAX_PRUNE_MOVES = _MIN_PRUNE_MOVES + 11

func findBestMoveInEval(tld *_ThreadLocalData, nnStates *[3]_NNState, nDice0 int, nDice1 int, anBoardIn _TanBoard, anBoardOut *_TanBoard, pci *_CubeInfo, pec *_EvalContext) {
	// 	 unsigned int i;
	var ml _MoveList
	var evalClass _PositionClass = _CLASS_OVER
	var bmovesi [_MAX_PRUNE_MOVES]int
	var prune_moves int
	// var s string

	// s += "=== findBestMoveInEval()\n"
	// s += fmt.Sprintf(" anBoardIn[0]: %v\n", anBoardIn[0])
	// s += fmt.Sprintf(" anBoardIn[1]: %v\n", anBoardIn[1])

	generateMoves(tld, &ml, anBoardIn, nDice0, nDice1, false)

	if ml.cMoves == 0 {
		/* no legal moves */
		return
	}

	if ml.cMoves == 1 {
		/* forced move */
		ml.iMoveBest = 0
		move := &ml.amMoves[ml.iMoveBest]
		move.key.toBoard(anBoardOut)
		return
	}

	/* LogCube() is floor(log2()) */
	prune_moves = _MIN_PRUNE_MOVES + logCube(ml.cMoves)

	if ml.cMoves <= prune_moves {
		scoreMoves(tld, &ml, pci, pec, 0)
		move := &ml.amMoves[ml.iMoveBest]
		move.key.toBoard(anBoardOut)

		// s += fmt.Sprintf(" anBoardOut[0]: %v\n", anBoardOut[0])
		// s += fmt.Sprintf(" anBoardOut[1]: %v\n", anBoardOut[1])
		// s += fmt.Sprintf(" ml.cMoves(%v) <= prune_moves(%v)\n", ml.cMoves, prune_moves)

		// fmt.Printf("%v", s)

		// if 1 == 1 {
		// 	os.Exit(0)
		// }

		return
	}

	pci.fMove = 1 - pci.fMove

	var i int
	for i = 0; i < ml.cMoves; i++ {
		var pc _PositionClass
		var arOutput [_NUM_OUTPUTS]float32
		var ec _CacheNodeDetail

		pm := &ml.amMoves[i]

		pm.key.toBoard(anBoardOut)
		swapSides(anBoardOut)

		// s += " --- after swapSides()\n"
		// s += fmt.Sprintf(" anBoardOut[0]: %v\n", anBoardOut[0])
		// s += fmt.Sprintf(" anBoardOut[1]: %v\n", anBoardOut[1])

		// fmt.Printf("%v", s)

		// if 1 == 1 {
		// 	os.Exit(0)
		// }

		pc = classifyPosition(*anBoardOut, _VARIATION_STANDARD)
		if i == 0 {
			if pc < _CLASS_RACE {
				break
			}
			evalClass = pc
		} else if pc != evalClass {
			break
		}

		ec.key.copyFrom(pm.key)
		ec.nEvalContext = 0
		if hit, l := cacheLookup(&cpEval, &ec, &arOutput, nil); !hit {
			var arInput []float32 = make([]float32, _NUM_PRUNING_INPUTS)

			baseInputs(*anBoardOut, arInput)

			// s += fmt.Sprintf(" anBoardOut[0]: %v\n", anBoardOut[0])
			// s += fmt.Sprintf(" anBoardOut[1]: %v\n", anBoardOut[1])
			// s += fmt.Sprintf(" arInput: %v\n", arInput)

			// fmt.Printf("%v", s)

			// if 1 == 1 {
			// 	os.Exit(0)
			// }

			nets := []*_NeuralNet{&nnpRace, &nnpCrashed, &nnpContact}
			n := nets[pc-_CLASS_RACE]

			var nnState *_NNState
			if nnStates != nil {
				nnState = &(*nnStates)[pc-_CLASS_RACE]
				if i == 0 {
					nnState.state = _NNSTATE_INCREMENTAL
				} else {
					nnState.state = _NNSTATE_DONE
				}
			}

			neuralNetEvaluateSSE(n, arInput, &arOutput, nnState)

			if pc == _CLASS_RACE {
				/* special evaluation of backgammons
				 * overrides net output */
				evalRaceBG(*anBoardOut, &arOutput, _VARIATION_STANDARD)
			}
			sanityCheck(*anBoardOut, &arOutput)

			copy(ec.ar[:], arOutput[:])
			ec.ar[5] = 0.0
			cacheAdd(&cpEval, &ec, l)
		}

		pm.rScore = utilityME(&arOutput, pci)

		if i < prune_moves {
			bmovesi[i] = i
			if pm.rScore > ml.amMoves[bmovesi[0]].rScore {
				bmovesi[i] = bmovesi[0]
				bmovesi[0] = i
			}
		} else if pm.rScore < ml.amMoves[bmovesi[0]].rScore {
			var m int
			bmovesi[0] = i
			for k := 1; k < prune_moves; k++ {
				if ml.amMoves[bmovesi[k]].rScore > ml.amMoves[bmovesi[m]].rScore {
					m = k
				}
			}
			bmovesi[0] = bmovesi[m]
			bmovesi[m] = i
		}
	}

	pci.fMove = 1 - pci.fMove

	if i == ml.cMoves {
		scoreMovesPruned(tld, &ml, pci, pec, &bmovesi, prune_moves)
	} else {
		scoreMoves(tld, &ml, pci, pec, 0)
	}

	bestMove := &ml.amMoves[ml.iMoveBest]
	bestMove.key.toBoard(anBoardOut)

	// s += fmt.Sprintf(" anBoardOut[0]: %v\n", anBoardOut[0])
	// s += fmt.Sprintf(" anBoardOut[1]: %v\n", anBoardOut[1])

	// fmt.Printf("%v", s)

	// if 1 == 1 {
	// 	os.Exit(0)
	// }
}

func findBestMovePlied(anMove *[8]int, nDice0 int, nDice1 int, anBoard *_TanBoard, pci *_CubeInfo, pec *_EvalContext, nPlies int, aamf *[_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter) (int, error) {
	var ec _EvalContext
	var ml _MoveList

	ec = *pec
	ec.nPlies = nPlies

	if anMove != nil {
		for i := 0; i < 8; i++ {
			anMove[i] = -1
		}
	}

	if err := findnSaveBestMoves(&ml, nDice0, nDice1, *anBoard, nil, 0.0, pci, &ec, aamf); err != nil {
		ml.amMoves = nil
		return -1, fmt.Errorf("error in findnSaveBestMoves: %v", err)
	}

	if anMove != nil {
		for i := 0; i < ml.cMaxMoves*2; i++ {
			anMove[i] = ml.amMoves[ml.iMoveBest].anMove[i]
		}
	}

	if ml.cMoves > 0 {
		move := &ml.amMoves[ml.iMoveBest]
		move.key.toBoard(anBoard)
	}
	ml.amMoves = nil

	return ml.cMaxMoves * 2, nil
}

func sortMoves(a []_Move) {
	sort.Slice(a, func(i, j int) bool {
		return compareMoves(&a[i], &a[j]) < 0
	})
}

func compareMoves(pm0 *_Move, pm1 *_Move) int {
	/*high score first */
	if pm1.rScore > pm0.rScore || (pm1.rScore == pm0.rScore && pm1.rScore2 > pm0.rScore2) {
		return 1
	} else {
		return -1
	}
}

var _NullFilter = _MoveFilter{0, 0, 0.0}

func findnSaveBestMoves(pml *_MoveList, nDice0 int, nDice1 int, anBoard _TanBoard, keyMove *_PositionKey, rThr float32, pci *_CubeInfo, pec *_EvalContext, aamf *[_MAX_FILTER_PLIES][_MAX_FILTER_PLIES]_MoveFilter) error {
	/* Find best moves.
	 * Ensure that keyMove is evaluated at the deepest ply. */

	var nMoves int
	var pm []_Move
	var mFilters *[_MAX_FILTER_PLIES]_MoveFilter
	var nMaxPly int
	var cOldMoves int

	// TODO: what level should this be at?
	tld := _ThreadLocalData{}

	/* Find all moves -- note that pml contains internal pointers to static
	 * data, so we can't call GenerateMoves again (or anything that calls
	 * it, such as ScoreMoves at more than 0 plies) until we have saved
	 * the moves we want to keep in amCandidates. */
	generateMoves(&tld, pml, anBoard, nDice0, nDice1, false)

	if pml.cMoves == 0 {
		/* no legal moves */
		pml.amMoves = nil
		return nil
	}

	/* Save moves */
	pm = make([]_Move, pml.cMoves)
	copy(pm, pml.amMoves)
	pml.amMoves = pm
	nMoves = pml.cMoves

	if pec.nPlies > 0 && pec.nPlies <= _MAX_FILTER_PLIES {
		mFilters = &aamf[pec.nPlies-1]
	} else {
		mFilters = &aamf[_MAX_FILTER_PLIES-1]
	}

	for iPly := 0; iPly < pec.nPlies; iPly++ {
		var mFilter *_MoveFilter
		if iPly < _MAX_FILTER_PLIES {
			mFilter = &mFilters[iPly]
		} else {
			mFilter = &_NullFilter
		}

		var k int

		if mFilter.accept < 0 {
			continue
		}

		if err := scoreMoves(&tld, pml, pci, pec, iPly); err != nil {
			pml.cMoves = 0
			pml.amMoves = nil
			return fmt.Errorf("erron in scoreMoves: %v", err)
		}

		sortMoves(pml.amMoves)
		pml.iMoveBest = 0

		k = pml.cMoves
		/* we check for mFilter->Accept < 0 above */
		pml.cMoves = math32.Imin(mFilter.accept, pml.cMoves)

		{
			var limit int = math32.Imin(k, pml.cMoves+mFilter.extra)

			for ; pml.cMoves < limit; pml.cMoves++ {
				if pml.amMoves[pml.cMoves].rScore < pml.amMoves[0].rScore-mFilter.threshold {
					break
				}
			}
		}

		nMaxPly = iPly

		if pml.cMoves == 1 && mFilter.accept != 1 {
			/* if there is only one move to evaluate there is no need to continue */
			goto finished
		}
	}

	/* evaluate moves on top ply */

	if err := scoreMoves(&tld, pml, pci, pec, pec.nPlies); err != nil {
		pml.cMoves = 0
		pml.amMoves = nil
		return fmt.Errorf("error in scoreMoves: %v", err)
	}

	nMaxPly = pec.nPlies

	/* Resort the moves, in case the new evaluation reordered them. */
	sortMoves(pml.amMoves)
	pml.iMoveBest = 0

	/* set the proper size of the movelist */

finished:

	cOldMoves = pml.cMoves
	pml.cMoves = nMoves

	/* Make sure that keyMove and top move are both
	 * evaluated at the deepest ply. */
	if keyMove != nil {
		fResort := false

		for i := 0; i < pml.cMoves; i++ {
			if keyMove.equals(pml.amMoves[i].key) {
				/* ensure top move is evaluted at deepest ply */

				if pml.amMoves[i].esMove.ec.nPlies < nMaxPly {
					if err := scoreMove(&tld, nil, &pml.amMoves[i], pci, pec, nMaxPly); err != nil {
						logWarningf("error in scoreMove: %v", err)
					}
					fResort = true
				}

				if (math32.Fabsf(pml.amMoves[i].rScore-pml.amMoves[0].rScore) > rThr) && (nMaxPly < pec.nPlies) {
					/* this is an error/blunder: re-analyse at top-ply */
					if err := scoreMove(&tld, nil, &pml.amMoves[0], pci, pec, pec.nPlies); err != nil {
						logWarningf("error in scoreMove: %v", err)
					}
					if err := scoreMove(&tld, nil, &pml.amMoves[i], pci, pec, pec.nPlies); err != nil {
						logWarningf("error in scoreMove: %v", err)
					}
					cOldMoves = 1 /* only one move scored at deepest ply */
					fResort = true
				}

				/* move it up to the other moves evaluated on nMaxPly */

				if fResort && pec.nPlies > 0 {
					var m []_Move = make([]_Move, cOldMoves-i)
					copy(m, pml.amMoves[i:])

					for j := i - 1; j >= cOldMoves; j-- {
						copy(pml.amMoves[j+1:], pml.amMoves[j:])
					}
					copy(pml.amMoves[cOldMoves:], m)

					/* reorder moves evaluated on nMaxPly */
					sortMoves(pml.amMoves)
				}
				break
			}
		}
	}

	return nil

}

func sanityCheck(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32) {
	var nciq int
	var ac [2]int
	var anBack [2]int
	var anCross [2]int
	var anGammonCross [2]int
	var anMaxTurns [2]int
	var fContact bool

	if !(arOutput[_OUTPUT_WIN] >= 0.0 && arOutput[_OUTPUT_WIN] <= 1.0) {
		panic("invalid OUTPUT_WIN")
	}
	if !(arOutput[_OUTPUT_WINGAMMON] >= 0.0 && arOutput[_OUTPUT_WINGAMMON] <= 1.0) {
		panic("invalid OUTPUT_WINGAMMON")
	}
	if !(arOutput[_OUTPUT_WINBACKGAMMON] >= 0.0 && arOutput[_OUTPUT_WINBACKGAMMON] <= 1.0) {
		panic("invalid OUTPUT_WINBACKGAMMON")
	}
	if !(arOutput[_OUTPUT_LOSEGAMMON] >= 0.0 && arOutput[_OUTPUT_LOSEGAMMON] <= 1.0) {
		panic("invalid OUTPUT_LOSEGAMMON")
	}
	if !(arOutput[_OUTPUT_LOSEBACKGAMMON] >= 0.0 && arOutput[_OUTPUT_LOSEBACKGAMMON] <= 1.0) {
		panic("invalid OUTPUT_LOSEBACKGAMMON")
	}

	anGammonCross[0] = 1
	anGammonCross[1] = 1

	for j := 0; j < 2; j++ {
		nciq = 0
		for i := 0; i < 6; i++ {
			if anBoard[j][i] > 0 {
				anBack[j] = i
				nciq += anBoard[j][i]
			}
		}
		ac[j] = nciq
		anCross[j] = nciq

		nciq = 0
		for i := 6; i < 12; i++ {
			if anBoard[j][i] > 0 {
				anBack[j] = i
				nciq += anBoard[j][i]
			}
		}
		ac[j] += nciq
		anCross[j] += 2 * nciq
		anGammonCross[j] += nciq

		nciq = 0
		for i := 12; i < 18; i++ {
			if anBoard[j][i] > 0 {
				anBack[j] = i
				nciq += anBoard[j][i]
			}
		}
		ac[j] += nciq
		anCross[j] += 3 * nciq
		anGammonCross[j] += 2 * nciq

		nciq = 0
		for i := 18; i < 24; i++ {
			if anBoard[j][i] > 0 {
				anBack[j] = i
				nciq += anBoard[j][i]
			}
		}
		ac[j] += nciq
		anCross[j] += 4 * nciq
		anGammonCross[j] += 3 * nciq

		if anBoard[j][24] > 0 {
			anBack[j] = 24
			ac[j] += anBoard[j][24]
			anCross[j] += 5 * anBoard[j][24]
			anGammonCross[j] += 4 * anBoard[j][24]
		}
	}

	fContact = anBack[0]+anBack[1] >= 24

	if !fContact {
		for i := 0; i < 2; i++ {
			if anBack[i] < 6 && pbc1 != nil {
				anMaxTurns[i] = maxTurns(positionBearoff(anBoard.getHomeBoard(i), pbc1.nPoints, pbc1.nChequers))
			} else {
				anMaxTurns[i] = anCross[i] * 2
			}
		}
		if anMaxTurns[1] == 0 {
			anMaxTurns[1] = 1
		}
	}

	if (!fContact) && anCross[0] > 4*(anMaxTurns[1]-1) {
		/* Certain win */
		arOutput[_OUTPUT_WIN] = 1.0
	}
	if ac[0] < 15 {
		/* Opponent has borne off; no gammons or backgammons possible */
		arOutput[_OUTPUT_WINGAMMON] = 0.0
		arOutput[_OUTPUT_WINBACKGAMMON] = 0.0
	} else if !fContact {
		if anCross[1] > 8*anGammonCross[0] {
			/* Gammon impossible */
			arOutput[_OUTPUT_WINGAMMON] = 0.0
		} else if anGammonCross[0] > 4*(anMaxTurns[1]-1) {
			/* Certain gammon */
			arOutput[_OUTPUT_WINGAMMON] = 1.0
		}
		if anBack[0] < 18 {
			/* Backgammon impossible */
			arOutput[_OUTPUT_WINBACKGAMMON] = 0.0
		}
	}

	if (!fContact) && anCross[1] > 4*anMaxTurns[0] {
		/* Certain loss */
		arOutput[_OUTPUT_WIN] = 0.0
	}

	if ac[1] < 15 {
		/* Player has borne off; no gammon or backgammon losses possible */
		arOutput[_OUTPUT_LOSEGAMMON] = 0.0
		arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0
	} else if !fContact {
		if anCross[0] > 8*anGammonCross[1]-4 {
			/* Gammon loss impossible */
			arOutput[_OUTPUT_LOSEGAMMON] = 0.0
		} else if anGammonCross[1] > 4*anMaxTurns[0] {
			/* Certain gammon loss */
			arOutput[_OUTPUT_LOSEGAMMON] = 1.0
		}
		if anBack[1] < 18 {
			/* Backgammon impossible */
			arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0
		}
	}

	/* gammons must be less than wins */
	if arOutput[_OUTPUT_WINGAMMON] > arOutput[_OUTPUT_WIN] {
		arOutput[_OUTPUT_WINGAMMON] = arOutput[_OUTPUT_WIN]
	}

	var lose float32 = 1.0 - arOutput[_OUTPUT_WIN]
	if arOutput[_OUTPUT_LOSEGAMMON] > lose {
		arOutput[_OUTPUT_LOSEGAMMON] = lose
	}

	/* Backgammons cannot exceed gammons */
	if arOutput[_OUTPUT_WINBACKGAMMON] > arOutput[_OUTPUT_WINGAMMON] {
		arOutput[_OUTPUT_WINBACKGAMMON] = arOutput[_OUTPUT_WINGAMMON]
	}

	if arOutput[_OUTPUT_LOSEBACKGAMMON] > arOutput[_OUTPUT_LOSEGAMMON] {
		arOutput[_OUTPUT_LOSEBACKGAMMON] = arOutput[_OUTPUT_LOSEGAMMON]
	}

	if fContact {
		var noise float32 = 1 / 10000.0

		for i := _OUTPUT_WINGAMMON; i < _NUM_OUTPUTS; i++ {
			if arOutput[i] < noise {
				arOutput[i] = 0.0
			}
		}
	}

}

/* An upper bound on the number of turns it can take to complete a bearoff
 * from bearoff position ID i. */
func maxTurns(id int) int {
	var aus [32]int

	bearoffDist(pbc1, id, nil, nil, nil, &aus, nil)

	for i := 31; i >= 0; i-- {
		if aus[i] > 0 {
			return i
		}
	}

	return -1
}

func evalRaceBG(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation) {
	/* anBoard[1] is on roll */

	/* total men for side not on roll */
	var totMen0 int

	/* total men for side on roll */
	var totMen1 int

	/* a set flag for every possible outcome */
	var any int

	for i := 23; i >= 0; i-- {
		totMen0 += anBoard[0][i]
		totMen1 += anBoard[1][i]
	}

	if totMen1 == 15 {
		any |= _OG_POSSIBLE
	}

	if totMen0 == 15 {
		any |= _G_POSSIBLE
	}

	if any > 0 {
		if any&_OG_POSSIBLE > 0 {
			var i int
			for i = 23; i >= 18; i-- {
				if anBoard[1][i] > 0 {
					break
				}
			}
			if i >= 18 {
				any |= _OBG_POSSIBLE
			}
		}

		if any&_G_POSSIBLE > 0 {
			var i int
			for i = 23; i >= 18; i-- {
				if anBoard[0][i] > 0 {
					break
				}
			}

			if i >= 18 {
				any |= _BG_POSSIBLE
			}
		}
	}

	if any&(_BG_POSSIBLE|_OBG_POSSIBLE) > 0 {
		/* side that can have the backgammon */
		var side int
		if any&_BG_POSSIBLE > 0 {
			side = 1
		}

		pr := raceBGprob(anBoard, side, bgv)

		if pr > 0.0 {
			if side == 1 {
				arOutput[_OUTPUT_WINBACKGAMMON] = pr

				if arOutput[_OUTPUT_WINGAMMON] < arOutput[_OUTPUT_WINBACKGAMMON] {
					arOutput[_OUTPUT_WINGAMMON] = arOutput[_OUTPUT_WINBACKGAMMON]
				}
			} else {
				arOutput[_OUTPUT_LOSEBACKGAMMON] = pr

				if arOutput[_OUTPUT_LOSEGAMMON] < arOutput[_OUTPUT_LOSEBACKGAMMON] {
					arOutput[_OUTPUT_LOSEGAMMON] = arOutput[_OUTPUT_LOSEBACKGAMMON]
				}
			}
		} else {
			if side == 1 {
				arOutput[_OUTPUT_WINBACKGAMMON] = 0.0
			} else {
				arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0
			}
		}
	}
}

func (t *_TanBoard) getHomeBoard(side int) [6]int {
	var board [6]int
	copy(board[:], t[side][:])
	return board
}

/* side - side that potentially can win a backgammon */
/* Return - Probablity that side will win a backgammon */
func raceBGprob(anBoard _TanBoard, side int, bgv _BGVariation) float32 {
	var totMenHome int
	var totPipsOp int
	var dummy _TanBoard

	for i := 0; i < 6; i++ {
		totMenHome += anBoard[side][i]
	}

	for i := 22; i >= 18; i-- {
		totPipsOp += anBoard[1-side][i] * (i - 17)
	}

	if !((totMenHome+3)/4-side <= (totPipsOp+2)/3) {
		return 0.0
	}

	for i := 0; i < 25; i++ {
		dummy[side][i] = anBoard[side][i]
	}

	for i := 0; i < 6; i++ {
		dummy[1-side][i] = anBoard[1-side][18+i]
	}

	for i := 6; i < 25; i++ {
		dummy[1-side][i] = 0
	}

	var p float32
	bgp := getRaceBGprobs(dummy.getHomeBoard(1 - side))
	if bgp != nil {
		k := positionBearoff(anBoard.getHomeBoard(side), pbc1.nPoints, pbc1.nChequers)
		var aProb [32]int

		var scale int
		if side == 0 {
			scale = 36
		} else {
			scale = 1
		}

		bearoffDist(pbc1, k, nil, nil, nil, &aProb, nil)

		for j := 1 - side; j < _RBG_NPROBS; j++ {
			var sum int
			scale *= 36
			for i := 1; i <= j+side; i++ {
				sum += aProb[i]
			}
			p += float32(bgp[j]) / float32(scale) * float32(sum)
		}

		p /= 65535.0

	} else {
		var ar [5]float32

		if positionBearoff(dummy.getHomeBoard(0), 6, 15) > 923 || positionBearoff(dummy.getHomeBoard(1), 6, 15) > 923 {
			if err := evalBearoff1(dummy, &ar, bgv, nil); err != nil {
				logWarningf("error in evalBearoff1: %v", err)
			}
		} else {
			if err := evalBearoff2(dummy, &ar, bgv, nil); err != nil {
				logWarningf("error in evalBearoff2: %v", err)
			}
		}

		if side == 1 {
			p = ar[0]
		} else {
			p = 1.0 - ar[0]
		}
	}

	return math32.Min(p, 1.0)

}

func makeCubePos(aciCubePos []_CubeInfo, cci int, fTop bool, aci []_CubeInfo, fInvert bool) {
	for ici, i := 0, 0; ici < cci; ici++ {
		/* no double */
		if aciCubePos[ici].nCube > 0 {
			if err := setCubeInfo(&aci[i],
				aciCubePos[ici].nCube,
				aciCubePos[ici].fCubeOwner,
				btoi(fInvert != itob(aciCubePos[ici].fMove)),
				aciCubePos[ici].nMatchTo,
				aciCubePos[ici].anScore,
				aciCubePos[ici].fCrawford,
				aciCubePos[ici].fJacoby, aciCubePos[ici].fBeavers, aciCubePos[ici].bgv); err != nil {
				logWarningf("error in setCubeInfo: %v", err)
			}
		} else {
			aci[i].nCube = -1
		}
		i++
		if !fTop && aciCubePos[ici].nCube > 0 && getDPEq(nil, nil, aciCubePos[ici]) {
			/* we may double */
			if err := setCubeInfo(&aci[i],
				2*aciCubePos[ici].nCube,
				1-aciCubePos[ici].fMove,
				btoi(fInvert != itob(aciCubePos[ici].fMove)),
				aciCubePos[ici].nMatchTo,
				aciCubePos[ici].anScore,
				aciCubePos[ici].fCrawford,
				aciCubePos[ici].fJacoby, aciCubePos[ici].fBeavers, aciCubePos[ici].bgv); err != nil {
				logWarningf("error in setCubeInfo: %v", err)
			}
		} else {
			/* mark cube position as unavailable */
			aci[i].nCube = -1
		}
		i++
	} /* loop cci */
}

func getDPEq(pfCube *bool, prDPEq *float32, pci _CubeInfo) bool {
	var fCube bool

	if pci.nMatchTo == 0 {

		/* Money game:
		 * Double, pass equity for money game is 1.0 points, since we always
		 * calculate equity normed to a 1-cube.
		 * Take the double branch if the cube is centered or I own the cube. */

		if prDPEq != nil {
			*prDPEq = 1.0
		}
		fCube = (pci.fCubeOwner == -1) || (pci.fCubeOwner == pci.fMove)

		if pfCube != nil {
			*pfCube = fCube
		}
	} else {

		/* Match play:
		 * Equity for double, pass is found from the match equity table.
		 * Take the double branch is I can/will use cube:
		 * - if it is not the Crawford game,
		 * - and if the cube is not dead,
		 * - and if it is post-Crawford and I'm trailing
		 * - and if I have access to the cube.
		 */

		/* FIXME: equity for double, pass */
		fPostCrawford := !pci.fCrawford && (pci.anScore[0] == pci.nMatchTo-1 || pci.anScore[1] == pci.nMatchTo-1)

		fCube = (!pci.fCrawford) && (pci.anScore[pci.fMove]+pci.nCube < pci.nMatchTo) && (!(fPostCrawford && (pci.anScore[pci.fMove] == pci.nMatchTo-1))) && ((pci.fCubeOwner == -1) || (pci.fCubeOwner == pci.fMove))

		if prDPEq != nil {
			*prDPEq = getME(pci.anScore[0], pci.anScore[1], pci.nMatchTo, pci.fMove, pci.nCube, pci.fMove, pci.fCrawford, &aafMET, &aafMETPostCrawford)
		}
		if pfCube != nil {
			*pfCube = fCube
		}
	}

	return fCube

}

func setCubeInfo(pci *_CubeInfo, nCube int, fCubeOwner int, fMove int, nMatchTo int, anScore [2]int, fCrawford bool, fJacoby bool, fBeavers bool, bgv _BGVariation) error {
	if nMatchTo > 0 {
		return setCubeInfoMatch(pci, nCube, fCubeOwner, fMove, nMatchTo, anScore, fCrawford, bgv)
	} else {
		return setCubeInfoMoney(pci, nCube, fCubeOwner, fMove, fJacoby, fBeavers, bgv)
	}
}

func setCubeInfoMatch(pci *_CubeInfo, nCube int, fCubeOwner int, fMove int, nMatchTo int, anScore [2]int, fCrawford bool, bgv _BGVariation) error {
	if nCube < 1 || fCubeOwner < -1 || fCubeOwner > 1 || fMove < 0 || fMove > 1 || nMatchTo < 1 || anScore[0] >= nMatchTo || anScore[1] >= nMatchTo { /* FIXME also illegal if nCube is not a power of 2 */
		// pci = &_CubeInfo{}
		return fmt.Errorf("illegal arguments")
	}

	pci.nCube = nCube
	pci.fCubeOwner = fCubeOwner
	pci.fMove = fMove
	pci.fJacoby = false
	pci.fBeavers = false
	pci.nMatchTo = nMatchTo
	pci.anScore[0] = anScore[0]
	pci.anScore[1] = anScore[1]
	pci.fCrawford = fCrawford
	pci.bgv = bgv

	/*
	 * FIXME: calculate gammon price when initializing program
	 * instead of recalculating it again and again, or cache it.
	 */
	nAway0 := pci.nMatchTo - pci.anScore[0] - 1
	nAway1 := pci.nMatchTo - pci.anScore[1] - 1

	if (nAway0 == 0 || nAway1 == 0) && !fCrawford {
		if nAway0 == 0 {
			pci.arGammonPrice = aaaafGammonPricesPostCrawford[logCube(pci.nCube)][nAway1][0]
		} else {
			pci.arGammonPrice = aaaafGammonPricesPostCrawford[logCube(pci.nCube)][nAway0][1]
		}
	} else {
		pci.arGammonPrice = aaaafGammonPrices[logCube(pci.nCube)][nAway0][nAway1]
	}

	return nil
}

func setCubeInfoMoney(pci *_CubeInfo, nCube int, fCubeOwner int, fMove int, fJacoby bool, fBeavers bool, bgv _BGVariation) error {

	if nCube < 1 || fCubeOwner < -1 || fCubeOwner > 1 || fMove < 0 || fMove > 1 { /* FIXME also illegal if nCube is not a power of 2 */
		// pci = &_CubeInfo{}
		return fmt.Errorf("illegal arguments")
	}

	pci.nCube = nCube
	pci.fCubeOwner = fCubeOwner
	pci.fMove = fMove
	pci.fJacoby = fJacoby
	pci.fBeavers = fBeavers
	pci.nMatchTo = 0
	pci.anScore[0] = 0
	pci.anScore[1] = 0
	pci.fCrawford = false
	pci.bgv = bgv

	var gp float32 = 1.0
	if fJacoby && fCubeOwner == 1 {
		gp = 0.0
	}
	pci.arGammonPrice[0] = gp
	pci.arGammonPrice[1] = gp
	pci.arGammonPrice[2] = gp
	pci.arGammonPrice[3] = gp

	return nil
}

func classifyPosition(anBoard _TanBoard, bgv _BGVariation) _PositionClass {
	var nOppBack, nBack int = -1, -1

	for nOppBack = 24; nOppBack >= 0; nOppBack-- {
		if anBoard[0][nOppBack] != 0 {
			break
		}
	}

	for nBack = 24; nBack >= 0; nBack-- {
		if anBoard[1][nBack] != 0 {
			break
		}
	}

	if nBack < 0 || nOppBack < 0 {
		return _CLASS_OVER
	}

	/* special classes for hypergammon variants */

	switch bgv {
	case _VARIATION_HYPERGAMMON_1:
		return _CLASS_HYPERGAMMON1

	case _VARIATION_HYPERGAMMON_2:
		return _CLASS_HYPERGAMMON2

	case _VARIATION_HYPERGAMMON_3:
		return _CLASS_HYPERGAMMON3

	case _VARIATION_STANDARD, _VARIATION_NACKGAMMON:

		/* normal backgammon */

		if nBack+nOppBack > 22 {

			/* contact position */
			N := 6

			for side := 0; side < 2; side++ {
				tot := 0

				var board [25]int = anBoard[side]

				for i := 0; i < 25; i++ {
					tot += board[i]
				}

				if tot <= N {
					return _CLASS_CRASHED
				} else {
					if board[0] > 1 {
						if tot <= (N + board[0]) {
							return _CLASS_CRASHED
						} else {
							if (1+tot-(board[0]+board[1]) <= N) && board[1] > 1 {
								return _CLASS_CRASHED
							}
						}
					} else {
						if tot <= (N + (board[1] - 1)) {
							return _CLASS_CRASHED
						}
					}
				}
			}

			return _CLASS_CONTACT
		} else {

			if isBearoff(pbc2, anBoard) {
				return _CLASS_BEAROFF2
			}

			if isBearoff(pbcTS, anBoard) {
				return _CLASS_BEAROFF_TS
			}

			if isBearoff(pbc1, anBoard) {
				return _CLASS_BEAROFF1
			}

			if isBearoff(pbcOS, anBoard) {
				return _CLASS_BEAROFF_OS
			}

			return _CLASS_RACE

		}

	default:
		panic("invalid position class")

	}
}

func swapSides(anBoard *_TanBoard) {
	for i := 0; i < 25; i++ {
		n := anBoard[0][i]
		anBoard[0][i] = anBoard[1][i]
		anBoard[1][i] = n
	}
}

/*
 * Utility returns the "correct" cubeless equity based on the current
 * gammon values.
 *
 * Use UtilityME to get the "true" money equity.
 */
func utility(ar *[_NUM_OUTPUTS]float32, pci *_CubeInfo) float32 {
	if pci.nMatchTo == 0 {
		/* equity calculation for money game */
		/* For money game the gammon price is the same for both
		 * players, so there is no need to use pci->fMove. */
		return ar[_OUTPUT_WIN]*2.0 - 1.0 + (ar[_OUTPUT_WINGAMMON]-ar[_OUTPUT_LOSEGAMMON])*pci.arGammonPrice[0] + (ar[_OUTPUT_WINBACKGAMMON]-ar[_OUTPUT_LOSEBACKGAMMON])*pci.arGammonPrice[1]

	} else {
		/* equity calculation for match play */
		return ar[_OUTPUT_WIN]*2.0 - 1.0 + ar[_OUTPUT_WINGAMMON]*pci.arGammonPrice[pci.fMove] - ar[_OUTPUT_LOSEGAMMON]*pci.arGammonPrice[(1-pci.fMove)] + ar[_OUTPUT_WINBACKGAMMON]*pci.arGammonPrice[2+pci.fMove] - ar[_OUTPUT_LOSEBACKGAMMON]*pci.arGammonPrice[2+(1-pci.fMove)]
	}
}

/*
 * UtilityME is identical to Utility for match play.
 * For money play it returns the money equity instead of the
 * correct cubeless equity.
 */
func utilityME(ar *[_NUM_OUTPUTS]float32, pci *_CubeInfo) float32 {
	if pci.nMatchTo == 0 {
		/* calculate money equity */
		return ar[_OUTPUT_WIN]*2.0 - 1.0 + (ar[_OUTPUT_WINGAMMON] - ar[_OUTPUT_LOSEGAMMON]) + (ar[_OUTPUT_WINBACKGAMMON] - ar[_OUTPUT_LOSEBACKGAMMON])
	} else {
		return utility(ar, pci)
	}
}

func bearoffHyper(pbc *_BearOffContext, iPos int, arOutput *[_NUM_OUTPUTS]float32, arEquity *[4]float32) error {
	return readHypergammon(pbc, iPos, arOutput, arEquity)
}

func evaluatePerfectCubeful(anBoard _TanBoard, arEquity *[4]float32, bgv _BGVariation) error {
	pc := classifyPosition(anBoard, bgv)

	switch pc {
	case _CLASS_BEAROFF2:
		return perfectCubeful(pbc2, anBoard, arEquity)
	case _CLASS_BEAROFF_TS:
		return perfectCubeful(pbcTS, anBoard, arEquity)
	}

	return fmt.Errorf("invalid position: %v", pc)
}

func perfectCubeful(pbc *_BearOffContext, anBoard _TanBoard, arEquity *[4]float32) error {
	nUs := positionBearoff(anBoard.getHomeBoard(1), pbc.nPoints, pbc.nChequers)
	nThem := positionBearoff(anBoard.getHomeBoard(0), pbc.nPoints, pbc.nChequers)
	n := combination(pbc.nPoints+pbc.nChequers, pbc.nPoints)
	iPos := nUs*n + nThem

	return bearoffCubeful(pbc, iPos, arEquity, nil)
}

func bearoffCubeful(pbc *_BearOffContext, iPos int, ar *[4]float32, aus *[4]int) error {
	if pbc == nil {
		return fmt.Errorf("pbc not supplied")
	}
	if !pbc.fCubeful {
		return fmt.Errorf("not a cubeful game")
	}

	readTwoSidedBearoff(pbc, iPos, ar, aus)

	return nil
}

func evaluatePosition(tld *_ThreadLocalData, nnStates *[3]_NNState, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, pec *_EvalContext) error {
	// var s string

	// s += "=== evaluatePosition()\n"
	// s += fmt.Sprintf(" anBoard[0]: %v\n", anBoard[0])
	// s += fmt.Sprintf(" anBoard[1]: %v\n", anBoard[1])

	pc := classifyPosition(anBoard, pci.bgv)

	var pecx *_EvalContext
	var nPlies int

	if pec != nil {
		pecx = pec
		nPlies = pec.nPlies
	} else {
		pecx = &ecBasic
		nPlies = 0
	}

	r := evaluatePositionCache(tld, nnStates, anBoard, arOutput, pci, pecx, nPlies, pc)

	// s += fmt.Sprintf(" arOutput: %v\n", arOutput)
	// fmt.Print(s)

	// if 1 == 1 {
	// 	os.Exit(0)
	// }

	return r
}

func evaluatePositionCache(tld *_ThreadLocalData, nnStates *[3]_NNState, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, pecx *_EvalContext, nPlies int, pc _PositionClass) error {
	var ec _CacheNodeDetail
	/* This should be a part of the code that is called in all
	 * time-consuming operations at a relatively steady rate, so is a
	 * good choice for a callback function. */
	if cCache == 0 || pecx.rNoise != 0.0 { /* non-deterministic noisy evaluations; cannot cache */
		return evaluatePositionFull(tld, nnStates, anBoard, arOutput, pci, pecx, nPlies, pc)
	}

	ec.key.fromBoard(anBoard)

	ec.nEvalContext = evalKey(pecx, nPlies, pci, false)
	hit, l := cacheLookup(&cEval, &ec, arOutput, nil)
	if hit {
		return nil
	}

	if err := evaluatePositionFull(tld, nnStates, anBoard, arOutput, pci, pecx, nPlies, pc); err != nil {
		return fmt.Errorf("error in evaluatePositionFull: %v", err)
	}

	copy(ec.ar[:], arOutput[:])
	ec.ar[5] = 0
	cacheAdd(&cEval, &ec, l)

	return nil
}

func evalKey(pec *_EvalContext, nPlies int, pci *_CubeInfo, fCubefulEquity bool) int {

	var iKey int
	/*
	 * Bit 00-03: nPlies
	 * Bit 04   : fCubeful
	 * Bit 05   : fMove
	 * Bit 06   : fUsePrune
	 * Bit 07-12: anScore[ 0 ]
	 * Bit 13-18: anScore[ 1 ]
	 * Bit 19-22: log2(nCube)
	 * Bit 23-24: fCubeOwner
	 * Bit 25   : fCrawford
	 * Bit 26   : fJacoby
	 * Bit 27   : fBeavers
	 */

	iKey = (nPlies | (btoi(pec.fCubeful) << 4) | (pci.fMove << 5))

	if nPlies > 0 {
		iKey ^= (btoi(pec.fUsePrune) << 6)
	}

	if nPlies > 0 || fCubefulEquity {
		var fCubeOwner int
		if pci.fCubeOwner < 0 {
			fCubeOwner = 2
		} else {
			fCubeOwner = btoi(pci.fCubeOwner == pci.fMove)
		}
		/* In match play, the score and cube value and position are important. */
		if pci.nMatchTo > 0 {
			iKey ^=
				((pci.nMatchTo - pci.anScore[pci.fMove] - 1) << 7) ^
					((pci.nMatchTo - pci.anScore[1-pci.fMove] - 1) << 13) ^
					(logCube(pci.nCube) << 19) ^
					((fCubeOwner) << 23) ^ (btoi(pci.fCrawford) << 25)
		} else if pec.fCubeful || fCubefulEquity {
			/* in cubeful money games the cube position and rules are important. */
			iKey ^=
				((fCubeOwner) << 23) ^ (btoi(pci.fJacoby) << 26) ^ (btoi(pci.fBeavers) << 27)
		}
		if fCubefulEquity {
			iKey ^= 0x6a47b47e
		}
	}

	return iKey

}

func evaluatePositionFull(tld *_ThreadLocalData, nnStates *[3]_NNState, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, pec *_EvalContext, nPlies int, pc _PositionClass) error {
	var arVariationOutput [_NUM_OUTPUTS]float32

	if pc > _CLASS_PERFECT && nPlies > 0 {
		/* internal node; recurse */

		var anBoardNew _TanBoard
		/* int anMove[ 8 ]; */
		var ciOpp _CubeInfo
		var rTemp float32

		usePrune := pec.fUsePrune && pec.rNoise == 0.0 && pci.bgv == _VARIATION_STANDARD

		for i := 0; i < _NUM_OUTPUTS; i++ {
			arOutput[i] = 0.0
		}

		/* loop over rolls */
		for n0 := 1; n0 <= 6; n0++ {
			for n1 := 1; n1 <= n0; n1++ {
				var w float32
				if n0 == n1 {
					w = 1.0
				} else {
					w = 2.0
				}

				for i := 0; i < 25; i++ {
					anBoardNew[0][i] = anBoard[0][i]
					anBoardNew[1][i] = anBoard[1][i]
				}

				if fInterrupt {
					return fmt.Errorf("interrupted")
				}

				if usePrune {
					findBestMoveInEval(tld, nnStates, n0, n1, anBoard, &anBoardNew, pci, pec)
				} else {

					findBestMovePlied(nil, n0, n1, &anBoardNew, pci, pec, 0, &defaultFilters)
				}

				swapSides(&anBoardNew)

				setCubeInfo(&ciOpp, pci.nCube, pci.fCubeOwner, 1-pci.fMove, pci.nMatchTo, pci.anScore, pci.fCrawford, pci.fJacoby, pci.fBeavers, pci.bgv)

				/* Evaluate at 0-ply */
				if err := evaluatePositionCache(tld, nnStates, anBoardNew, &arVariationOutput, &ciOpp, pec, nPlies-1, classifyPosition(anBoardNew, ciOpp.bgv)); err != nil {
					return fmt.Errorf("error in evaluatePositionCache: %v", err)
				}
				for i := 0; i < _NUM_OUTPUTS; i++ {
					arOutput[i] += w * arVariationOutput[i]
				}
			}

		}

		/* normalize */
		for i := 0; i < _NUM_OUTPUTS; i++ {
			arOutput[i] /= 36
		}

		/* flop eval */
		arOutput[_OUTPUT_WIN] = 1.0 - arOutput[_OUTPUT_WIN]

		rTemp = arOutput[_OUTPUT_WINGAMMON]
		arOutput[_OUTPUT_WINGAMMON] = arOutput[_OUTPUT_LOSEGAMMON]
		arOutput[_OUTPUT_LOSEGAMMON] = rTemp

		rTemp = arOutput[_OUTPUT_WINBACKGAMMON]
		arOutput[_OUTPUT_WINBACKGAMMON] = arOutput[_OUTPUT_LOSEBACKGAMMON]
		arOutput[_OUTPUT_LOSEBACKGAMMON] = rTemp

	} else {
		/* at leaf node; use static evaluation */

		if err := acef[pc](anBoard, arOutput, pci.bgv, nnStates); err != nil {
			return fmt.Errorf("error in acef: %v", err)
		}

		if pec.rNoise > 0.0 && pc != _CLASS_OVER {
			for i := 0; i < _NUM_OUTPUTS; i++ {
				arOutput[i] += noise(pec, anBoard, i)
				arOutput[i] = math32.Max(arOutput[i], 0.0)
				arOutput[i] = math32.Min(arOutput[i], 1.0)
			}
		}

		if pc > _CLASS_GOOD || pec.rNoise > 0.0 {
			/* no sanity check needed for accurate evaluations */
			sanityCheck(anBoard, arOutput)
		}
	}

	return nil
}

var acef = [_N_CLASSES]classEvalFunc{
	evalOver,
	evalHypergammon1,
	evalHypergammon2,
	evalHypergammon3,
	evalBearoff2, evalBearoffTS,
	evalBearoff1, evalBearoffOS,
	evalRace, evalCrashed, evalContact,
}

func evalRace(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	var arInput []float32 = make([]float32, _NUM_RACE_INPUTS)

	calculateRaceInputs(anBoard, arInput)

	var pnState *_NNState
	if nnStates != nil {
		pnState = &nnStates[0] // [_CLASS_RACE-_CLASS_RACE]
	}

	if err := neuralNetEvaluateSSE(&nnRace, arInput, arOutput, pnState); err != nil {
		return fmt.Errorf("error in %v", err)
	}
	/* special evaluation of backgammons overrides net output */

	evalRaceBG(anBoard, arOutput, bgv)

	/* sanity check will take care of rest */

	return nil
}

func evalContact(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	var arInput []float32 = make([]float32, _NUM_INPUTS)

	calculateContactInputs(anBoard, arInput)

	var pnState *_NNState
	if nnStates != nil {
		pnState = &nnStates[_CLASS_CONTACT-_CLASS_RACE]
	}

	return neuralNetEvaluateSSE(&nnContact, arInput, arOutput, pnState)
}

func evalCrashed(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	var arInput []float32 = make([]float32, _NUM_INPUTS)

	calculateCrashedInputs(anBoard, arInput)

	var pnState *_NNState
	if nnStates != nil {
		pnState = &nnStates[_CLASS_CRASHED-_CLASS_RACE]
	}

	return neuralNetEvaluateSSE(&nnCrashed, arInput, arOutput, pnState)

}

func evalOver(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	var i, c int
	var n int = anChequers[bgv]

	for i = 0; i < 25; i++ {
		if anBoard[0][i] > 0 {
			break
		}
	}

	if i == 25 {
		/* opponent has no pieces on board; player has lost */
		arOutput[_OUTPUT_WIN] = 0.0
		arOutput[_OUTPUT_WINGAMMON] = 0.0
		arOutput[_OUTPUT_WINBACKGAMMON] = 0.0

		for i, c = 0, 0; i < 25; i++ {
			c += anBoard[1][i]
		}

		if c == n {
			/* player still has all pieces on board; loses gammon */
			arOutput[_OUTPUT_LOSEGAMMON] = 1.0

			for i = 18; i < 25; i++ {
				if anBoard[1][i] > 0 {
					/* player still has pieces in opponent's home board;
					 * loses backgammon */
					arOutput[_OUTPUT_LOSEBACKGAMMON] = 1.0

					return nil
				}
			}

			arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0

			return nil
		}

		arOutput[_OUTPUT_LOSEGAMMON] = 0.0
		arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0

		return nil
	}

	for i = 0; i < 25; i++ {
		if anBoard[1][i] > 0 {
			break
		}
	}

	if i == 25 {
		/* player has no pieces on board; wins */
		arOutput[_OUTPUT_WIN] = 1.0
		arOutput[_OUTPUT_LOSEGAMMON] = 0.0
		arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0

		for i, c = 0, 0; i < 25; i++ {
			c += anBoard[0][i]
		}

		if c == n {
			/* opponent still has all pieces on board; win gammon */
			arOutput[_OUTPUT_WINGAMMON] = 1.0

			for i = 18; i < 25; i++ {
				if anBoard[0][i] > 0 {
					/* opponent still has pieces in player's home board;
					 * win backgammon */
					arOutput[_OUTPUT_WINBACKGAMMON] = 1.0

					return nil
				}
			}

			arOutput[_OUTPUT_WINBACKGAMMON] = 0.0

			return nil
		}

		arOutput[_OUTPUT_WINGAMMON] = 0.0
		arOutput[_OUTPUT_WINBACKGAMMON] = 0.0
	}

	return nil

}

func evalBearoff2(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	if pbc2 == nil {
		panic("pbc2 == nil")
	}
	return bearoffEval(pbc2, anBoard, arOutput)
}

func evalBearoffOS(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(pbcOS, anBoard, arOutput)
}

func evalBearoffTS(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(pbcTS, anBoard, arOutput)
}

func evalHypergammon1(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(apbcHyper[0], anBoard, arOutput)
}

func evalHypergammon2(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(apbcHyper[1], anBoard, arOutput)
}

func evalHypergammon3(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(apbcHyper[2], anBoard, arOutput)
}

func evalBearoff1(anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32, bgv _BGVariation, nnStates *[3]_NNState) error {
	return bearoffEval(pbc1, anBoard, arOutput)
}

func calculateRaceInputs(anBoard _TanBoard, inputs []float32) {
	for side := 0; side < 2; side++ {
		board := anBoard[side]
		afInput := inputs[side*_HALF_RACE_INPUTS:]

		var menOff int = 15

		{
			if !(board[23] == 0 && board[24] == 0) {
				panic("invalid board")
			}
		}

		/* Points */
		for i := 0; i < 23; i++ {
			var nc int = board[i]

			k := i * 4

			menOff -= nc

			if nc == 1 {
				afInput[k] = 1.0
			} else {
				afInput[k] = 0.0
			}
			k++
			if nc == 2 {
				afInput[k] = 1.0
			} else {
				afInput[k] = 0.0
			}
			k++
			if nc >= 3 {
				afInput[k] = 1.0
			} else {
				afInput[k] = 0.0
			}
			k++
			if nc > 3 {
				afInput[k] = float32(nc-3) / 2.0
			} else {
				afInput[k] = 0.0
			}
		}

		/* Men off */
		for k := 0; k < 14; k++ {
			if menOff == (k + 1) {
				afInput[_RI_OFF+k] = 1.0
			} else {
				afInput[_RI_OFF+k] = 0.0
			}
		}

		{
			var nCross int

			for k := 1; k < 4; k++ {
				for i := 6 * k; i < 6*k+6; i++ {
					nc := board[i]

					if nc > 0 {
						nCross += nc * k
					}
				}
			}

			afInput[_RI_NCROSS] = float32(nCross) / 10.0
		}
	}
}

/* Calculates contact neural net inputs from the board position. */
func calculateContactInputs(anBoard _TanBoard, arInput []float32) {
	baseInputs(anBoard, arInput)

	{
		b := arInput[_MINPPERPOINT*25*2:]

		/* I accidentally switched sides (0 and 1) when I trained the net */
		menOffNonCrashed(anBoard[0], b[_I_OFF1:])

		calculateHalfInputs(anBoard[1], anBoard[0], b)
	}

	{
		b := arInput[(_MINPPERPOINT*25*2 + _MORE_INPUTS):]

		menOffNonCrashed(anBoard[1], b[_I_OFF1:])

		calculateHalfInputs(anBoard[0], anBoard[1], b)
	}
}

/* Calculates crashed neural net inputs from the board position. */
func calculateCrashedInputs(anBoard _TanBoard, arInput []float32) {
	baseInputs(anBoard, arInput)

	{
		b := arInput[_MINPPERPOINT*25*2:]

		menOffAll(anBoard[1], b[_I_OFF1:])

		calculateHalfInputs(anBoard[1], anBoard[0], b)
	}

	{
		b := arInput[(_MINPPERPOINT*25*2 + _MORE_INPUTS):]

		menOffAll(anBoard[0], b[_I_OFF1:])

		calculateHalfInputs(anBoard[0], anBoard[1], b)
	}
}

func menOffNonCrashed(anBoard [25]int, afInput []float32) {
	var menOff int = 15

	for i := 0; i < 25; i++ {
		menOff -= anBoard[i]
	}
	if menOff > 8 {
		panic("menOff > 8")
	}

	if menOff <= 2 {
		if menOff > 0 {
			afInput[0] = float32(menOff) / 3.0
		} else {
			afInput[0] = 0.0
		}
		afInput[1] = 0.0
		afInput[2] = 0.0
	} else if menOff <= 5 {
		afInput[0] = 1.0
		afInput[1] = float32(menOff-3) / 3.0
		afInput[2] = 0.0
	} else {
		afInput[0] = 1.0
		afInput[1] = 1.0
		afInput[2] = float32(menOff-6) / 3.0
	}

}

func menOffAll(anBoard [25]int, afInput []float32) {
	/* Men off */
	var menOff int = 15

	for i := 0; i < 25; i++ {
		menOff -= anBoard[i]
	}

	if menOff <= 5 {
		if menOff > 0 {
			afInput[0] = float32(menOff) / 5.0
		} else {
			afInput[0] = 0.0
		}
		afInput[1] = 0.0
		afInput[2] = 0.0
	} else if menOff <= 10 {
		afInput[0] = 1.0
		afInput[1] = float32(menOff-5) / 5.0
		afInput[2] = 0.0
	} else {
		afInput[0] = 1.0
		afInput[1] = 1.0
		afInput[2] = (float32(menOff) - 10) / 5.0
	}
}

func noise(pec *_EvalContext, anBoard _TanBoard, iOutput int) float32 {
	const UB4MAXVAL = 0xffffffff
	var r float32

	if pec.fDeterministic {
		var auchBoard [50]byte
		var auch [16]byte

		for i := 0; i < 25; i++ {
			auchBoard[i<<1] = byte(anBoard[0][i])
			auchBoard[(i<<1)+1] = byte(anBoard[1][i])
		}

		auchBoard[0] += byte(iOutput)

		auch = md5.Sum(auchBoard[:])

		/* We can't use a Box-Muller transform here, because generating
		 * a point in the unit circle requires a potentially unbounded
		 * number of integers, and all we have is the board.  So we
		 * just take the sum of the bytes in the hash, which (by the
		 * central limit theorem) should have a normal-ish distribution. */

		r = 0.0
		for i := 0; i < 16; i++ {
			r += float32(auch[i])
		}

		r -= 2040.0
		r /= 295.6
	} else {
		/* Box-Muller transform of a point in the unit circle. */
		var x, y float32

		for {
			x = float32(math32.Irand())*2.0/float32(UB4MAXVAL) - 1.0
			y = float32(math32.Irand())*2.0/float32(UB4MAXVAL) - 1.0
			r = x*x + y*y
			if !(r > 1.0 || r == 0.0) {
				break
			}
		}

		r = y * math32.Sqrtf(-2.0*math32.Logf(r)/r)
		// (void) x;
	}

	r *= pec.rNoise

	if iOutput == _OUTPUT_WINGAMMON || iOutput == _OUTPUT_LOSEGAMMON {
		r *= 0.25
	} else if iOutput == _OUTPUT_WINBACKGAMMON || iOutput == _OUTPUT_LOSEBACKGAMMON {
		r *= 0.01
	}

	return r
}

/* Calculates inputs for any contact position, for one player only. */
func calculateHalfInputs(anBoard [25]int, anBoardOpp [25]int, afInput []float32) {
	var nOppBack int
	var aHit [39]int
	var nBoard int

	/* aanCombination[n] -
	 * How many ways to hit from a distance of n pips.
	 * Each number is an index into aIntermediate below.
	 */
	var aanCombination = [24][5]int{
		{0, -1, -1, -1, -1},  /*  1 */
		{1, 2, -1, -1, -1},   /*  2 */
		{3, 4, 5, -1, -1},    /*  3 */
		{6, 7, 8, 9, -1},     /*  4 */
		{10, 11, 12, -1, -1}, /*  5 */
		{13, 14, 15, 16, 17}, /*  6 */
		{18, 19, 20, -1, -1}, /*  7 */
		{21, 22, 23, 24, -1}, /*  8 */
		{25, 26, 27, -1, -1}, /*  9 */
		{28, 29, -1, -1, -1}, /* 10 */
		{30, -1, -1, -1, -1}, /* 11 */
		{31, 32, 33, -1, -1}, /* 12 */
		{-1, -1, -1, -1, -1}, /* 13 */
		{-1, -1, -1, -1, -1}, /* 14 */
		{34, -1, -1, -1, -1}, /* 15 */
		{35, -1, -1, -1, -1}, /* 16 */
		{-1, -1, -1, -1, -1}, /* 17 */
		{36, -1, -1, -1, -1}, /* 18 */
		{-1, -1, -1, -1, -1}, /* 19 */
		{37, -1, -1, -1, -1}, /* 20 */
		{-1, -1, -1, -1, -1}, /* 21 */
		{-1, -1, -1, -1, -1}, /* 22 */
		{-1, -1, -1, -1, -1}, /* 23 */
		{38, -1, -1, -1, -1}, /* 24 */
	}

	/* One way to hit */
	type _Inter struct {
		/* if true, all intermediate points (if any) are required;
		 * if false, one of two intermediate points are required.
		 * Set to true for a direct hit, but that can be checked with
		 * nFaces == 1,
		 */
		fAll int

		/* Intermediate points required */
		anIntermediate [3]int

		/* Number of faces used in hit (1 to 4) */
		nFaces int

		/* Number of pips used to hit */
		nPips int
	}

	/* All ways to hit */
	var aIntermediate = [39]_Inter{
		{1, [3]int{0, 0, 0}, 1, 1},    /*  0: 1x hits 1 */
		{1, [3]int{0, 0, 0}, 1, 2},    /*  1: 2x hits 2 */
		{1, [3]int{1, 0, 0}, 2, 2},    /*  2: 11 hits 2 */
		{1, [3]int{0, 0, 0}, 1, 3},    /*  3: 3x hits 3 */
		{0, [3]int{1, 2, 0}, 2, 3},    /*  4: 21 hits 3 */
		{1, [3]int{1, 2, 0}, 3, 3},    /*  5: 11 hits 3 */
		{1, [3]int{0, 0, 0}, 1, 4},    /*  6: 4x hits 4 */
		{0, [3]int{1, 3, 0}, 2, 4},    /*  7: 31 hits 4 */
		{1, [3]int{2, 0, 0}, 2, 4},    /*  8: 22 hits 4 */
		{1, [3]int{1, 2, 3}, 4, 4},    /*  9: 11 hits 4 */
		{1, [3]int{0, 0, 0}, 1, 5},    /* 10: 5x hits 5 */
		{0, [3]int{1, 4, 0}, 2, 5},    /* 11: 41 hits 5 */
		{0, [3]int{2, 3, 0}, 2, 5},    /* 12: 32 hits 5 */
		{1, [3]int{0, 0, 0}, 1, 6},    /* 13: 6x hits 6 */
		{0, [3]int{1, 5, 0}, 2, 6},    /* 14: 51 hits 6 */
		{0, [3]int{2, 4, 0}, 2, 6},    /* 15: 42 hits 6 */
		{1, [3]int{3, 0, 0}, 2, 6},    /* 16: 33 hits 6 */
		{1, [3]int{2, 4, 0}, 3, 6},    /* 17: 22 hits 6 */
		{0, [3]int{1, 6, 0}, 2, 7},    /* 18: 61 hits 7 */
		{0, [3]int{2, 5, 0}, 2, 7},    /* 19: 52 hits 7 */
		{0, [3]int{3, 4, 0}, 2, 7},    /* 20: 43 hits 7 */
		{0, [3]int{2, 6, 0}, 2, 8},    /* 21: 62 hits 8 */
		{0, [3]int{3, 5, 0}, 2, 8},    /* 22: 53 hits 8 */
		{1, [3]int{4, 0, 0}, 2, 8},    /* 23: 44 hits 8 */
		{1, [3]int{2, 4, 6}, 4, 8},    /* 24: 22 hits 8 */
		{0, [3]int{3, 6, 0}, 2, 9},    /* 25: 63 hits 9 */
		{0, [3]int{4, 5, 0}, 2, 9},    /* 26: 54 hits 9 */
		{1, [3]int{3, 6, 0}, 3, 9},    /* 27: 33 hits 9 */
		{0, [3]int{4, 6, 0}, 2, 10},   /* 28: 64 hits 10 */
		{1, [3]int{5, 0, 0}, 2, 10},   /* 29: 55 hits 10 */
		{0, [3]int{5, 6, 0}, 2, 11},   /* 30: 65 hits 11 */
		{1, [3]int{6, 0, 0}, 2, 12},   /* 31: 66 hits 12 */
		{1, [3]int{4, 8, 0}, 3, 12},   /* 32: 44 hits 12 */
		{1, [3]int{3, 6, 9}, 4, 12},   /* 33: 33 hits 12 */
		{1, [3]int{5, 10, 0}, 3, 15},  /* 34: 55 hits 15 */
		{1, [3]int{4, 8, 12}, 4, 16},  /* 35: 44 hits 16 */
		{1, [3]int{6, 12, 0}, 3, 18},  /* 36: 66 hits 18 */
		{1, [3]int{5, 10, 15}, 4, 20}, /* 37: 55 hits 20 */
		{1, [3]int{6, 12, 18}, 4, 24}, /* 38: 66 hits 24 */
	}

	/* aaRoll[n] - All ways to hit with the n'th roll
	 * Each entry is an index into aIntermediate above.
	 */

	var aaRoll = [21][4]int{
		{0, 2, 5, 9},     /* 11 */
		{1, 8, 17, 24},   /* 22 */
		{3, 16, 27, 33},  /* 33 */
		{6, 23, 32, 35},  /* 44 */
		{10, 29, 34, 37}, /* 55 */
		{13, 31, 36, 38}, /* 66 */
		{0, 1, 4, -1},    /* 21 */
		{0, 3, 7, -1},    /* 31 */
		{1, 3, 12, -1},   /* 32 */
		{0, 6, 11, -1},   /* 41 */
		{1, 6, 15, -1},   /* 42 */
		{3, 6, 20, -1},   /* 43 */
		{0, 10, 14, -1},  /* 51 */
		{1, 10, 19, -1},  /* 52 */
		{3, 10, 22, -1},  /* 53 */
		{6, 10, 26, -1},  /* 54 */
		{0, 13, 18, -1},  /* 61 */
		{1, 13, 21, -1},  /* 62 */
		{3, 13, 25, -1},  /* 63 */
		{6, 13, 28, -1},  /* 64 */
		{10, 13, 30, -1}, /* 65 */
	}

	/* One roll stat */

	var aRoll [21]struct {
		/* number of chequers this roll hits */
		nChequers int

		/* count of pips this roll hits */
		nPips int
	}

	{
		var np int

		for nOppBack = 24; nOppBack >= 0; nOppBack-- {
			if anBoardOpp[nOppBack] > 0 {
				break
			}
		}

		nOppBack = 23 - nOppBack

		for i := nOppBack + 1; i < 25; i++ {
			if anBoard[i] > 0 {
				np += (i + 1 - nOppBack) * anBoard[i]
			}
		}

		afInput[_I_BREAK_CONTACT] = float32(np) / (15 + 152.0)
	}
	{
		var p int

		for i := 0; i < nOppBack; i++ {
			if anBoard[i] > 0 {
				p += (i + 1) * anBoard[i]
			}
		}

		afInput[_I_FREEPIP] = float32(p) / 100.0
	}

	{
		var t int
		var no int
		var m int
		var i int

		if nOppBack >= 11 {
			m = nOppBack
		} else {
			m = 11
		}

		t += 24 * anBoard[24]
		no += anBoard[24]

		for i = 23; i > m; i-- {
			if anBoard[i] > 0 && anBoard[i] != 2 {
				var ns int
				if anBoard[i] > 2 {
					ns = (anBoard[i] - 2)
				} else {
					ns = 1
				}
				no += ns
				t += i * ns
			}
		}

		for ; i >= 6; i-- {
			if anBoard[i] > 0 {
				var nc int = anBoard[i]
				no += nc
				t += i * nc
			}
		}

		for i = 5; i >= 0; i-- {
			if anBoard[i] > 2 {
				t += i * (anBoard[i] - 2)
				no += (anBoard[i] - 2)
			} else if anBoard[i] < 2 {
				var nm int = (2 - anBoard[i])

				if no >= nm {
					t -= i * nm
					no -= nm
				}
			}
		}

		afInput[_I_TIMING] = float32(t) / 100.0
	}

	/* Back chequer */

	{
		var nBack int
		var i, j, n int

		for nBack = 24; nBack >= 0; nBack-- {
			if anBoard[nBack] > 0 {
				break
			}
		}

		afInput[_I_BACK_CHEQUER] = float32(nBack) / 24.0

		/* Back anchor */
		if nBack == 24 {
			i = 23
		} else {
			i = nBack
		}
		for ; i >= 0; i-- {
			if anBoard[i] >= 2 {
				break
			}
		}

		afInput[_I_BACK_ANCHOR] = float32(i) / 24.0

		/* Forward anchor */

		n = 0
		for j = 18; j <= i; j++ {
			if anBoard[j] >= 2 {
				n = 24 - j
				break
			}
		}

		if n == 0 {
			for j = 17; j >= 12; j-- {
				if anBoard[j] >= 2 {
					n = 24 - j
					break
				}
			}
		}

		if n == 0 {
			afInput[_I_FORWARD_ANCHOR] = 2.0
		} else {
			afInput[_I_FORWARD_ANCHOR] = float32(n) / 6.0
		}
	}

	/* Piploss */

	nBoard = 0
	for i := 0; i < 6; i++ {
		if anBoard[i] > 0 {
			nBoard++
		}
	}

	aHit = [39]int{0}

	/* for every point we'd consider hitting a blot on, */
	var start int
	if nBoard > 2 {
		start = 23
	} else {
		start = 21
	}
	for i := start; i >= 0; i-- {
		/* if there's a blot there, then */

		if anBoardOpp[i] == 1 {
			/* for every point beyond */

			for j := 24 - i; j < 25; j++ {
				/* if we have a hitter and are willing to hit */

				if anBoard[j] > 0 && !(j < 6 && anBoard[j] == 2) {
					/* for every roll that can hit from that point */

					for n := 0; n < 5; n++ {
						if aanCombination[j-24+i][n] == -1 {
							break
						}

						/* find the intermediate points required to play */

						pi := aIntermediate[aanCombination[j-24+i][n]]

						if pi.fAll > 0 {
							/* if nFaces is 1, there are no intermediate points */

							if pi.nFaces > 1 {
								/* all the intermediate points are required */

								for k := 0; k < 3 && pi.anIntermediate[k] > 0; k++ {
									if anBoardOpp[i-pi.anIntermediate[k]] > 1 {
										/* point is blocked; look for other hits */
										goto cannot_hit
									}
								}
							}
						} else {
							/* either of two points are required */

							if anBoardOpp[i-pi.anIntermediate[0]] > 1 && anBoardOpp[i-pi.anIntermediate[1]] > 1 {
								/* both are blocked; look for other hits */
								goto cannot_hit
							}
						}

						/* enter this shot as available */

						aHit[aanCombination[j-24+i][n]] |= 1 << j
					cannot_hit:
					}
				}
			}
		}
	}

	if anBoard[24] == 0 {
		/* we're not on the bar; for each roll, */

		for i := 0; i < 21; i++ {
			var n int = -1 /* (hitter used) */

			/* for each way that roll hits, */
			for j := 0; j < 4; j++ {
				var r int = aaRoll[i][j]

				if r < 0 {
					break
				}

				if aHit[r] == 0 {
					continue
				}

				pi := aIntermediate[r]

				if pi.nFaces == 1 {
					/* direct shot */
					k := msb32(aHit[r])
					/* select the most advanced blot; if we still have
					 * a chequer that can hit there */

					if n != k || anBoard[k] > 1 {
						aRoll[i].nChequers++
					}

					n = k

					if k-pi.nPips+1 > aRoll[i].nPips {
						aRoll[i].nPips = k - pi.nPips + 1
					}

					/* if rolling doubles, check for multiple
					 * direct shots */

					if aaRoll[i][3] >= 0 && aHit[r] & ^(1<<k) > 0 {
						aRoll[i].nChequers++
					}
				} else {
					// /* indirect shot */
					if aRoll[i].nChequers == 0 {
						aRoll[i].nChequers = 1
					}

					/* find the most advanced hitter */

					k := msb32(aHit[r])

					if k-pi.nPips+1 > aRoll[i].nPips {
						aRoll[i].nPips = k - pi.nPips + 1
					}

					/* check for blots hit on intermediate points */

					for l := 0; l < 3 && pi.anIntermediate[l] > 0; l++ {
						slot := 23 - k + pi.anIntermediate[l]
						if slot > 24 {
							panic(fmt.Sprintf("slot > 24: %v", slot))
						}
						if anBoardOpp[slot] == 1 {

							aRoll[i].nChequers++
							break
						}
					}
				}
			}
		}
	} else if anBoard[24] == 1 {
		/* we have one on the bar; for each roll, */

		for i := 0; i < 21; i++ {
			var n int = 0 /* (free to use either die to enter) */

			for j := 0; j < 4; j++ {
				var r int = aaRoll[i][j]

				if r < 0 {
					break
				}

				if aHit[r] == 0 {
					continue
				}

				pi := aIntermediate[r]

				if pi.nFaces == 1 {
					/* direct shot */

					for k := msb32(aHit[r]); k > 0; k-- {
						if aHit[r]&(1<<k) > 0 {
							/* if we need this die to enter, we can't hit elsewhere */

							if n > 0 && k != 24 {
								break
							}

							/* if this isn't a shot from the bar, the
							 * other die must be used to enter */

							if k != 24 {
								var npip int = aIntermediate[aaRoll[i][1-j]].nPips

								if anBoardOpp[npip-1] > 1 {
									break
								}

								n = 1
							}

							aRoll[i].nChequers++

							if k-pi.nPips+1 > aRoll[i].nPips {
								aRoll[i].nPips = k - pi.nPips + 1
							}
						}
					}
				} else {
					/* indirect shot -- consider from the bar only */
					if (aHit[r] & (1 << 24)) == 0 {
						continue
					}

					if aRoll[i].nChequers == 0 {
						aRoll[i].nChequers = 1
					}

					if 25-pi.nPips > aRoll[i].nPips {
						aRoll[i].nPips = 25 - pi.nPips
					}

					/* check for blots hit on intermediate points */
					for k := 0; k < 3 && pi.anIntermediate[k] > 0; k++ {
						if anBoardOpp[pi.anIntermediate[k]+1] == 1 {

							aRoll[i].nChequers++
							break
						}
					}
				}
			}
		}
	} else {
		/* we have more than one on the bar --
		 * count only direct shots from point 24 */

		for i := 0; i < 21; i++ {
			/* for the first two ways that hit from the bar */

			for j := 0; j < 2; j++ {
				var r int = aaRoll[i][j]

				if (aHit[r] & (1 << 24)) == 0 {
					continue
				}

				pi := aIntermediate[r]

				/* only consider direct shots */

				if pi.nFaces != 1 {
					continue
				}

				aRoll[i].nChequers++

				if 25-pi.nPips > aRoll[i].nPips {
					aRoll[i].nPips = 25 - pi.nPips
				}
			}
		}
	}

	{
		var np int
		var n1 int
		var n2 int
		var i int

		for i = 0; i < 6; i++ {
			var nc int = aRoll[i].nChequers

			np += aRoll[i].nPips

			if nc > 0 {
				n1 += 1

				if nc > 1 {
					n2 += 1
				}
			}
		}

		for ; i < 21; i++ {
			var nc int = aRoll[i].nChequers

			np += aRoll[i].nPips * 2

			if nc > 0 {
				n1 += 2

				if nc > 1 {
					n2 += 2
				}
			}
		}

		afInput[_I_PIPLOSS] = float32(np) / (12.0 * 36.0)

		afInput[_I_P1] = float32(n1) / 36.0
		afInput[_I_P2] = float32(n2) / 36.0
	}

	afInput[_I_BACKESCAPES] = float32(escapes(anBoard, 23-nOppBack)) / 36.0

	afInput[_I_BACKRESCAPES] = float32(escapes1(anBoard, 23-nOppBack)) / 36.0

	var n, i, j, k int

	for n, i = 36, 15; i < 24-nOppBack; i++ {
		if j = escapes(anBoard, i); j < n {
			n = j
		}
	}

	afInput[_I_ACONTAIN] = float32((36 - n)) / 36.0
	afInput[_I_ACONTAIN2] = afInput[_I_ACONTAIN] * afInput[_I_ACONTAIN]

	if nOppBack < 0 {
		/* restart loop, point 24 should not be included */
		i = 15
		n = 36
	}

	for ; i < 24; i++ {
		if j = escapes(anBoard, i); j < n {
			n = j
		}
	}

	afInput[_I_CONTAIN] = float32(36-n) / 36.0
	afInput[_I_CONTAIN2] = afInput[_I_CONTAIN] * afInput[_I_CONTAIN]

	for n, i = 0, 6; i < 25; i++ {
		if anBoard[i] > 0 {
			n += (i - 5) * anBoard[i] * escapes(anBoardOpp, i)
		}
	}

	afInput[_I_MOBILITY] = float32(n) / 3600.0

	j = 0
	n = 0
	for i = 0; i < 25; i++ {
		var ni int = anBoard[i]

		if ni > 0 {
			j += ni
			n += i * ni
		}
	}

	n = (n + j - 1) / j

	j = 0
	for k, i = 0, n+1; i < 25; i++ {
		var ni int = anBoard[i]

		if ni > 0 {
			j += ni
			k += ni * (i - n) * (i - n)
		}
	}

	if j > 0 {
		k = (k + j - 1) / j
	}

	afInput[_I_MOMENT2] = float32(k) / 400.0

	if anBoard[24] > 0 {
		var loss int
		var two bool = anBoard[24] > 1

		for i = 0; i < 6; i++ {
			if anBoardOpp[i] > 1 {
				/* any double loses */

				loss += 4 * (i + 1)

				for j = i + 1; j < 6; j++ {
					if anBoardOpp[j] > 1 {
						loss += 2 * (i + j + 2)
					} else {
						if two {
							loss += 2 * (i + 1)
						}
					}
				}
			} else {
				if two {
					for j = i + 1; j < 6; j++ {
						if anBoardOpp[j] > 1 {
							loss += 2 * (j + 1)
						}
					}
				}
			}
		}

		afInput[_I_ENTER] = float32(loss) / (36.0 * (49.0 / 6.0))
	} else {
		afInput[_I_ENTER] = 0.0
	}

	n = 0
	for i = 0; i < 6; i++ {
		n += btoi(anBoardOpp[i] > 1)
	}

	afInput[_I_ENTER2] = float32(36-(n-6)*(n-6)) / 36.0

	{
		var pa int = -1
		var w int = 0
		var tot int = 0
		var np int

		for np = 23; np > 0; np-- {
			if anBoard[np] >= 2 {
				if pa == -1 {
					pa = np
					continue
				}

				{
					var d int = pa - np

					var ac = [23]int{
						11, 11, 11, 11, 11, 11, 11,
						6, 5, 4, 3, 2,
						0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					}

					w += ac[d] * anBoard[pa]
					tot += anBoard[pa]
				}
			}
		}

		if tot > 0 {
			afInput[_I_BACKBONE] = 1.0 - (float32(w) / (float32(tot) * 11.0))
		} else {
			afInput[_I_BACKBONE] = 0.0
		}
	}

	{
		var nAc int = 0

		for i = 18; i < 24; i++ {
			if anBoard[i] > 1 {
				nAc++
			}
		}

		afInput[_I_BACKG] = 0.0
		afInput[_I_BACKG1] = 0.0

		if nAc >= 1 {
			var tot int = 0
			for i = 18; i < 25; i++ {
				tot += anBoard[i]
			}

			if nAc > 1 {
				/* g_assert( tot >= 4 ); */

				afInput[_I_BACKG] = float32(tot-3) / 4.0
			} else if nAc == 1 {
				afInput[_I_BACKG1] = float32(tot) / 8.0
			}
		}
	}
}

func computeTable() {
	computeTable0()
	computeTable1()
}

func computeTable0() {
	var n0, n1 int

	anEscapes = [0x1000]int{}

	for i := 0; i < 0x1000; i++ {
		var c int = 0

		for n0 = 0; n0 <= 5; n0++ {
			for n1 = 0; n1 <= n0; n1++ {
				if (i&(1<<(n0+n1+1))) == 0 && !((i&(1<<n0)) > 0 && (i&(1<<n1)) > 0) {
					if n0 == n1 {
						c += 1
					} else {
						c += 2
					}
				}
			}
		}
		anEscapes[i] = c
	}
}

func computeTable1() {
	var n0, n1 int

	anEscapes1 = [0x1000]int{}

	anEscapes1[0] = 0

	for i := 1; i < 0x1000; i++ {
		var c, low int

		for (i & (1 << low)) == 0 {
			low++
		}

		for n0 = 0; n0 <= 5; n0++ {
			for n1 = 0; n1 <= n0; n1++ {

				if (n0+n1+1 > low) && !(i&(1<<(n0+n1+1)) > 0) && !((i&(1<<n0)) > 0 && (i&(1<<n1)) > 0) {
					if n0 == n1 {
						c += 1
					} else {
						c += 2
					}
				}
			}
		}

		anEscapes1[i] = c
	}
}

func escapes(anBoard [25]int, n int) int {
	var af, m int

	if n < 12 {
		m = n
	} else {
		m = 12
	}

	for i := 0; i < m; i++ {
		af |= (anPoint[anBoard[24+i-n]] << i)
	}

	return anEscapes[af]
}

func escapes1(anBoard [25]int, n int) int {
	var af, m int

	if n < 12 {
		m = n
	} else {
		m = 12
	}

	for i := 0; i < m; i++ {
		af |= (anPoint[anBoard[24+i-n]] << i)
	}
	return anEscapes1[af]
}

func evalEfficiency(anBoard _TanBoard, pc _PositionClass) float32 {
	/* Since it's somewhat costly to call CalcInputs, the
	 * inputs should preferably be cached to save time. */

	switch pc {
	case _CLASS_OVER:
		return 0.0 /* dead cube */

	case _CLASS_HYPERGAMMON1, _CLASS_HYPERGAMMON2, _CLASS_HYPERGAMMON3:
		/* FIXME */
		return 0.60

	case _CLASS_BEAROFF1, _CLASS_BEAROFF_OS:
		/* FIXME: calculate based on #rolls to get off.
		 * For example, 15 rolls probably have cube eff. of
		 * 0.7, and 1.25 rolls have cube eff. of 1.0.
		 *
		 * It's not so important to have cube eff. correct here as an
		 * n-ply evaluation will take care of last-roll and 2nd-last-roll
		 * situations. */
		return rOSCubeX

	case _CLASS_RACE:
		{
			var anPips [2]int

			var rEff float32

			pipCount(anBoard, &anPips)

			rEff = float32(anPips[1])*rRaceFactorX + rRaceCoefficientX
			if rEff > rRaceMax {
				return rRaceMax
			} else {
				if rEff < rRaceMin {
					return rRaceMin
				} else {
					return rEff
				}
			}
		}

	case _CLASS_CONTACT:
		/* FIXME: should CLASS_CRASHED be handled differently? */

		/* FIXME: use Oystein's values published in rec.games.backgammon,
		 * or work some other semiempirical values */

		/* FIXME: very important: use opponents inputs as well */
		return rContactX

	case _CLASS_CRASHED:
		return rCrashedX

	case _CLASS_BEAROFF2, _CLASS_BEAROFF_TS:
		return rTSCubeX /* for match play only */

	default:
		panic(fmt.Sprintf("invalid position class: %v", pc))

	}
}

func pipCount(anBoard _TanBoard, anPips *[2]int) {
	anPips[0] = 0
	anPips[1] = 0

	for i := 0; i < 25; i++ {
		anPips[0] += anBoard[0][i] * (i + 1)
		anPips[1] += anBoard[1][i] * (i + 1)
	}
}

func _CFMONEY(arEquity [4]float32, pci *_CubeInfo) float32 {
	if pci.fCubeOwner == -1 {
		return arEquity[2]
	}
	if pci.fCubeOwner == pci.fMove {
		return arEquity[1]
	}
	return arEquity[3]
}

func _CFHYPER(arEquity [4]float32, pci *_CubeInfo) float32 {
	if pci.fCubeOwner == -1 {
		if pci.fJacoby {
			return arEquity[2]
		} else {
			return arEquity[1]
		}
	}
	if pci.fCubeOwner == pci.fMove {
		return arEquity[0]
	}
	return arEquity[3]
}

func _Cl2CfMoney(arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, rCubeX float32) float32 {
	const epsilon float32 = 0.0000001
	const omepsilon float32 = 0.9999999

	var rW, rL float32
	var rEqDead, rEqLive float32

	/* money game */

	/* Transform cubeless 0-ply equity to cubeful 0-ply equity using
	 * Rick Janowski's formulas [insert ref here]. */

	/* First calculate average win and loss W and L: */

	if arOutput[_OUTPUT_WIN] > epsilon {
		rW = 1.0 + (arOutput[_OUTPUT_WINGAMMON]+arOutput[_OUTPUT_WINBACKGAMMON])/arOutput[_OUTPUT_WIN]
	} else {
		/* basically a dead cube */
		return utility(arOutput, pci)
	}

	if arOutput[_OUTPUT_WIN] < omepsilon {
		rL = 1.0 + (arOutput[_OUTPUT_LOSEGAMMON]+arOutput[_OUTPUT_LOSEBACKGAMMON])/(1.0-arOutput[_OUTPUT_WIN])
	} else {
		/* basically a dead cube */
		return utility(arOutput, pci)
	}

	rEqDead = utility(arOutput, pci)
	rEqLive = moneyLive(rW, rL, arOutput[_OUTPUT_WIN], pci)

	return rEqDead*(1.0-rCubeX) + rEqLive*rCubeX

}

func moneyLive(rW float32, rL float32, p float32, pci *_CubeInfo) float32 {
	if pci.fCubeOwner == -1 {
		/* centered cube */
		var rTP float32 = (rL - 0.5) / (rW + rL + 0.5)
		var rCP float32 = (rL + 1.0) / (rW + rL + 0.5)

		if p < rTP {
			/* linear interpolation between
			 * (0,-rL) and ( rTP,-1) */
			if pci.fJacoby {
				return -1.0
			} else {
				return (-rL + (-1.0+rL)*p/rTP)
			}
		} else if p < rCP {
			/* linear interpolation between
			 * (rTP,-1) and (rCP,+1) */
			return -1.0 + 2.0*(p-rTP)/(rCP-rTP)
		} else {
			/* linear interpolation between
			 * (rCP,+1) and (1,+rW) */
			if pci.fJacoby {
				return 1.0
			} else {
				return (+1.0 + (rW-1.0)*(p-rCP)/(1.0-rCP))
			}
		}
	} else if pci.fCubeOwner == pci.fMove {
		/* owned cube */

		/* cash point */
		var rCP float32 = (rL + 1.0) / (rW + rL + 0.5)

		if p < rCP {
			/* linear interpolation between
			 * (0,-rL) and (rCP,+1) */
			return -rL + (1.0+rL)*p/rCP
		} else {
			/* linear interpolation between
			 * (rCP,+1) and (1,+rW) */
			return +1.0 + (rW-1.0)*(p-rCP)/(1.0-rCP)
		}
	} else {
		/* unavailable cube */

		/* take point */
		var rTP float32 = (rL - 0.5) / (rW + rL + 0.5)

		if p < rTP {
			/* linear interpolation between
			 * (0,-rL) and ( rTP,-1) */
			return -rL + (-1.0+rL)*p/rTP
		} else {
			/* linear interpolation between
			 * (rTP,-1) and (1,rW) */
			return -1.0 + (rW+1.0)*(p-rTP)/(1.0-rTP)
		}
	}
}

func _Cl2CfMatch(arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, rCubeX float32) float32 {
	/* Check if this requires a cubeful evaluation */

	if !fDoCubeful(pci) {
		/* cubeless eval */
		return eq2mwc(utility(arOutput, pci), pci)

	} else {
		/* cubeful eval */
		if pci.fCubeOwner == -1 {
			return _Cl2CfMatchCentered(arOutput, pci, rCubeX)
		} else if pci.fCubeOwner == pci.fMove {
			return _Cl2CfMatchOwned(arOutput, pci, rCubeX)
		} else {
			return _Cl2CfMatchUnavailable(arOutput, pci, rCubeX)
		}
	}

}

func fDoCubeful(pci *_CubeInfo) bool {
	if pci.anScore[0]+pci.nCube >= pci.nMatchTo && pci.anScore[1]+pci.nCube >= pci.nMatchTo {
		/* cube is dead */
		return false
	}

	if pci.anScore[0] == pci.nMatchTo-2 && pci.anScore[1] == pci.nMatchTo-2 {
		/* score is -2,-2 */
		return false
	}

	if pci.fCrawford {
		/* cube is dead in Crawford game */
		return false
	}

	return true
}

func mwc2eq(rMwc float32, pci *_CubeInfo) float32 {

	/* mwc if I win/lose */

	var rMwcWin, rMwcLose float32

	rMwcWin = getME(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.fMove, pci.nCube, pci.fMove, pci.fCrawford, &aafMET, &aafMETPostCrawford)

	rMwcLose = getME(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.fMove, pci.nCube, 1-pci.fMove, pci.fCrawford, &aafMET, &aafMETPostCrawford)

	/*
	 * make linear inter- or extrapolation:
	 * equity       mwc
	 *  -1          rMwcLose
	 *  +1          rMwcWin
	 *
	 * Interpolation formula:
	 *
	 *       2 * rMwc - ( rMwcWin + rMwcLose )
	 * rEq = ---------------------------------
	 *            rMwcWin - rMwcLose
	 *
	 * FIXME: numerical problems?
	 * If you are trailing 30-away, 1-away the difference between
	 * 29-away, 1-away and 30-away, 0-away is not very large, and it may
	 * give numerical problems.
	 *
	 */

	return (2.0*rMwc - (rMwcWin + rMwcLose)) / (rMwcWin - rMwcLose)

}

func eq2mwc(rEq float32, pci *_CubeInfo) float32 {
	/* mwc if I win/lose */

	var rMwcWin, rMwcLose float32

	rMwcWin = getME(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.fMove, pci.nCube, pci.fMove, pci.fCrawford, &aafMET, &aafMETPostCrawford)

	rMwcLose = getME(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.fMove, pci.nCube, 1-pci.fMove, pci.fCrawford, &aafMET, &aafMETPostCrawford)

	/*
	 * Linear inter- or extrapolation.
	 * Solve the formula in the routine above (mwc2eq):
	 *
	 *        rEq * ( rMwcWin - rMwcLose ) + ( rMwcWin + rMwcLose )
	 * rMwc = -----------------------------------------------------
	 *                                   2
	 */

	return 0.5 * (rEq*(rMwcWin-rMwcLose) + (rMwcWin + rMwcLose))

}

func _Cl2CfMatchCentered(arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, rCubeX float32) float32 {

	/* normalized score */

	var rG0, rBG0, rG1, rBG1 float32
	var arCP [2]float32

	var rMWCDead, rMWCLive float32
	var rMWCOppCash, rMWCCash, rOppTG, rTG float32
	var aarMETResult [2][_DTLBP1 + 1]float32

	/* Centered cube */

	/* Calculate normal, gammon, and backgammon ratios */

	if arOutput[_OUTPUT_WIN] > 0.0 {
		rG0 = (arOutput[_OUTPUT_WINGAMMON] - arOutput[_OUTPUT_WINBACKGAMMON]) / arOutput[_OUTPUT_WIN]
		rBG0 = arOutput[_OUTPUT_WINBACKGAMMON] / arOutput[_OUTPUT_WIN]
	} else {
		rG0 = 0.0
		rBG0 = 0.0
	}

	if arOutput[_OUTPUT_WIN] < 1.0 {
		rG1 = (arOutput[_OUTPUT_LOSEGAMMON] - arOutput[_OUTPUT_LOSEBACKGAMMON]) / (1.0 - arOutput[_OUTPUT_WIN])
		rBG1 = arOutput[_OUTPUT_LOSEBACKGAMMON] / (1.0 - arOutput[_OUTPUT_WIN])
	} else {
		rG1 = 0.0
		rBG1 = 0.0
	}

	/* MWC(dead cube) = cubeless equity */

	rMWCDead = eq2mwc(utility(arOutput, pci), pci)

	/* Get live cube cash points */

	getPoints(arOutput, pci, arCP)

	getMEMultiple(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.nCube, -1, -1, pci.fCrawford, &aafMET, &aafMETPostCrawford, aarMETResult[0][:], aarMETResult[1][:])

	rMWCCash = aarMETResult[pci.fMove][_NDW]

	rMWCOppCash = aarMETResult[pci.fMove][_NDL]

	rOppTG = 1.0 - arCP[1-pci.fMove]
	rTG = arCP[pci.fMove]

	if arOutput[_OUTPUT_WIN] <= rOppTG {

		/* Opp too good to double */

		var rMWCLose float32 = (1.0-rG1-rBG1)*aarMETResult[pci.fMove][_NDL] + rG1*aarMETResult[pci.fMove][_NDLG] + rBG1*aarMETResult[pci.fMove][_NDLB]

		if rOppTG > 0.0 {
			/* avoid division by zero */
			rMWCLive = rMWCLose + (rMWCOppCash-rMWCLose)*arOutput[_OUTPUT_WIN]/rOppTG
		} else {
			rMWCLive = rMWCLose
		}

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	} else if arOutput[_OUTPUT_WIN] < rTG {

		/* In doubling window */

		rMWCLive = rMWCOppCash + (rMWCCash-rMWCOppCash)*(arOutput[_OUTPUT_WIN]-rOppTG)/(rTG-rOppTG)
		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	} else {

		/* I'm too good to double */

		/* MWC(live cube) linear interpolation between the
		 * points:
		 *
		 * p = TG, MWC = I win 1 point
		 * p = 1, MWC = I win (normal, gammon, or backgammon)
		 *
		 */

		var rMWCWin float32 = (1.0-rG0-rBG0)*aarMETResult[pci.fMove][_NDW] + rG0*aarMETResult[pci.fMove][_NDWG] + rBG0*aarMETResult[pci.fMove][_NDWB]

		if rTG < 1.0 {
			rMWCLive = rMWCCash + (rMWCWin-rMWCCash)*(arOutput[_OUTPUT_WIN]-rTG)/(1.0-rTG)
		} else {
			rMWCLive = rMWCWin
		}

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	}
}

func _Cl2CfMatchOwned(arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, rCubeX float32) float32 {
	// /* normalized score */

	var rG0, rBG0, rG1, rBG1 float32
	var arCP [2]float32

	var rMWCDead, rMWCLive float32
	var rMWCCash, rTG float32
	var aarMETResult [2][_DTLBP1 + 1]float32

	/* I own cube */

	/* Calculate normal, gammon, and backgammon ratios */

	if arOutput[_OUTPUT_WIN] > 0.0 {
		rG0 = (arOutput[_OUTPUT_WINGAMMON] - arOutput[_OUTPUT_WINBACKGAMMON]) / arOutput[_OUTPUT_WIN]
		rBG0 = arOutput[_OUTPUT_WINBACKGAMMON] / arOutput[_OUTPUT_WIN]
	} else {
		rG0 = 0.0
		rBG0 = 0.0
	}

	if arOutput[_OUTPUT_WIN] < 1.0 {
		rG1 = (arOutput[_OUTPUT_LOSEGAMMON] - arOutput[_OUTPUT_LOSEBACKGAMMON]) / (1.0 - arOutput[_OUTPUT_WIN])
		rBG1 = arOutput[_OUTPUT_LOSEBACKGAMMON] / (1.0 - arOutput[_OUTPUT_WIN])
	} else {
		rG1 = 0.0
		rBG1 = 0.0
	}

	/* MWC(dead cube) = cubeless equity */

	rMWCDead = eq2mwc(utility(arOutput, pci), pci)

	/* Get live cube cash points */

	getPoints(arOutput, pci, arCP)

	getMEMultiple(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.nCube, -1, -1, pci.fCrawford, &aafMET, &aafMETPostCrawford, aarMETResult[0][:], aarMETResult[1][:])

	rMWCCash = aarMETResult[pci.fMove][_NDW]

	rTG = arCP[pci.fMove]

	if arOutput[_OUTPUT_WIN] <= rTG {

		/* MWC(live cube) linear interpolation between the
		 * points:
		 *
		 * p = 0, MWC = I lose (normal, gammon, or backgammon)
		 * p = TG, MWC = I win 1 point
		 *
		 */

		var rMWCLose float32 = (1.0-rG1-rBG1)*aarMETResult[pci.fMove][_NDL] + rG1*aarMETResult[pci.fMove][_NDLG] + rBG1*aarMETResult[pci.fMove][_NDLB]

		if rTG > 0.0 {
			rMWCLive = rMWCLose + (rMWCCash-rMWCLose)*arOutput[_OUTPUT_WIN]/rTG
		} else {
			rMWCLive = rMWCLose
		}

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	} else {

		/* we are too good to double */

		/* MWC(live cube) linear interpolation between the
		 * points:
		 *
		 * p = TG, MWC = I win 1 point
		 * p = 1, MWC = I win (normal, gammon, or backgammon)
		 *
		 */

		var rMWCWin float32 = (1.0-rG0-rBG0)*aarMETResult[pci.fMove][_NDW] + rG0*aarMETResult[pci.fMove][_NDWG] + rBG0*aarMETResult[pci.fMove][_NDWB]

		if rTG < 1.0 {
			rMWCLive = rMWCCash + (rMWCWin-rMWCCash)*(arOutput[_OUTPUT_WIN]-rTG)/(1.0-rTG)
		} else {
			rMWCLive = rMWCWin
		}

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	}
}

func _Cl2CfMatchUnavailable(arOutput *[_NUM_OUTPUTS]float32, pci *_CubeInfo, rCubeX float32) float32 {
	// /* normalized score */

	var rG0, rBG0, rG1, rBG1 float32
	var arCP [2]float32

	var rMWCDead, rMWCLive float32
	var rMWCOppCash, rOppTG float32
	var aarMETResult [2][_DTLBP1 + 1]float32

	/* I own cube */

	/* Calculate normal, gammon, and backgammon ratios */

	if arOutput[_OUTPUT_WIN] > 0.0 {
		rG0 = (arOutput[_OUTPUT_WINGAMMON] - arOutput[_OUTPUT_WINBACKGAMMON]) / arOutput[_OUTPUT_WIN]
		rBG0 = arOutput[_OUTPUT_WINBACKGAMMON] / arOutput[_OUTPUT_WIN]
	} else {
		rG0 = 0.0
		rBG0 = 0.0
	}

	if arOutput[_OUTPUT_WIN] < 1.0 {
		rG1 = (arOutput[_OUTPUT_LOSEGAMMON] - arOutput[_OUTPUT_LOSEBACKGAMMON]) / (1.0 - arOutput[_OUTPUT_WIN])
		rBG1 = arOutput[_OUTPUT_LOSEBACKGAMMON] / (1.0 - arOutput[_OUTPUT_WIN])
	} else {
		rG1 = 0.0
		rBG1 = 0.0
	}

	/* MWC(dead cube) = cubeless equity */

	rMWCDead = eq2mwc(utility(arOutput, pci), pci)

	/* Get live cube cash points */

	getPoints(arOutput, pci, arCP)

	getMEMultiple(pci.anScore[0], pci.anScore[1], pci.nMatchTo,
		pci.nCube, -1, -1, pci.fCrawford, &aafMET, &aafMETPostCrawford, aarMETResult[0][:], aarMETResult[1][:])

	rMWCOppCash = aarMETResult[pci.fMove][_NDL]

	rOppTG = 1.0 - arCP[1-pci.fMove]

	if arOutput[_OUTPUT_WIN] <= rOppTG {

		/* Opponent is too good to double.
		 *
		 * MWC(live cube) linear interpolation between the
		 * points:
		 *
		 * p = 0, MWC = opp win normal, gammon, backgammon
		 * p = OppTG, MWC = opp cashes
		 *
		 */

		var rMWCLose float32 = (1.0-rG1-rBG1)*aarMETResult[pci.fMove][_NDL] + rG1*aarMETResult[pci.fMove][_NDLG] + rBG1*aarMETResult[pci.fMove][_NDLB]

		if rOppTG > 0.0 {
			/* avoid division by zero */
			rMWCLive = rMWCLose + (rMWCOppCash-rMWCLose)*arOutput[_OUTPUT_WIN]/rOppTG
		} else {
			rMWCLive = rMWCLose
		}

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	} else {

		/* MWC(live cube) linear interpolation between the
		 * points:
		 *
		 * p = OppTG, MWC = opponent cashes
		 * p = 1, MWC = I win (normal, gammon, or backgammon)
		 *
		 */

		var rMWCWin float32 = (1.0-rG0-rBG0)*aarMETResult[pci.fMove][_NDW] + rG0*aarMETResult[pci.fMove][_NDWG] + rBG0*aarMETResult[pci.fMove][_NDWB]

		rMWCLive = rMWCOppCash + (rMWCWin-rMWCOppCash)*(arOutput[_OUTPUT_WIN]-rOppTG)/(1.0-rOppTG)

		/* (1-x) MWC(dead) + x MWC(live) */

		return rMWCDead*(1.0-rCubeX) + rMWCLive*rCubeX

	}
}

func formatEval(pes _EvalSetup) string {
	switch pes.et {
	case _EVAL_NONE:
		return ""
	case _EVAL_EVAL:
		if pes.ec.fCubeful {
			return fmt.Sprintf("Cubeful %d-ply", pes.ec.nPlies)
		} else {
			return fmt.Sprintf("Cubeless %d-ply", pes.ec.nPlies)
		}
	case _EVAL_ROLLOUT:
		return "Rollout"
	default:
		return fmt.Sprintf("Unknown (%d)", pes.et)
	}
}
