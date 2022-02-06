package gnubg

type _PositionKey struct {
	data [7]int
}

func (t *_PositionKey) copyFrom(that _PositionKey) {
	for i := 0; i < len(t.data); i++ {
		t.data[i] = that.data[i]
	}
}

func (t _PositionKey) equals(that _PositionKey) bool {
	for i := 0; i < len(t.data); i++ {
		if t.data[i] != that.data[i] {
			return false
		}
	}
	return true
}

func (t *_PositionKey) fromBoard(anBoard _TanBoard) {
	var anpBoard *[7]int = &t.data

	for i, j := 0, 0; i < 3; i, j = i+1, j+8 {
		anpBoard[i] = anBoard[1][j] + (anBoard[1][j+1] << 4) + (anBoard[1][j+2] << 8) + (anBoard[1][j+3] << 12) + (anBoard[1][j+4] << 16) + (anBoard[1][j+5] << 20) + (anBoard[1][j+6] << 24) + (anBoard[1][j+7] << 28)
		anpBoard[i+3] = anBoard[0][j] + (anBoard[0][j+1] << 4) + (anBoard[0][j+2] << 8) + (anBoard[0][j+3] << 12) + (anBoard[0][j+4] << 16) + (anBoard[0][j+5] << 20) + (anBoard[0][j+6] << 24) + (anBoard[0][j+7] << 28)
	}
	anpBoard[6] = anBoard[0][24] + (anBoard[1][24] << 4)
}

func (t _PositionKey) toBoard(anBoard *_TanBoard) {
	var anpBoard *[7]int = &t.data

	for i, j := 0, 0; i < 3; i, j = i+1, j+8 {
		anBoard[1][j] = anpBoard[i] & 0x0f
		anBoard[1][j+1] = (anpBoard[i] >> 4) & 0x0f
		anBoard[1][j+2] = (anpBoard[i] >> 8) & 0x0f
		anBoard[1][j+3] = (anpBoard[i] >> 12) & 0x0f
		anBoard[1][j+4] = (anpBoard[i] >> 16) & 0x0f
		anBoard[1][j+5] = (anpBoard[i] >> 20) & 0x0f
		anBoard[1][j+6] = (anpBoard[i] >> 24) & 0x0f
		anBoard[1][j+7] = (anpBoard[i] >> 28) & 0x0f

		anBoard[0][j] = anpBoard[i+3] & 0x0f
		anBoard[0][j+1] = (anpBoard[i+3] >> 4) & 0x0f
		anBoard[0][j+2] = (anpBoard[i+3] >> 8) & 0x0f
		anBoard[0][j+3] = (anpBoard[i+3] >> 12) & 0x0f
		anBoard[0][j+4] = (anpBoard[i+3] >> 16) & 0x0f
		anBoard[0][j+5] = (anpBoard[i+3] >> 20) & 0x0f
		anBoard[0][j+6] = (anpBoard[i+3] >> 24) & 0x0f
		anBoard[0][j+7] = (anpBoard[i+3] >> 28) & 0x0f
	}
	anBoard[0][24] = anpBoard[6] & 0x0f
	anBoard[1][24] = (anpBoard[6] >> 4) & 0x0f
}

func positionFromBearoff(anBoard *[6]int, usID int, nPoints int, nChequers int) {
	fBits := positionInv(usID, nChequers+nPoints, nPoints)

	for i := 0; i < nPoints; i++ {
		anBoard[i] = 0
	}

	j := nPoints - 1
	for i := 0; i < (nChequers + nPoints); i++ {
		if fBits&(1<<i) > 0 {
			if j == 0 {
				break
			}
			j--
		} else {
			anBoard[j]++
		}
	}
}

func positionInv(nID int, n int, r int) int {
	var nC int

	if r == 0 {
		return 0
	} else if n == r {
		return (1 << n) - 1
	}

	nC = combination(n-1, r)

	if nID >= nC {
		return (1 << (n - 1)) | positionInv(nID-nC, n-1, r-1)
	} else {
		return positionInv(nID, n-1, r)
	}
}

const _MAX_N = 40
const _MAX_R = 25

var anCombination [_MAX_N][_MAX_R]int
var fCalculated bool

func initCombination() {
	for i := 0; i < _MAX_N; i++ {
		anCombination[i][0] = i + 1
	}

	for j := 1; j < _MAX_R; j++ {
		anCombination[0][j] = 0
	}

	for i := 1; i < _MAX_N; i++ {
		for j := 1; j < _MAX_R; j++ {
			anCombination[i][j] = anCombination[i-1][j-1] + anCombination[i-1][j]
		}
	}

	fCalculated = true
}

func combination(n int, r int) int {
	if n > _MAX_N && r > _MAX_R {
		panic("n > MAX_N && r > MAX_R")
	}

	if !fCalculated {
		initCombination()
	}

	return anCombination[n-1][r-1]
}

func positionBearoff(anBoard [6]int, nPoints int, nChequers int) int {
	var fBits, i, j int

	if nPoints == 0 {
		panic("zero point bearoff")
	}

	for j, i = nPoints-1, 0; i < nPoints; i++ {
		j += anBoard[i]
	}

	fBits = 1 << j

	for i = 0; i < nPoints-1; i++ {
		j -= anBoard[i] + 1
		fBits |= (1 << j)
	}

	return positionF(fBits, nChequers+nPoints, nPoints)
}

func positionF(fBits int, n int, r int) int {
	if n == r {
		return 0
	}

	if fBits&(1<<(n-1)) > 0 {
		return combination(n-1, r) + positionF(fBits, n-1, r-1)
	} else {
		return positionF(fBits, n-1, r)
	}
}

func positionIndex(g int, anBoard [6]int) int {
	var fBits int
	var j int

	if g == 0 {
		panic("g == 0")
	}

	j = g - 1

	for i := 0; i < g; i++ {
		j += anBoard[i]
	}

	fBits = 1 << j

	for i := 0; i < g-1; i++ {
		j -= anBoard[i] + 1
		fBits |= (1 << j)
	}

	/* FIXME: 15 should be replaced by nChequers, but the function is
	 * only called from bearoffgammon, so this should be fine. */
	return positionF(fBits, 15, g)
}
