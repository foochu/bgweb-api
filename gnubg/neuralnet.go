package gnubg

import (
	"bgweb-api/gnubg/sigmoid"
	"fmt"
	"io/fs"
)

const _WEIGHTS_VERSION = "1.00"

type _NeuralNet struct {
	cInput            int
	cHidden           int
	cOutput           int
	nTrained          int
	rBetaHidden       float32
	rBetaOutput       float32
	arHiddenWeight    []float32
	arOutputWeight    []float32
	arHiddenThreshold []float32
	arOutputThreshold []float32
}

type _NNEvalType int

const (
	_NNEVAL_NONE _NNEvalType = iota
	_NNEVAL_SAVE
	_NNEVAL_FROMBASE
)

type _NNStateType int

const (
	_NNSTATE_NONE _NNStateType = iota - 1
	_NNSTATE_INCREMENTAL
	_NNSTATE_DONE
)

type _NNState struct {
	state       _NNStateType
	savedBase   []float32
	savedIBase  []float32
	cSavedIBase int
}

func neuralNetCreate(pnn *_NeuralNet, cInput int, cHidden int, cOutput int, rBetaHidden float32, rBetaOutput float32) {
	pnn.cInput = cInput
	pnn.cHidden = cHidden
	pnn.cOutput = cOutput
	pnn.rBetaHidden = rBetaHidden
	pnn.rBetaOutput = rBetaOutput
	pnn.nTrained = 0
	pnn.arHiddenWeight = make([]float32, cHidden*cInput)
	pnn.arOutputWeight = make([]float32, cOutput*cHidden)
	pnn.arHiddenThreshold = make([]float32, cHidden)
	pnn.arOutputThreshold = make([]float32, cOutput)
}

func neuralNetDestroy(pnn *_NeuralNet) {
	pnn.arHiddenWeight = nil
	pnn.arOutputWeight = nil
	pnn.arHiddenThreshold = nil
	pnn.arOutputThreshold = nil
}

func verifyWeights(pf fs.File, szFilename string) error {
	var file_version string
	if n, err := fmt.Fscanf(pf, "GNU Backgammon %15s\n", &file_version); n != 1 || err != nil {
		return fmt.Errorf("%v is not a weights file: %v/%v", szFilename, n, err)
	}
	if file_version != _WEIGHTS_VERSION {
		return fmt.Errorf("weights file %v, has incorrect version (%v), expected (%v)", szFilename, file_version, _WEIGHTS_VERSION)
	}
	return nil
}

func neuralNetEvaluate(pnn *_NeuralNet, arInput *[_NUM_INPUTS]float32, arOutput *[_NUM_OUTPUTS]float32, pnState *_NNState) error {
	ar := make([]float32, pnn.cHidden)
	// var s string

	// s += "=== neuralNetEvaluate()\n"
	// s += fmt.Sprintf(" arInput: %v\n", arInput)

	switch _NNevalAction(pnState) {
	case _NNEVAL_NONE:
		evaluate(pnn, arInput, ar, arOutput, nil)

	case _NNEVAL_SAVE:
		pnState.cSavedIBase = pnn.cInput
		pnState.savedBase = make([]float32, pnn.cHidden)
		pnState.savedIBase = make([]float32, pnn.cInput)
		copy(pnState.savedIBase, (*arInput)[:])
		evaluate(pnn, arInput, ar, arOutput, pnState.savedBase)

	case _NNEVAL_FROMBASE:
		if pnState.cSavedIBase != pnn.cInput {
			evaluate(pnn, arInput, ar, arOutput, nil)
			break
		}
		copy(ar, pnState.savedBase)

		r := arInput[:]
		s := pnState.savedIBase[:]

		for i := 0; i < pnn.cInput; i, r, s = i+1, r[1:], s[1:] {
			if len(r) == 0 {
				break
			}
			if r[0] != s[0] {
				r[0] -= s[0]
			} else {
				r[0] = 0.0
			}
		}

		evaluateFromBase(pnn, arInput, ar, arOutput)
	}

	// s += fmt.Sprintf(" arOutput: %v\n", arOutput)

	// fmt.Printf("%v", s)

	// if 1 == 1 {
	// 	os.Exit(0)
	// }

	return nil
}

/* separate context for race, crashed, contact
 * -1: regular eval
 * 0: save base
 * 1: from base
 */
func _NNevalAction(pnState *_NNState) _NNEvalType {
	if pnState == nil {
		return _NNEVAL_NONE
	}
	switch pnState.state {
	case _NNSTATE_NONE:
		/* incremental evaluation not useful */
		return _NNEVAL_NONE
	case _NNSTATE_INCREMENTAL:
		/* next call should return FROMBASE */
		pnState.state = _NNSTATE_DONE

		/* starting a new context; save base in the hope it will be useful */
		return _NNEVAL_SAVE
	case _NNSTATE_DONE:
		/* context hit!  use the previously computed base */
		return _NNEVAL_FROMBASE
	}
	/* never reached */
	return _NNEVAL_NONE /* for the picky compiler */
}

func evaluate(pnn *_NeuralNet, arInput *[_NUM_INPUTS]float32, ar []float32, arOutput *[_NUM_OUTPUTS]float32, saveAr []float32) {
	cHidden := pnn.cHidden

	/* Calculate activity at hidden nodes */
	for i := 0; i < cHidden; i++ {
		ar[i] = pnn.arHiddenThreshold[i]
	}
	prWeight := pnn.arHiddenWeight[:]

	for i := 0; i < pnn.cInput; i++ {
		var ari float32 = arInput[i]

		if ari == 0.0 {
			prWeight = prWeight[cHidden:]
		} else {
			pr := ar[:]

			if ari == 1.0 {
				for j := cHidden; j > 0; j-- {
					//  *pr++ += *prWeight++;
					pr[0] += prWeight[0]
					pr = pr[1:]
					prWeight = prWeight[1:]
				}
			} else {
				for j := cHidden; j > 0; j-- {
					//  *pr++ += *prWeight++ * ari;
					pr[0] += prWeight[0] * ari
					pr = pr[1:]
					prWeight = prWeight[1:]
				}
			}
		}
	}

	if saveAr != nil {
		copy(saveAr, ar)
	}

	for i := 0; i < cHidden; i++ {
		ar[i] = sigmoid.Sigmoid(-pnn.rBetaHidden * ar[i])
	}
	/* Calculate activity at output nodes */
	prWeight = pnn.arOutputWeight[:]

	for i := 0; i < pnn.cOutput; i++ {
		r := pnn.arOutputThreshold[i]

		for j := 0; j < cHidden; j++ {
			r += ar[j] * prWeight[0]
			prWeight = prWeight[1:]
		}
		arOutput[i] = sigmoid.Sigmoid(-pnn.rBetaOutput * r)
	}
}

func evaluateFromBase(pnn *_NeuralNet, arInputDif *[_NUM_INPUTS]float32, ar []float32, arOutput *[_NUM_OUTPUTS]float32) {
	/* Calculate activity at hidden nodes */
	/*    for( i = 0; i < pnn->cHidden; i++ )
	 * ar[ i ] = pnn->arHiddenThreshold[ i ]; */

	prWeight := pnn.arHiddenWeight

	for i := 0; i < pnn.cInput; i++ {
		var ari float32 = arInputDif[i]

		if ari == 0.0 {
			prWeight = prWeight[pnn.cHidden:]
		} else {
			pr := ar[:]

			if ari == 1.0 {
				for j := pnn.cHidden; j > 0; j-- {
					//  *pr++ += *prWeight++;
					pr[0] += prWeight[0]
					pr = pr[1:]
					prWeight = prWeight[1:]
				}
			} else if ari == -1.0 {
				for j := pnn.cHidden; j > 0; j-- {
					//  *pr++ -= *prWeight++;
					pr[0] -= prWeight[0]
					pr = pr[1:]
					prWeight = prWeight[1:]
				}
			} else {
				for j := pnn.cHidden; j > 0; j-- {
					//  *pr++ += *prWeight++ * ari;
					pr[0] += prWeight[0] * ari
					pr = pr[1:]
					prWeight = prWeight[1:]
				}
			}
		}
	}

	for i := 0; i < pnn.cHidden; i++ {
		ar[i] = sigmoid.Sigmoid(-pnn.rBetaHidden * ar[i])
	}

	/* Calculate activity at output nodes */
	prWeight = pnn.arOutputWeight[:]

	for i := 0; i < pnn.cOutput; i++ {
		r := pnn.arOutputThreshold[i]

		for j := 0; j < pnn.cHidden; j++ {
			r += ar[j] * prWeight[0]
			prWeight = prWeight[1:]
		}

		arOutput[i] = sigmoid.Sigmoid(-pnn.rBetaOutput * r)
	}
}

func neuralNetLoad(pnn *_NeuralNet, pf fs.File) error {
	var dummy string

	items, err := fmt.Fscanf(pf, "%d %d %d %s %f %f\n", &pnn.cInput, &pnn.cHidden, &pnn.cOutput, &dummy, &pnn.rBetaHidden, &pnn.rBetaOutput)
	if err != nil {
		return fmt.Errorf("error while reading neural net file: %v", err)
	}
	if items < 5 || pnn.cInput < 1 || pnn.cHidden < 1 || pnn.cOutput < 1 || pnn.rBetaHidden <= 0.0 || pnn.rBetaOutput <= 0.0 {
		return fmt.Errorf("invalid neural net file")
	}

	neuralNetCreate(pnn, pnn.cInput, pnn.cHidden, pnn.cOutput, pnn.rBetaHidden, pnn.rBetaOutput)

	pnn.nTrained = 1

	scan := func(ar []float32, len int) error {
		for i, pr := len, 0; i > 0; i, pr = i-1, pr+1 {
			if n, err := fmt.Fscanf(pf, "%f\n", &ar[pr]); n < 1 || err != nil {
				return fmt.Errorf("invalid neural net file: %v/%v", n, err)
			}
		}
		return nil
	}

	if err := scan(pnn.arHiddenWeight, pnn.cInput*pnn.cHidden); err != nil {
		return err
	}

	if err := scan(pnn.arOutputWeight, pnn.cHidden*pnn.cOutput); err != nil {
		return err
	}

	if err := scan(pnn.arHiddenThreshold, pnn.cHidden); err != nil {
		return err
	}

	if err := scan(pnn.arOutputThreshold, pnn.cOutput); err != nil {
		return err
	}

	return nil
}
