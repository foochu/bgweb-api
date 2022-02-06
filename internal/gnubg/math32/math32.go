package math32

import (
	"math"
	"math/rand"
)

func Sqrtf(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

func Erff(x float32) float32 {
	return float32(math.Erf(float64(x)))
}

func Fabsf(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func Logf(x float32) float32 {
	return float32(math.Log(float64(x)))
}

func Min(x float32, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func Imin(x int, y int) int {
	return int(math.Min(float64(x), float64(y)))
}

func Max(x float32, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func Irand() int {
	return rand.Int()
}
