package gotile

// import (
// 	"math"

// 	"gonum.org/v1/gonum/mat"
// )

// // VecFloor performs an element-wise floor division of a vector by some
// // constant b
// func VecFloor(a *mat.VecDense, b float64) {
// 	for i := 0; i < a.Len(); i++ {
// 		mod := math.Floor(a.AtVec(i) / b)
// 		a.SetVec(i, mod)
// 	}
// }

// // VecClip performs an element-wise clipping of a vector's values such
// // that each value is at least min and at most max
// func VecClip(a *mat.VecDense, min, max float64) {
// 	ClipSlice(a.RawVector().Data, min, max)
// }

// // VecOnes returns a vector of 1.0's
// func VecOnes(length int) *mat.VecDense {
// 	oneSlice := make([]float64, length)
// 	for i := 0; i < length; i++ {
// 		oneSlice[i] = 1.0
// 	}
// 	return mat.NewVecDense(length, oneSlice)
// }

// // Clip clips a float to be in [min, max]
// func Clip(f, min, max float64) float64 {
// 	if f < min {
// 		return min
// 	} else if f > max {
// 		return max
// 	}
// 	return f
// }

// // ClipSlice clips all elements of a slice in-place.
// func ClipSlice(slice []float64, min, max float64) []float64 {
// 	for i := range slice {
// 		slice[i] = Clip(slice[i], min, max)
// 	}
// 	return slice
// }
