package gnubg

import (
	"bgweb-api/gnubg/math32"
	"bgweb-api/gnubg/met"
	"encoding/xml"
	"io/fs"
	"io/ioutil"
)

const _MAXSCORE = 64
const _MAXCUBELEVEL = 7

const _GAMMONRATE = 0.25

/*
 * A1 (A2) is the match equity of player 1 (2)
 * Btilde is the post-crawford match equities.
 */

var aafMET [_MAXSCORE][_MAXSCORE]float32
var aafMETPostCrawford [2][_MAXSCORE]float32

var aaaafGammonPrices [_MAXCUBELEVEL][_MAXSCORE][_MAXSCORE][4]float32
var aaaafGammonPricesPostCrawford [_MAXCUBELEVEL][_MAXSCORE][2][4]float32

/* enums for the entries in the arrays returned by getMEMultiple
 * DoublePass, DoubleTakeWin, DoubleTakeWinGammon... for the first 8
 * then the same values using CubePrimeValues
 * DP = Double/Pass, DTWG = double/take/wing gammon, etc
 * DPP = double/Pass with CubePrime values, etc.
 */
const (
	/* player 0 wins, first cube value */
	_DP   int = 0
	_NDW  int = 0
	_DTW  int = 1
	_NDWG int = 1
	_NDWB int = 2
	_DTWG int = 3
	_DTWB int = 4
	/* player 0 loses, first cube value */
	_NDL  int = 5
	_DTL  int = 6
	_NDLG int = 6
	_NDLB int = 7
	_DTLG int = 7
	_DTLB int = 8
	/* player 0 wins, 2nd cube value */
	_DPP0   int = 9
	_DTWP0  int = 10
	_NDWBP0 int = 11
	_DTWGP0 int = 12
	_DTWBP0 int = 13
	/* player 0 loses, 2nd cube value */
	_NDLP0  int = 14
	_DTLP0  int = 15
	_NDLBP0 int = 16
	_DTLGP0 int = 17
	_DTLBP0 int = 18
	/* player 0 wins, 3rd cube value */
	_DPP1   int = 19
	_DTWP1  int = 20
	_NDWBP1 int = 21
	_DTWGP1 int = 22
	_DTWBP1 int = 23
	/* player 0 loses, 3rd cube value */
	_NDLP1  int = 24
	_DTLP1  int = 25
	_NDLBP1 int = 26
	_DTLGP1 int = 27
	_DTLBP1 int = 28
)

// var miCurrent *met.METInfo

func readMET(met *met.METData, dataDir fs.FS, filename string) int {
	met.Info.FileName = filename

	xmlFile, err := dataDir.Open(filename)
	if err != nil {
		panic(err)
	}
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	if err := xml.Unmarshal(byteValue, met); err != nil {
		panic(err)
	}

	if met.PostCrawford[0].Player == "both" {
		met.PostCrawford = append(met.PostCrawford, met.PostCrawford[0])
	}

	met.PreCrawford.Parameters.Name = met.PreCrawford.Type

	for i := 0; i < len(met.PostCrawford); i++ {
		met.PostCrawford[i].Parameters.Name = met.PostCrawford[i].Type
	}

	return 0
}

func initMatchEquity(dataDir fs.FS, szFileName string) {
	md := met.METData{}

	/* Read match equity table from XML file */
	if readMET(&md, dataDir, szFileName) != 0 { /* load failed - make default as must have a met */
		// TODO: getDefaultMET(&md)
		panic("readMET() != 0")
	}

	/* Copy met to current met, extend met (if needed) */

	/* post-Crawford table */
	for j := 0; j < 2; j++ {
		if md.PostCrawford[j].Parameters.Name == "explicit" {
			/* copy and extend table */

			/* FIXME: implement better extension of post-Crawford table */

			/* Note that the post Crawford table is extended from
			 * n - 1 as the  post Crawford table of a n match equity table
			 * might not include the post Crawford equity at n-away, since
			 * the first "legal" post Crawford score is n-1. */

			for i := 0; i < md.Info.Length-1; i++ {
				aafMETPostCrawford[j][i] = md.PostCrawford[j].Row.ME[i]
			}

			initPostCrawfordMET(&aafMETPostCrawford[j], md.Info.Length-1, _GAMMONRATE, 0.015, 0.004)

		} else {

			/* generate match equity table using Zadeh's formula */

			panic("TODO: not implemented")
			// if initPostCrawfordMETFromParameters(aafMETPostCrawford[j], &md.ampPostCrawford[j]) < 0 {

			// 	fprintf(stderr, _("Error generating post-Crawford MET\n"))
			// 	return

			// }

		}
	}

	// /* pre-Crawford table */
	if md.PreCrawford.Parameters.Name == "explicit" { /* copy table */
		for i := 0; i < md.Info.Length; i++ {
			for j := 0; j < md.Info.Length; j++ {
				aafMET[i][j] = md.PreCrawford.Rows[i].ME[j]
			}
		}
	} else {
		/* generate match equity table using Zadeh's formula */
		panic("TODO: not implemented")
		// if (initMETFromParameters(aafMET, aafMETPostCrawford, &md.mpPreCrawford) < 0) {

		//     fprintf(stderr, _("Error generating pre-Crawford MET\n"));
		//     return;
		// }
	}

	// /* Extend match equity table */
	extendMET(&aafMET, md.Info.Length)

	/* save match equity table information */
	// miCurrent = &md.Info

	// /* initialise gammon prices */
	calcGammonPrices(&aafMET, &aafMETPostCrawford, &aaaafGammonPrices, &aaaafGammonPricesPostCrawford)
}

func initPostCrawfordMET(afMETPostCrawford *[_MAXSCORE]float32, iStart int, rG float32, rFD2 float32, rFD4 float32) {
	/*
	 * Calculate post-crawford match equities
	 */

	for i := iStart; i < _MAXSCORE; i++ {
		if i-4 >= 0 {
			if i-2 >= 0 {
				afMETPostCrawford[i] = rG*0.5*afMETPostCrawford[i-4] + (1.0-rG)*0.5*afMETPostCrawford[i-2]
			} else {
				afMETPostCrawford[i] = rG*0.5*afMETPostCrawford[i-4] + (1.0-rG)*0.5*1.0
			}
		} else {
			if i-2 >= 0 {
				afMETPostCrawford[i] = rG*0.5*1.0 + (1.0-rG)*0.5*afMETPostCrawford[i-2]
			} else {
				afMETPostCrawford[i] = rG*0.5*1.0 + (1.0-rG)*0.5*1.0
			}
		}

		/*"insane post crawford equity" */
		if afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0 {
			panic("afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0")
		}
		/*
		 * add 1.5% at 1-away, 2-away for the free drop
		 * add 0.4% at 1-away, 4-away for the free drop
		 */

		if i == 1 {
			afMETPostCrawford[i] -= rFD2
		}
		/*"insane post crawford equity(1)" */
		if afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0 {
			panic("afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0")
		}

		if i == 3 {
			afMETPostCrawford[i] -= rFD4
		}
		/*"insane post crawford equity(2)" */
		if afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0 {
			panic("afMETPostCrawford[i] < 0.0 && afMETPostCrawford[i] > 1.0")
		}
	}

}

func extendMET(aarMET *[_MAXSCORE][_MAXSCORE]float32, nMaxScore int) {
	arStddevTable := []float32{0, 1.24, 1.27, 1.47, 1.50, 1.60, 1.61, 1.66, 1.68, 1.70, 1.72, 1.77}

	var rStddev0, rStddev1, rGames, rSigma float32

	/* Extend match equity table */
	for i := nMaxScore; i < _MAXSCORE; i++ {
		nScore0 := i + 1

		if nScore0 > 10 {
			rStddev0 = 1.77
		} else {
			rStddev0 = arStddevTable[nScore0]
		}

		for j := 0; j <= i; j++ {
			nScore1 := j + 1

			rGames = float32(nScore0+nScore1) / 2.0

			if nScore1 > 10 {
				rStddev1 = 1.77
			} else {
				rStddev1 = arStddevTable[nScore1]
			}

			rSigma = math32.Sqrtf(rStddev0*rStddev0+rStddev1*rStddev1) * math32.Sqrtf(rGames)

			if 6.0*rSigma > float32(nScore0-nScore1) {
				aarMET[i][j] = normalDistArea(float32(nScore0-nScore1), 6.0*rSigma, 0.0, rSigma)
			} else {
				aarMET[i][j] = 0.0
			}
		}
	}

	/* Generate j > i part of MET */

	for i := 0; i < _MAXSCORE; i++ {
		var from int
		if i < nMaxScore {
			from = nMaxScore
		} else {
			from = i + 1
		}
		for j := from; j < _MAXSCORE; j++ {
			aarMET[i][j] = 1.0 - aarMET[j][i]
		}
	}
}

func normalDistArea(rMin float32, rMax float32, rMu float32, rSigma float32) float32 {
	var rtMin, rtMax float32
	var rInt1, rInt2 float32

	rtMin = (rMin - rMu) / rSigma
	rtMax = (rMax - rMu) / rSigma

	rInt1 = (math32.Erff(rtMin/math32.Sqrtf(2)) + 1.0) / 2.0
	rInt2 = (math32.Erff(rtMax/math32.Sqrtf(2)) + 1.0) / 2.0

	return rInt2 - rInt1
}

func calcGammonPrices(aafMET *[_MAXSCORE][_MAXSCORE]float32, aafMETPostCrawford *[2][_MAXSCORE]float32, aaaafGammonPrices *[_MAXCUBELEVEL][_MAXSCORE][_MAXSCORE][4]float32, aaaafGammonPricesPostCrawford *[_MAXCUBELEVEL][_MAXSCORE][2][4]float32) {
	for i, nCube := 0, 1; i < _MAXCUBELEVEL; i, nCube = i+1, nCube*2 {
		for j := 0; j < _MAXSCORE; j++ {
			for k := 0; k < _MAXSCORE; k++ {
				getGammonPrice(&aaaafGammonPrices[i][j][k],
					_MAXSCORE-j-1, _MAXSCORE-k-1, _MAXSCORE,
					nCube, (_MAXSCORE == j) || (_MAXSCORE == k), aafMET, aafMETPostCrawford)
			}
		}
	}
	for i, nCube := 0, 1; i < _MAXCUBELEVEL; i, nCube = i+1, nCube*2 {
		for j := 0; j < _MAXSCORE; j++ {
			getGammonPrice(&aaaafGammonPricesPostCrawford[i][j][0],
				_MAXSCORE-1, _MAXSCORE-j-1, _MAXSCORE, nCube, false, aafMET, aafMETPostCrawford)
			getGammonPrice(&aaaafGammonPricesPostCrawford[i][j][1],
				_MAXSCORE-j-1, _MAXSCORE-1, _MAXSCORE, nCube, false, aafMET, aafMETPostCrawford)
		}
	}
}

func getGammonPrice(arGammonPrice *[4]float32, nScore0 int, nScore1 int, nMatchTo int, nCube int, fCrawford bool, aafMET *[_MAXSCORE][_MAXSCORE]float32, aafMETPostCrawford *[2][_MAXSCORE]float32) {
	epsilon := float32(1.0e-7)

	rWin := getME(nScore0, nScore1, nMatchTo, 0, nCube, 0, fCrawford, aafMET, aafMETPostCrawford)

	rWinGammon := getME(nScore0, nScore1, nMatchTo, 0, 2*nCube, 0, fCrawford, aafMET, aafMETPostCrawford)

	rWinBG := getME(nScore0, nScore1, nMatchTo, 0, 3*nCube, 0, fCrawford, aafMET, aafMETPostCrawford)

	rLose := getME(nScore0, nScore1, nMatchTo, 0, nCube, 1, fCrawford, aafMET, aafMETPostCrawford)

	rLoseGammon := getME(nScore0, nScore1, nMatchTo, 0, 2*nCube, 1, fCrawford, aafMET, aafMETPostCrawford)

	rLoseBG := getME(nScore0, nScore1, nMatchTo, 0, 3*nCube, 1, fCrawford, aafMET, aafMETPostCrawford)

	rCenter := (rWin + rLose) / 2.0

	/* FIXME: correct numerical problems in a better way, than done
	 * below. If cube is dead gammon or backgammon price might be a
	 * small negative number. For example, at -2,-3 with cube on 2
	 * the current code gives: 0.9090..., 0, -2.7e-8, 0 instead
	 * of the correct 0.9090..., 0, 0, 0. */

	/* avoid division by zero */

	if math32.Fabsf(rWin-rCenter) > epsilon {

		/* this expression can be reduced to:
		 * 2 * ( rWinGammon - rWin ) / ( rWin - rLose )
		 * which is twice the "usual" gammon value */

		arGammonPrice[0] = (rWinGammon-rCenter)/(rWin-rCenter) - 1.0

		/* this expression can be reduced to:
		 * 2 * ( rLose - rLoseGammon ) / ( rWin - rLose )
		 * which is twice the "usual" gammon value */

		arGammonPrice[1] = (rCenter-rLoseGammon)/(rWin-rCenter) - 1.0

		arGammonPrice[2] = (rWinBG-rCenter)/(rWin-rCenter) - (arGammonPrice[0] + 1.0)
		arGammonPrice[3] = (rCenter-rLoseBG)/(rWin-rCenter) - (arGammonPrice[1] + 1.0)

	} else {
		arGammonPrice[0] = 0.0
		arGammonPrice[1] = 0.0
		arGammonPrice[2] = 0.0
		arGammonPrice[3] = 0.0
	}

	/* Correct numerical problems */
	if arGammonPrice[0] < 0.0 {
		arGammonPrice[0] = 0.0
	}
	if arGammonPrice[1] < 0.0 {
		arGammonPrice[1] = 0.0
	}
	if arGammonPrice[2] < 0.0 {
		arGammonPrice[2] = 0.0
	}
	if arGammonPrice[3] < 0.0 {
		arGammonPrice[3] = 0.0
	}
}

/*
 * Return match equity (mwc) assuming player fWhoWins wins nPoints points.
 *
 * If fCrawford then afMETPostCrawford is used, otherwise
 * aafMET is used.
 *
 * Input:
 *    nAway0: points player 0 needs to win
 *    nAway1: points player 1 needs to win
 *    fPlayer: get mwc for this player
 *    fCrawford: is this the Crawford game
 *    aafMET: match equity table for player 0
 *    afMETPostCrawford: post-Crawford match equity table for player 0
 *
 */
func getME(nScore0 int, nScore1 int, nMatchTo int, fPlayer int, nPoints int, fWhoWins int, fCrawford bool, aafMET *[_MAXSCORE][_MAXSCORE]float32, aafMETPostCrawford *[2][_MAXSCORE]float32) float32 {
	n0 := nMatchTo - (nScore0 + (1-fWhoWins)*nPoints) - 1
	n1 := nMatchTo - (nScore1 + fWhoWins*nPoints) - 1

	/* check if any player has won the match */

	if n0 < 0 {
		/* player 0 has won the game */
		if fPlayer > 0 {
			return 0.0
		}
		return 1.0
	} else if n1 < 0 {
		/* player 1 has won the game */
		if fPlayer > 0 {
			return 1.0
		}
		return 0.0
	}
	/* the match is not finished */

	if fCrawford || (nMatchTo-nScore0 == 1) || (nMatchTo-nScore1 == 1) {

		/* the next game will be post-Crawford */

		if n0 == 0 {
			/* player 0 is leading match */
			/* FIXME: use pc-MET for player 0 */
			if fPlayer > 0 {
				return aafMETPostCrawford[1][n1]
			}
			return 1.0 - aafMETPostCrawford[1][n1]
		} else {
			/* player 1 is leading the match */
			if fPlayer > 0 {
				return 1.0 - aafMETPostCrawford[0][n0]
			}
			return aafMETPostCrawford[0][n0]
		}
	} else {
		/* non-post-Crawford games */
		if fPlayer > 0 {
			return 1.0 - aafMET[n0][n1]
		}
		return aafMET[n0][n1]
	}
}

/* given a match score, return a pair of arrays with the METs for
 * player0 and player 1 winning/losing including gammons & backgammons
 *
 * if nCubePrime0 < 0, then we're only interested in the first
 * values in each array, using nCube.
 *
 * Otherwise, if nCubePrime0 >= 0, we do another set of METs with
 * both sides using nCubePrime0
 *
 * if nCubePrime1 >= 0, we do a third set using nCubePrime1
 *
 * FIXME ? It looks like if nCubePrime0 >= 0, nCubePrime1 is as well
 *         That could simplify the code below a little
 *
 * This reduces the *huge* number of calls to get equity table entries
 * when analyzing matches by something like 40 times
 */
func getMEMultiple(nScore0 int, nScore1 int, nMatchTo int,
	nCube int, nCubePrime0 int, nCubePrime1 int,
	fCrawford bool, aafMET *[_MAXSCORE][_MAXSCORE]float32,
	aafMETPostCrawford *[2][_MAXSCORE]float32, player0 []float32, player1 []float32) {

	var scores [2][_DTLBP1 + 1]int /* the resulting match scores */
	var max_res int
	var score0 []int
	var score1 []int
	var mult = []int{1, 2, 3, 4, 6}
	var p0 []float32
	var p1 []float32
	var f float32
	var away0, away1 int
	var fCrawf bool = fCrawford

	/* figure out how many results we'll be returning */
	if nCubePrime0 < 0 {
		max_res = _DTLB + 1
	} else if nCubePrime1 < 0 {
		max_res = _DTLBP0 + 1
	} else {
		max_res = _DTLBP1 + 1
	}

	/* set up a table of resulting match scores for all
	 * the results we're calculating */
	score0 = scores[0][:]
	score1 = scores[1][:]
	away0 = nMatchTo - nScore0 - 1
	away1 = nMatchTo - nScore1 - 1
	fCrawf = fCrawf || ((nMatchTo-nScore0 == 1) || (nMatchTo-nScore1 == 1))

	/* player 0 wins normal, doubled, gammon, backgammon */
	for i := 0; i < _NDL; i++ {
		score0[0] = away0 - mult[i]*nCube
		score0 = score0[1:]
		score1[0] = away1
		score1 = score1[1:]
	}
	/* player 1 wins normal, doubled, etc. */
	for i := 0; i < _NDL; i++ {
		score0[0] = away0
		score0 = score0[1:]
		score1[0] = away1 - mult[i]*nCube
		score1 = score1[1:]
	}

	if max_res > _DPP0 {
		/* same using the second cube value */
		for i := 0; i < _NDL; i++ {
			score0[0] = away0 - mult[i]*nCubePrime0
			score0 = score0[1:]
			score1[0] = away1
			score1 = score1[1:]
		}
		for i := 0; i < _NDL; i++ {
			score0[0] = away0
			score0 = score0[1:]
			score1[0] = away1 - mult[i]*nCubePrime0
			score1 = score1[1:]
		}
		if max_res > _DPP1 {
			/* same using the third cube value */
			for i := 0; i < _NDL; i++ {
				score0[0] = away0 - mult[i]*nCubePrime1
				score0 = score0[1:]
				score1[0] = away1
				score1 = score1[1:]
			}
			for i := 0; i < _NDL; i++ {
				score0[0] = away0
				score0 = score0[1:]
				score1[0] = away1 - mult[i]*nCubePrime1
				score1 = score1[1:]
			}
		}
	}

	score0 = scores[0][:]
	score1 = scores[1][:]
	p0 = player0
	p1 = player1

	/* now go through the resulting scores, looking up the equities */
	for i := 0; i < max_res; i++ {
		var s0 int = score0[0]
		score0 = score0[1:]
		var s1 int = score1[0]
		score1 = score1[1:]

		if s0 < 0 {
			/* player 0 wins */
			p0[0] = 1.0
			p0 = p0[1:]
			p1[0] = 0.0
			p1 = p1[1:]
		} else if s1 < 0 {
			p0[0] = 0.0
			p0 = p0[1:]
			p1[0] = 1.0
			p1 = p1[1:]
		} else if fCrawf {
			if s0 == 0 { /* player 0 is leading */
				p0[0] = 1.0 - aafMETPostCrawford[1][s1]
				p0 = p0[1:]
				p1[0] = aafMETPostCrawford[1][s1]
				p1 = p1[1:]
			} else {
				p0[0] = aafMETPostCrawford[0][s0]
				p0 = p0[1:]
				p1[0] = 1.0 - aafMETPostCrawford[0][s0]
				p1 = p1[1:]
			}
		} else { /* non-post-Crawford */
			p0[0] = aafMET[s0][s1]
			p0 = p0[1:]
			p1[0] = 1.0 - aafMET[s0][s1]
			p1 = p1[1:]
		}
	}

	/* results for player 0 are done, results for player 1 have the
	 *  losses in cols 0-4 and 8-12, but we want them to be in the same
	 *  order as results0 - e.g wins in cols 0-4, and 8-12
	 */
	p0 = player1
	p1 = player1[_NDL:]
	for i := 0; i < _NDL; i++ {
		f = p0[0]
		p0[0] = p1[0]
		p0 = p0[1:]
		p1[0] = f
		p1 = p1[1:]
	}

	if max_res > _DTLBP0 {
		p0 = p0[_NDL:]
		p1 = p1[_NDL:]
		for i := 0; i < _NDL; i++ {
			f = p0[0]
			p0[0] = p1[0]
			p0 = p0[1:]
			p1[0] = f
			p1 = p1[1:]
		}
	}

	if max_res > _DTLBP1 {
		p0 = p0[_NDL:]
		p1 = p1[_NDL:]
		for i := 0; i < _NDL; i++ {
			f = p0[0]
			p0[0] = p1[0]
			p0 = p0[1:]
			p1[0] = f
			p1 = p1[1:]
		}
	}

}

func getPoints(arOutput *[5]float32, pci *_CubeInfo, arCP [2]float32) int {

	/*
	 * Input:
	 * - arOutput: we need the gammon and backgammon ratios
	 *   (we assume arOutput is evaluate for pci -> fMove)
	 * - anScore: the current score.
	 * - nMatchTo: matchlength
	 * - pci: value of cube, who's turn is it
	 *
	 *
	 * Output:
	 * - arCP : cash points with live cube
	 * These points are necessary for the linear
	 * interpolation used in cubeless -> cubeful equity
	 * transformation.
	 */

	/* Match play */

	/* normalize score */

	var i int = pci.nMatchTo - pci.anScore[0] - 1
	var j int = pci.nMatchTo - pci.anScore[1] - 1

	var nCube int = pci.nCube

	var arCPLive [2][_MAXCUBELEVEL]float32
	var arCPDead [2][_MAXCUBELEVEL]float32
	var arG [2]float32
	var arBG [2]float32

	var rDP, rRDP, rDTW, rDTL float32

	var nDead, n, nMax, nCubeValue, k int

	var aarMETResults [2][_DTLBP1 + 1]float32

	/* Gammon and backgammon ratio's.
	 * Avoid division by zero in extreme cases. */

	if pci.fMove == 0 {

		/* arOutput evaluated for player 0 */

		if arOutput[_OUTPUT_WIN] > 0.0 {
			arG[0] = (arOutput[_OUTPUT_WINGAMMON] - arOutput[_OUTPUT_WINBACKGAMMON]) / arOutput[_OUTPUT_WIN]
			arBG[0] = arOutput[_OUTPUT_WINBACKGAMMON] / arOutput[_OUTPUT_WIN]
		} else {
			arG[0] = 0.0
			arBG[0] = 0.0
		}

		if arOutput[_OUTPUT_WIN] < 1.0 {
			arG[1] = (arOutput[_OUTPUT_LOSEGAMMON] - arOutput[_OUTPUT_LOSEBACKGAMMON]) / (1.0 - arOutput[_OUTPUT_WIN])
			arBG[1] = arOutput[_OUTPUT_LOSEBACKGAMMON] / (1.0 - arOutput[_OUTPUT_WIN])
		} else {
			arG[1] = 0.0
			arBG[1] = 0.0
		}

	} else {

		/* arOutput evaluated for player 1 */

		if arOutput[_OUTPUT_WIN] > 0.0 {
			arG[1] = (arOutput[_OUTPUT_WINGAMMON] - arOutput[_OUTPUT_WINBACKGAMMON]) / arOutput[_OUTPUT_WIN]
			arBG[1] = arOutput[_OUTPUT_WINBACKGAMMON] / arOutput[_OUTPUT_WIN]
		} else {
			arG[1] = 0.0
			arBG[1] = 0.0
		}

		if arOutput[_OUTPUT_WIN] < 1.0 {
			arG[0] = (arOutput[_OUTPUT_LOSEGAMMON] - arOutput[_OUTPUT_LOSEBACKGAMMON]) / (1.0 - arOutput[_OUTPUT_WIN])
			arBG[0] = arOutput[_OUTPUT_LOSEBACKGAMMON] / (1.0 - arOutput[_OUTPUT_WIN])
		} else {
			arG[0] = 0.0
			arBG[0] = 0.0
		}
	}

	/* Find out what value the cube has when you or your
	 * opponent give a dead cube. */

	nDead = nCube
	nMax = 0

	for (i >= 2*nDead) && (j >= 2*nDead) {
		nMax++
		nDead *= 2
	}

	for nCubeValue, n = nDead, nMax; n >= 0; nCubeValue, n = nCubeValue>>1, n-1 {

		/* Calculate dead and live cube cash points.
		 * See notes by me (Joern Thyssen) available from the
		 * 'doc' directory.  (FIXME: write notes :-) ) */

		/* Even though it's a dead cube we take account of the opponents
		 * automatic redouble. */

		/* Dead cube cash point for player 0 */

		getMEMultiple(pci.anScore[0], pci.anScore[1], pci.nMatchTo, nCubeValue, getCubePrimeValue(i, j, nCubeValue), /* 0 */
			getCubePrimeValue(j, i, nCubeValue), /* 1 */
			pci.fCrawford, &aafMET, &aafMETPostCrawford, aarMETResults[0][:], aarMETResults[1][:])

		for k = 0; k < 2; k++ {

			/* Live cube cash point for player */

			if (i < 2*nCubeValue) || (j < 2*nCubeValue) {
				var i1, i2, i3 int
				if k > 0 {
					i1 = _DTLP1
					i2 = _DTLGP1
					i3 = _DTLBP1
				} else {
					i1 = _DTLP0
					i2 = _DTLGP0
					i3 = _DTLBP0
				}
				rDTL = (1.0-arG[1-k]-arBG[1-k])*aarMETResults[k][i1] + arG[1-k]*aarMETResults[k][i2] + arBG[1-k]*aarMETResults[k][i3]

				rDP = aarMETResults[k][_DP]

				rDTW = (1.0-arG[k]-arBG[k])*aarMETResults[k][i1] + arG[k]*aarMETResults[k][i2] + arBG[k]*aarMETResults[k][i3]

				arCPDead[k][n] = (rDTL - rDP) / (rDTL - rDTW)

				/* The doubled cube is going to be dead */
				arCPLive[k][n] = arCPDead[k][n]

			} else {

				/* Doubled cube is alive */

				/* redouble, pass */
				rRDP = aarMETResults[k][_DTL]

				/* double, pass */
				rDP = aarMETResults[k][_DP]

				/* double, take win */

				rDTW = (1.0-arG[k]-arBG[k])*aarMETResults[k][_DTW] + arG[k]*aarMETResults[k][_DTWG] + arBG[k]*aarMETResults[k][_DTWB]

				arCPLive[k][n] = 1.0 - arCPLive[1-k][n+1]*(rDP-rDTW)/(rRDP-rDTW)

			}

		} /* loop k */

	}

	/* return cash point for current cube level */

	arCP[0] = arCPLive[0][0]
	arCP[1] = arCPLive[1][0]

	// #if 0
	//     for (n = nMax; n >= 0; n--) {

	//         printf("Cube %i\n"
	//                "Dead cube:    cash point 0 %6.3f\n"
	//                "              cash point 1 %6.3f\n"
	//                "Live cube:    cash point 0 %6.3f\n"
	//                "              cash point 1 %6.3f\n\n",
	//                n, arCPDead[0][n], arCPDead[1][n], arCPLive[0][n], arCPLive[1][n]);

	//     }
	// #endif

	return 0

}

func getCubePrimeValue(i int, j int, nCubeValue int) int {
	if (i < 2*nCubeValue) && (j >= 2*nCubeValue) {
		/* automatic double */
		return 2 * nCubeValue
	} else {
		return nCubeValue
	}
}
