package gnubg

import (
	"bgweb-api/gnubg/math32"
	"fmt"
	"os"
	"strconv"
)

type _BearOffType int

const (
	_BEAROFF_INVALID _BearOffType = iota
	_BEAROFF_ONESIDED
	_BEAROFF_TWOSIDED
	_BEAROFF_HYPERGAMMON
)

type _BearOffContext struct {
	bt        _BearOffType /* type of bearoff database */
	nPoints   int          /* number of points covered by database */
	nChequers int          /* number of chequers for one-sided database */
	// /* one sided dbs */
	fCompressed bool /* is database compressed? */
	fGammon     bool /* gammon probs included */
	fND         bool /* normal distibution instead of exact dist? */
	fHeuristic  bool /* heuristic database? */
	// /* two sided dbs */
	fCubeful   bool   /* cubeful equities included */
	szFilename string /* filename */
	// GMappedFile *map;
	p []byte /* pointer to data in memory */
}

const (
	_BO_NONE              int = 0
	_BO_MUST_BE_ONE_SIDED int = 1
	_BO_MUST_BE_TWO_SIDED int = 2
	_BO_HEURISTIC         int = 4
)

const _HEURISTIC_C = 15
const _HEURISTIC_P = 6

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func bearoffInit(szFilename string, bo int) (*_BearOffContext, error) {
	var pbc _BearOffContext

	if bo&_BO_HEURISTIC > 0 {
		pbc.bt = _BEAROFF_ONESIDED
		pbc.nPoints = _HEURISTIC_P
		pbc.nChequers = _HEURISTIC_C
		pbc.fHeuristic = true
		pbc.p = heuristicDatabase()
		return &pbc, nil
	}

	if len(szFilename) == 0 {
		invalidDb(&pbc)
		return nil, fmt.Errorf("no bearoff database filename provided")
	}
	pbc.szFilename = szFilename

	sz, err := os.ReadFile(szFilename)
	if err != nil {
		invalidDb(&pbc)
		return nil, fmt.Errorf("error while reading bearoff database: %v", err)
	}

	if len(sz) < 40 {
		invalidDb(&pbc)
		return nil, fmt.Errorf("invalid bearoff database")
	}

	/* detect bearoff program */

	if string(sz[:5]) != "gnubg" {
		invalidDb(&pbc)
		return nil, fmt.Errorf("unknown bearoff database")
	}

	/* one sided or two sided? */

	if string(sz[6:6+2]) == "TS" {
		pbc.bt = _BEAROFF_TWOSIDED
	} else if string(sz[6:6+2]) == "OS" {
		pbc.bt = _BEAROFF_ONESIDED
	} else if sz[6] == 'H' {
		pbc.bt = _BEAROFF_HYPERGAMMON
	} else {
		invalidDb(&pbc)
		return nil, fmt.Errorf("%v: %v\n (%v: '%v')", szFilename, "incomplete bearoff database", "illegal bearoff type", sz[6:6+2])
	}

	if ((bo&_BO_MUST_BE_ONE_SIDED > 0) && (pbc.bt != _BEAROFF_ONESIDED)) || ((bo&_BO_MUST_BE_TWO_SIDED > 0) && (pbc.bt != _BEAROFF_TWOSIDED)) {
		invalidDb(&pbc)
		return nil, fmt.Errorf("%v: %v\n (%v: '%v')", szFilename, "incorrect bearoff database", "wrong bearoff type", sz[6:6+2])
	}

	if pbc.bt == _BEAROFF_TWOSIDED || pbc.bt == _BEAROFF_ONESIDED {

		/* normal onesided or twosided bearoff database */

		/* number of points */

		pbc.nPoints, _ = strconv.Atoi(string(sz[9 : 9+2]))
		if pbc.nPoints < 1 || pbc.nPoints >= 24 {
			invalidDb(&pbc)
			return nil, fmt.Errorf("%v: %v\n (%v: %v)", szFilename, "incomplete bearoff database", "illegal number of points", pbc.nPoints)
		}

		/* number of chequers */

		pbc.nChequers, _ = strconv.Atoi(string(sz[12 : 12+2]))
		if pbc.nChequers < 1 || pbc.nChequers > 15 {
			invalidDb(&pbc)
			return nil, fmt.Errorf("%v: %v\n (%v: %v)", szFilename, "incomplete bearoff database", "illegal number of chequers", pbc.nChequers)
		}

	} else {

		/* hypergammon database */

		pbc.nPoints = 25
		pbc.nChequers, _ = strconv.Atoi(string(sz[7 : 7+2]))

	}
	switch pbc.bt {
	case _BEAROFF_TWOSIDED:
		/* options for two-sided dbs */
		pbc.fCubeful = atoi(string(sz[15])) == 1
	case _BEAROFF_ONESIDED:
		/* options for one-sided dbs */
		pbc.fGammon = atoi(string(sz[15])) == 1
		pbc.fCompressed = atoi(string(sz[17])) == 1
		pbc.fND = atoi(string(sz[19])) == 1
	case _BEAROFF_HYPERGAMMON, _BEAROFF_INVALID:
		break
	default:
		break
	}

	pbc.p = sz

	return &pbc, nil
}

func bearoffClose(pbc *_BearOffContext) {
	if pbc == nil {
		return
	}
	pbc.p = nil
}

func invalidDb(pbc *_BearOffContext) {
	bearoffClose(pbc)
}

func isBearoff(pbc *_BearOffContext, anBoard _TanBoard) bool {
	var nOppBack, nBack int
	var n, nOpp int

	if pbc == nil {
		return false
	}

	for nOppBack = 24; nOppBack > 0; nOppBack-- {
		if anBoard[0][nOppBack] != 0 {
			break
		}
	}
	for nBack = 24; nBack > 0; nBack-- {
		if anBoard[1][nBack] != 0 {
			break
		}
	}
	if anBoard[0][nOppBack] == 0 || anBoard[1][nBack] == 0 {
		/* the game is over */
		return false
	}

	if (nBack+nOppBack > 22) && !(pbc.bt == _BEAROFF_HYPERGAMMON) {
		/* contact position */
		return false
	}

	for i := 0; i <= nOppBack; i++ {
		nOpp += anBoard[0][i]
	}

	for i := 0; i <= nBack; i++ {
		n += anBoard[1][i]
	}

	if n <= pbc.nChequers && nOpp <= pbc.nChequers && nBack < pbc.nPoints && nOppBack < pbc.nPoints {
		return true
	} else {
		return false
	}
}

func generateBearoff(p []byte, nId int) {
	var anRoll [2]int
	var anBoard [6]int
	var aProb [32]int
	var iBest int

	for i := 0; i < 32; i++ {
		aProb[i] = 0
	}

	for anRoll[0] = 1; anRoll[0] <= 6; anRoll[0]++ {
		for anRoll[1] = 1; anRoll[1] <= anRoll[0]; anRoll[1]++ {
			positionFromBearoff(&anBoard, nId, _HEURISTIC_P, _HEURISTIC_C)
			iBest = heuristicBearoff(&anBoard, anRoll)

			if iBest >= nId {
				panic("iBest >= nId")
			}

			if anRoll[0] == anRoll[1] {
				for i := 0; i < 31; i++ {
					aProb[i+1] += int(p[(iBest<<6)|(i<<1)]) + int(p[(iBest<<6)|(i<<1)|1])<<8
				}
			} else {
				for i := 0; i < 31; i++ {
					aProb[i+1] += int(p[(iBest<<6)|(i<<1)]) + int(p[(iBest<<6)|(i<<1)|1])<<8<<1
				}
			}
		}
	}

	for i := 0; i < 32; i++ {
		var us int = ((aProb[i] + 18) / 36)

		p[(nId<<6)|(i<<1)] = byte(us & 0xFF)
		p[(nId<<6)|(i<<1)|1] = byte(us >> 8)
	}
}

func heuristicDatabase() []byte {
	pm := make([]byte, 40+54264*64)
	p := pm[40:]

	p[0] = 0xff
	p[1] = 0xff
	for i := 2; i < 64; i++ {
		p[i] = 0
	}

	for i := 1; i < 54264; i++ {
		generateBearoff(p, i)
	}

	return pm
}

/* Make a plausible bearoff move (used to create approximate bearoff database). */
func heuristicBearoff(anBoard *[6]int, anRoll [2]int) int {
	var i int    /* current die being played */
	var c int    /* number of dice to play */
	var nMax int /* highest occupied point */
	var anDice [4]int
	var j, iSearch, nTotal int
	var n int /* point to play from */

	if anRoll[0] == anRoll[1] {
		/* doubles */
		anDice[3] = anRoll[0]
		anDice[2] = anDice[3]
		anDice[1] = anDice[2]
		anDice[0] = anDice[1]
		c = 4
	} else {
		/* non-doubles */
		if anRoll[0] <= anRoll[1] {
			panic("anRoll[0] <= anRoll[1]")
		}
		anDice[0] = anRoll[0]
		anDice[1] = anRoll[1]
		c = 2
	}

	for i = 0; i < c; i++ {
		for nMax = 5; nMax > 0; nMax-- {
			if anBoard[nMax] > 0 {
				break
			}
		}

		if anBoard[nMax] == 0 {
			/* finished bearoff */
			break
		}

		for {
			if anBoard[anDice[i]-1] > 0 {
				/* bear off exactly */
				n = anDice[i] - 1
				break
			}

			if anDice[i]-1 > nMax {
				/* bear off highest chequer */
				n = nMax
				break
			}

			nTotal = anDice[i] - 1
			for n, j = -1, i+1; j < c; j++ {
				nTotal += anDice[j]
				if nTotal < 6 && anBoard[nTotal] > 0 {
					/* there's a chequer we can bear off with subsequent dice;
					 * do it */
					n = nTotal
					break
				}
			}
			if n >= 0 {
				break
			}

			for n, iSearch = -1, anDice[i]; iSearch <= nMax; iSearch++ {
				if anBoard[iSearch] >= 2 && /* at least 2 on source point */
					anBoard[iSearch-anDice[i]] == 0 && /* dest empty */
					(n == -1 || anBoard[iSearch] > anBoard[n]) {
					n = iSearch
				}
			}
			if n >= 0 {
				break
			}

			/* find the point with the most on it (or least on dest) */
			for iSearch = anDice[i]; iSearch <= nMax; iSearch++ {
				if n == -1 || anBoard[iSearch] > anBoard[n] ||
					(anBoard[iSearch] == anBoard[n] &&
						anBoard[iSearch-anDice[i]] < anBoard[n-anDice[i]]) {
					n = iSearch
				}
			}

			if n < 0 {
				panic("n < 0")
			}
		}

		if anBoard[n] == 0 {
			panic("anBoard[n] == 0")
		}
		anBoard[n]--

		if n >= anDice[i] {
			anBoard[n-anDice[i]]++
		}
	}

	return positionBearoff(*anBoard, _HEURISTIC_P, _HEURISTIC_C)
}

func bearoffDist(pbc *_BearOffContext, nPosID int, arProb *[32]float32, arGammonProb *[32]float32, ar *[4]float32, ausProb *[32]int, ausGammonProb *[32]int) error {
	if pbc == nil {
		return fmt.Errorf("pbc not supplied")
	}
	if pbc.bt != _BEAROFF_ONESIDED {
		return fmt.Errorf("invalid bearoff type: %v", pbc.bt)
	}
	if pbc.fND {
		// TODO: return readBearoffOneSidedND(pbc, nPosID, arProb, arGammonProb, ar, ausProb, ausGammonProb)
		panic("not implemented")
	} else {
		return readBearoffOneSidedExact(pbc, nPosID, arProb, arGammonProb, ar, ausProb, ausGammonProb)
	}
}

func readBearoffOneSidedExact(pbc *_BearOffContext, nPosID int, arProb *[32]float32, arGammonProb *[32]float32, ar *[4]float32, ausProb *[32]int, ausGammonProb *[32]int) error {
	var aus [64]int
	var pus *[64]int

	/* get distribution */
	if pbc.fCompressed {
		pus = getDistCompressed(&aus, pbc, nPosID)
	} else {
		// pus = getDistUncompressed(&aus, pbc, nPosID)
		panic("not implemented")
	}

	if pus == nil {
		return fmt.Errorf("unable to get distribution")
	}

	assignOneSided(arProb, arGammonProb, ar, ausProb, ausGammonProb, pus[:], pus[32:])

	return nil
}

func readBearoffDatabase(pbc *_BearOffContext, offset int, bytes int) []byte {
	if pbc.p == nil {
		panic("bearoff database not initialised")
	}
	return pbc.p[offset : offset+bytes]
}

func makeInt(a byte, b byte, c byte, d byte) int {
	return int(a) | int(b)<<8 | int(c)<<16 | int(d)<<24
}

func copyBytes(aus *[64]int, ac []byte, nz int, ioff int, nzg int, ioffg int) {
	(*aus) = [64]int{0}
	i := 0
	for j := 0; j < nz; j, i = j+1, i+2 {
		aus[ioff+j] = int(ac[i]) | int(ac[i+1])<<8
	}
	for j := 0; j < nzg; j, i = j+1, i+2 {
		aus[32+ioffg+j] = int(ac[i]) | int(ac[i+1])<<8
	}
}

func getDistCompressed(aus *[64]int, pbc *_BearOffContext, nPosID int) *[64]int {
	var puch []byte
	var iOffset int
	var nBytes int
	var ioff, nz, ioffg, nzg int
	var nPos int = combination(pbc.nPoints+pbc.nChequers, pbc.nPoints)
	var index_entry_size int

	if pbc.fGammon {
		index_entry_size = 8
	} else {
		index_entry_size = 6
	}

	/* find offsets and no. of non-zero elements */
	puch = readBearoffDatabase(pbc, 40+nPosID*index_entry_size, index_entry_size)

	/* find offset */
	iOffset = makeInt(puch[0], puch[1], puch[2], puch[3])

	nz = int(puch[4])
	ioff = int(puch[5])
	if pbc.fGammon {
		nzg = int(puch[6])
		ioffg = int(puch[7])
	}

	/* Sanity checks */
	if (iOffset > 64*nPos && 64*nPos > 0) || nz > 32 || ioff > 32 || nzg > 32 || ioffg > 32 {
		panic(fmt.Sprintf("The bearoff file '%v' is likely to be corrupted.\n"+
			"Offset %v, dist size %v (offset %v), "+
			"gammon dist size %v (offset %v)\n", pbc.szFilename, iOffset, nz, ioff, nzg, ioffg))
	}

	/* read prob + gammon probs */
	iOffset = 40 /* the header */ + nPos*index_entry_size /* the offset data */ + 2*iOffset /* offset to current position */

	/* read values */
	nBytes = 2 * (nz + nzg)

	/* get distribution */
	puch = readBearoffDatabase(pbc, iOffset, nBytes)

	copyBytes(aus, puch, nz, ioff, nzg, ioffg)

	return aus
}

func assignOneSided(arProb *[32]float32, arGammonProb *[32]float32, ar *[4]float32, ausProb *[32]int, ausGammonProb *[32]int, ausProbx []int, ausGammonProbx []int) {
	if ausProb != nil {
		copy((*ausProb)[:], ausProbx)
	}
	if ausGammonProb != nil {
		copy((*ausGammonProb)[:], ausGammonProbx)
	}
	if ar != nil || arProb != nil || arGammonProb != nil {
		var arx [64]float32

		for i := 0; i < 32; i++ {
			arx[i] = float32(ausProbx[i]) / 65535.0
		}
		for i := 0; i < 32; i++ {
			arx[32+i] = float32(ausGammonProbx[i]) / 65535.0
		}
		if arProb != nil {
			copy((*arProb)[:], arx[:])
		}
		if arGammonProb != nil {
			copy((*arGammonProb)[:], arx[32:])
		}
		if ar != nil {
			averageRolls(arx[:], ar[:])
			averageRolls(arx[32:], ar[2:])
		}
	}
}

func averageRolls(arProb []float32, ar []float32) {
	var sx float32
	var sx2 float32

	for i := 1; i < 32; i++ {
		p := float32(i) * arProb[i]
		sx += p
		sx2 += float32(i) * p
	}

	ar[0] = sx
	ar[1] = math32.Sqrtf(sx2 - sx*sx)
}

func bearoffEval(pbc *_BearOffContext, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32) error {
	if pbc == nil {
		return fmt.Errorf("pbc not supplied")
	}

	switch pbc.bt {
	case _BEAROFF_TWOSIDED:
		return bearoffEvalTwoSided(pbc, anBoard, arOutput)
	case _BEAROFF_ONESIDED:
		return bearoffEvalOneSided(pbc, anBoard, arOutput)
	case _BEAROFF_HYPERGAMMON:
		return bearoffEvalHypergammon(pbc, anBoard, arOutput)
	case _BEAROFF_INVALID:
	default:
		return fmt.Errorf("invalid type in BearoffEval: %v", pbc.bt)
	}

	return nil
}

func bearoffEvalTwoSided(pbc *_BearOffContext, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32) error {
	nUs := positionBearoff(anBoard.getHomeBoard(1), pbc.nPoints, pbc.nChequers)
	nThem := positionBearoff(anBoard.getHomeBoard(0), pbc.nPoints, pbc.nChequers)
	n := combination(pbc.nPoints+pbc.nChequers, pbc.nPoints)
	iPos := nUs*n + nThem
	var ar [4]float32

	readTwoSidedBearoff(pbc, iPos, &ar, nil)

	*arOutput = [_NUM_OUTPUTS]float32{0.0}
	arOutput[_OUTPUT_WIN] = ar[0]/2.0 + 0.5

	return nil
}

/* BEAROFF_GNUBG: read two sided bearoff database */
func readTwoSidedBearoff(pbc *_BearOffContext, iPos int, ar *[4]float32, aus *[4]int) {
	var k int = 1
	if pbc.fCubeful {
		k = 4
	}
	var pc []byte = readBearoffDatabase(pbc, 40+2*iPos*k, k*2)

	/* add to cache */
	for i := 0; i < k; i++ {
		var us int = int(pc[2*i]) | (int(pc[2*i+1]) << 8)

		if aus != nil {
			aus[i] = us
		}
		if ar != nil {
			ar[i] = float32(us)/32767.5 - 1.0
		}
	}
}

func bearoffEvalOneSided(pbc *_BearOffContext, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32) error {
	var aarProb [2][32]float32
	var aarGammonProb [2][32]float32
	var r float32
	var anOn [2]int
	var an [2]int
	var ar [2][4]float32

	/* get bearoff probabilities */
	for i := 0; i < 2; i++ {
		an[i] = positionBearoff(anBoard.getHomeBoard(i), pbc.nPoints, pbc.nChequers)
		if err := bearoffDist(pbc, an[i], &aarProb[i], &aarGammonProb[i], &ar[i], nil, nil); err != nil {
			return err
		}
	}

	/* calculate winning chance */

	r = 0.0
	for i := 0; i < 32; i++ {
		for j := i; j < 32; j++ {
			r += aarProb[1][i] * aarProb[0][j]
		}
	}
	arOutput[_OUTPUT_WIN] = r

	/* calculate gammon chances */

	for i := 0; i < 2; i++ {
		anOn[i] = 0
		for j := 0; j < 25; j++ {
			anOn[i] += anBoard[i][j]
		}
	}
	if anOn[0] == 15 || anOn[1] == 15 {
		if pbc.fGammon {
			/* my gammon chance: I'm out in i rolls and my opponent isn't inside
			 * home quadrant in less than i rolls */
			r = 0
			for i := 0; i < 32; i++ {
				for j := i; j < 32; j++ {
					r += aarProb[1][i] * aarGammonProb[0][j]
				}
			}
			arOutput[_OUTPUT_WINGAMMON] = r

			/* opp gammon chance */
			r = 0
			for i := 0; i < 32; i++ {
				for j := i + 1; j < 32; j++ {
					r += aarProb[0][i] * aarGammonProb[1][j]
				}
			}
			arOutput[_OUTPUT_LOSEGAMMON] = r

		} else {
			if err := setGammonProb(anBoard, an[0], an[1], &arOutput[_OUTPUT_LOSEGAMMON], &arOutput[_OUTPUT_WINGAMMON]); err != nil {
				return err
			}
		}
	} else {
		/* no gammons possible */
		arOutput[_OUTPUT_WINGAMMON] = 0.0
		arOutput[_OUTPUT_LOSEGAMMON] = 0.0
	}

	/* no backgammons possible */
	arOutput[_OUTPUT_LOSEBACKGAMMON] = 0.0
	arOutput[_OUTPUT_WINBACKGAMMON] = 0.0

	return nil
}

func setGammonProb(anBoard _TanBoard, bp0 int, bp1 int, g0 *float32, g1 *float32) error {
	var prob [32]int

	var tot0 int
	var tot1 int

	for i := 5; i >= 0; i-- {
		tot0 += anBoard[0][i]
		tot1 += anBoard[1][i]
	}

	if tot0 != 15 || tot1 != 15 {
		panic("assert")
	}

	*g0 = 0.0
	*g1 = 0.0

	if tot0 == 15 {
		gp := getBearoffGammonProbs(anBoard.getHomeBoard(0))
		var make [3]float32

		if err := bearoffDist(pbc1, bp1, nil, nil, nil, &prob, nil); err != nil {
			return err
		}

		make[0] = float32(gp.p0) / 36.0
		make[1] = (make[0] + float32(gp.p1)/(36.0*36.0))
		make[2] = (make[1] + float32(gp.p2)/(36.0*36.0*36.0))

		*g1 = ((float32(prob[1]) / 65535.0) +
			(1-make[0])*(float32(prob[2])/65535.0) +
			(1-make[1])*(float32(prob[3])/65535.0) +
			(1-make[2])*(float32(prob[4])/65535.0))
	}

	if tot1 == 15 {
		gp := getBearoffGammonProbs(anBoard.getHomeBoard(1))
		var make [3]float32

		if err := bearoffDist(pbc1, bp0, nil, nil, nil, &prob, nil); err != nil {
			return err
		}
		make[0] = float32(gp.p0) / 36.0
		make[1] = (make[0] + float32(gp.p1)/(36.0*36.0))
		make[2] = (make[1] + float32(gp.p2)/(36.0*36.0*36.0))

		*g0 = ((float32(prob[1])/65535.0)*(1-make[0]) +
			(float32(prob[2])/65535.0)*(1-make[1]) +
			(float32(prob[3])/65535.0)*(1-make[2]))
	}
	return nil
}

func bearoffEvalHypergammon(pbc *_BearOffContext, anBoard _TanBoard, arOutput *[_NUM_OUTPUTS]float32) error {
	nUs := positionBearoff(anBoard.getHomeBoard(1), pbc.nPoints, pbc.nChequers)
	nThem := positionBearoff(anBoard.getHomeBoard(0), pbc.nPoints, pbc.nChequers)
	n := combination(pbc.nPoints+pbc.nChequers, pbc.nPoints)
	iPos := nUs*n + nThem

	return readHypergammon(pbc, iPos, arOutput, nil)
}

func readHypergammon(pbc *_BearOffContext, iPos int, arOutput *[_NUM_OUTPUTS]float32, arEquity *[4]float32) error {
	x := 28

	var pc []byte = readBearoffDatabase(pbc, 40+x*iPos, x)

	if arOutput != nil {
		for i := 0; i < _NUM_OUTPUTS; i++ {
			var us int = int(pc[3*i]) | int(pc[3*i+1])<<8 | int(pc[3*i+2])<<16
			arOutput[i] = float32(us) / 16777215.0
		}
	}
	if arEquity != nil {
		for i := 0; i < 4; i++ {
			var us int = int(pc[15+3*i]) | int(pc[15+3*i+1])<<8 | int(pc[15+3*i+2])<<16
			arEquity[i] = (float32(us)/16777215.0 - 0.5) * 6.0
		}
	}
	return nil
}
