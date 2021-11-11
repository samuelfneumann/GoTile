package gotile

import (
	"fmt"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// Format formats a matrix for printing
func Format(X mat.Matrix) string {
	fa := mat.Formatted(X, mat.Prefix(""), mat.Squeeze())
	return fmt.Sprintf("%v", fa)
}

func TestBatch(t *testing.T) {
	batch := mat.NewDense(2, 2, []float64{1., 2., 3., 4.})
	v1 := mat.NewVecDense(2, []float64{1, 3})
	v2 := mat.NewVecDense(2, []float64{2, 4})

	minDims := mat.NewVecDense(2, []float64{0, 0})
	maxDims := mat.NewVecDense(2, []float64{5, 5})

	coder := New(minDims, maxDims, [][]int{{2, 3}, {2, 2}}, 1, true)

	indices := coder.EncodeIndicesBatch(batch)
	fmt.Println(Format(batch))
	fmt.Println(Format(indices))
	fmt.Println()

	v1tc := coder.Encode(v1)
	v2tc := coder.Encode(v2)
	v1Indices := coder.EncodeIndices(v1)
	v2Indices := coder.EncodeIndices(v2)
	fmt.Println(v1)
	fmt.Println(v1tc)
	fmt.Println(v1Indices)
	fmt.Println()
	fmt.Println(v2)
	fmt.Println(v2tc)
	fmt.Println(v2Indices)
}

func BenchmarkTileCoder(b *testing.B) {
	tc := New(
		mat.NewVecDense(8, []float64{0, 0, 0, 0, 0, 0, 0, 0}),
		mat.NewVecDense(8, []float64{1, 1, 1, 1, 1, 1, 1, 1}),
		[][]int{{8, 8, 8, 8, 8, 8, 8, 8}},
		12,
		true,
	)

	y := mat.NewVecDense(8, []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5})

	for i := 0; i < b.N; i++ {
		tc.Encode(y)
	}
}
