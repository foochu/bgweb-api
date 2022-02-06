package gnubg

import "bgweb-api/internal/gnubg/sigmoid"

func neuralNetEvaluateSSE(pnn *_NeuralNet, arInput []float32, arOutput *[_NUM_OUTPUTS]float32, pnState *_NNState) error {
	ar := make([]float32, pnn.cHidden)

	// var s string

	// s += "=== neuralNetEvaluateSSE()\n"
	// s += fmt.Sprintf(" arInput: %v\n", arInput)

	evaluateSSE(pnn, arInput, ar, arOutput)

	// s += fmt.Sprintf(" arOutput: %v\n", arOutput)

	// fmt.Printf("%v", s)

	// if 1 == 1 {
	// 	os.Exit(0)
	// }

	return nil
}

func evaluateSSE(pnn *_NeuralNet, arInput []float32, ar []float32, arOutput *[_NUM_OUTPUTS]float32) {
	var cHidden int = pnn.cHidden

	/* Calculate activity at hidden nodes */
	copy(ar, pnn.arHiddenThreshold)
	prWeight := pnn.arHiddenWeight[:]

	if pnn.cInput != 214 { /* everything but the racing net */
		for i := 0; i < 200; { /* base inputs */
			var ari float32 = arInput[i]
			i++

			/* 3 binaries, 1 float */

			if ari == 0.0 {
				prWeight = prWeight[cHidden:]
			} else {
				for j := range ar {
					ar[j] += prWeight[j]
				}
				prWeight = prWeight[cHidden:]
			}

			ari = arInput[i]
			i++

			if ari == 0.0 {
				prWeight = prWeight[cHidden:]
			} else {
				for j := range ar {
					ar[j] += prWeight[j]
				}
				prWeight = prWeight[cHidden:]
			}

			ari = arInput[i]
			i++

			if ari == 0.0 {
				prWeight = prWeight[cHidden:]
				/* If 3rd element is 0, so is 4th. Skip it */
				prWeight = prWeight[cHidden:]
				i++
				continue
			} else {
				for j := range ar {
					ar[j] += prWeight[j]
				}
				prWeight = prWeight[cHidden:]
			}

			ari = arInput[i]
			i++

			if ari == 0.0 {
				prWeight = prWeight[cHidden:]
			} else {
				if ari == 1.0 {
					for j := range ar {
						ar[j] += prWeight[j]
					}
				} else {
					for j := range ar {
						ar[j] += prWeight[j] * ari
					}
				}
				prWeight = prWeight[cHidden:]
			} /* base inputs are done */
		}

		if pnn.cInput == 250 { /* Pruning nets are over, contact/crashed still have 2 * 25 floats */
			for i := 200; i < 250; i++ {
				var ari float32 = arInput[i]

				if ari == 0.0 {
					prWeight = prWeight[cHidden:]
				} else {
					for j := range ar {
						ar[j] += prWeight[j] * ari
					}
					prWeight = prWeight[cHidden:]
				}
			}
		}
	} else { /* racing net */
		for i := 0; i < pnn.cInput; i++ {
			var ari float32 = arInput[i]

			if ari == 0.0 {
				prWeight = prWeight[cHidden:]
			} else {
				if ari == 1.0 {
					for j := range ar {
						ar[j] += prWeight[j]
					}
				} else {
					for j := range ar {
						ar[j] += prWeight[j] * ari
					}
				}
				prWeight = prWeight[cHidden:]
			}
		}
	}

	for i := 0; i < cHidden; i++ {
		ar[i] = sigmoid.Sigmoid(-pnn.rBetaHidden * ar[i])
	}

	/* Calculate activity at output nodes */
	prWeight = pnn.arOutputWeight

	for i := 0; i < pnn.cOutput; i++ {
		r := pnn.arOutputThreshold[i]

		for j := 0; j < cHidden; j++ {
			r += ar[j] * prWeight[0]
			prWeight = prWeight[1:]
		}
		arOutput[i] = sigmoid.Sigmoid(-pnn.rBetaOutput * r)
	}
}
