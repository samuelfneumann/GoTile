// Package gotile implements tile coding of vectors
package gotile

import (
	"fmt"
	"sync"

	"gonum.org/v1/gonum/mat"
)

// TileCoder implements functionality for tile coding a vector. Tile
// coding takes a low-dimensional vector and changes it into a large,
// sparse vector consisting of only 0's and 1's. Each 1 represents the
// coordinates of the original vector in some space of tilings. For
// example:
//
//		[0.5, 0.1] -> [0, 0, 0, 1, 0, 0, 1, 0]
//
//
// The number of nonzero elements in the tile-coded representation equals
// the number of tilings used to encode the vector. The number of total
// features in the tile-coded representation is the number of tilings
// times the number of tiles per tiling. Tile coding requires that the
// space to be tiled be bounded.
//
// This implementation of tile coding uses dense tilings over the entire
// state space. That is, each dimension of state space is fully tiled,
// and hash-based tile coding is not used. This implementation also
// uses multiple tilings, each of which consist of the name number
// of tiles per tiling.
type TileCoder struct {
	// numTilings int
	// minDims     mat.Vector
	// offsets     []*mat.Dense
	// bins        [][]int
	// binLengths  [][]float64
	// seed        uint64
	tilings     []*Tiling
	includeBias bool

	// Concurrent encoding parameters
	wait     sync.WaitGroup
	indices  chan int
	vIndices chan *mat.VecDense
}

// NewTileCoder creates and returns a new TileCoder struct. The minDims
// and maxDims arguments are the bounds on each dimension between which
// tilings will be placed. These arguments should have the same shape
// as vectors which will be tile coded.
//
// The bins argument determines both the number of tilings to use and
// the number of tiles per each tiling. This parameter is a [][]int.
// The number of elements in the outer slice determines the number of
// tilings to use. The sub-slices determine how many tiles are placed
// along each dimension for the respective tiling. For example, if
// bins := [][]int{{2, 2}, {4, 3}}, then the TileCoder uses two tilings.
// The first tiling is a 2x2 tiling. The second tiling uses 4 tiles
// along the first dimension and 3 tiles along the second dimension.
// The number of tiles along each dimension should equal the length of
// the minDims and maxDims parameters. That is, len(bins[i]) ==
// minDims.Len() == maxDims.Len() for any i in [0, len(bins)-1].
//
//The parameter includeBias determines whether or not a
// bias unit is kept as the first unit in the tile coded representation.
func New(minDims, maxDims mat.Vector, bins [][]int,
	seed uint64, includeBias bool) (*TileCoder, error) {
	numTilings := len(bins)
	tilings := make([]*Tiling, numTilings)
	var err error
	for tiling := range bins {
		tilings[tiling], err = NewTiling(minDims, maxDims, bins[tiling], seed)
		if err != nil {
			return nil, fmt.Errorf("new: could not create tiling %v: %v",
				tiling, err)
		}
	}

	// Channel along which encoded indices are sent
	indices := make(chan int, numTilings)
	vIndices := make(chan *mat.VecDense, numTilings)
	return &TileCoder{tilings, includeBias, sync.WaitGroup{}, indices,
		vIndices}, nil
}

// EncodeIndicesBatch returns a matrix of the non-zero indices in the
// tile coded batch of vectors when the batch is tile coded with the
// receiver. Similarly to EncodeBatch, it is assumed that each column
// of b is a specific vector in the sample to tile code, and each row
// is a specific feature for each vector in the batch. The returned
// matrix is of the same form, where column i of the returned matrix
// refers to the indices of non-zero elements in the tile coded
// representation of column i in b. The returned matrix is of the size
// k x c, where k is the number of non-zero indices (tilings + bias
// unit) and c is the number of samples in the batch (the number of
// columns in the input matrix).
func (t *TileCoder) EncodeIndicesBatch(b *mat.Dense) *mat.Dense {
	// Check if using a bias unit
	bias := 0
	if t.includeBias {
		bias = 1
	}

	// Create the slice of non-zero indices
	indices := make([]*mat.VecDense, t.NumTilings()+bias)
	// indices := make([]*mat.VecDense, t.numTilings+bias)

	// Listen on the vIndices channel for indices to set non-zero
	t.wait.Add(1)
	go func() {
		for i := 0; i < t.NumTilings(); i++ {
			index := <-t.vIndices
			indices[i] = index
		}
		t.wait.Done()
	}()

	// Concurrently calculate the non-zero indices for each tiling
	t.wait.Add(t.NumTilings())
	for i := 0; i < t.NumTilings(); i++ {
		go func(tiling int) {
			t.vIndices <- t.encodeBatchWithTiling(b, tiling)
			t.wait.Done()
		}(i)
	}

	// If using a bias unit, add its index to the list of non-zero indices
	if t.includeBias {
		_, batchSize := b.Dims()
		indices[len(indices)-1] = mat.NewVecDense(batchSize, nil)
	}

	// Ensure all goroutines have finished adding non-zero indices to
	// the indices slice before returning
	t.wait.Wait()

	out := mat.NewDense(len(indices), indices[0].Len(), nil)
	for row := 0; row < len(indices); row++ {
		out.SetRow(row, indices[row].RawVector().Data)
	}

	return out
}

// EncodeIndices returns a slice of the non-zero indices in the tile
// coded vector when v is tile coded with the receiving TileCoder t.
func (t *TileCoder) EncodeIndices(v mat.Vector) []float64 {
	// Check if using a bias unit
	bias := 0
	if t.includeBias {
		bias = 1
	}

	// Create the slice of non-zero indices
	indices := make([]float64, t.NumTilings()+bias)

	// Listen on the indices channel for indices to set non-zero
	t.wait.Add(1)
	go func() {
		for i := 0; i < t.NumTilings(); i++ {
			index := float64(<-t.indices)
			indices[i] = index
		}
		t.wait.Done()
	}()

	// Concurrently calculate the non-zero indices for each tiling
	t.wait.Add(t.NumTilings())
	for i := 0; i < t.NumTilings(); i++ {
		go func(tiling int) {
			t.indices <- t.encodeWithTiling(v, tiling)
			// t.indices <- t.tilings[tiling].index(v)
			t.wait.Done()
		}(i)
	}

	// If using a bias unit, add its index to the list of non-zero indices
	if t.includeBias {
		indices[len(indices)-1] = 0.0
	}

	// Ensure all goroutines have finished adding non-zero indices to
	// the indices slice before returning
	t.wait.Wait()
	return indices
}

// EncodeBatch encodes a batch of vectors held in a Dense matrix. In
// this batch, each row should be a sequential feature, while each
// column should be a sequential sample in the batch. This function
// returns a new matrix which holds the tile coded representation of
// each vector in the batch. The returned matrix is of the size
// k x c, where k is the number of features in the tile coded
// representation and c is the number of samples in the batch (the
// number of columns in the input matrix).
func (t *TileCoder) EncodeBatch(b *mat.Dense) *mat.Dense {
	_, batchSize := b.Dims()
	tileCoded := mat.NewDense(t.VecLength(), batchSize, nil)

	indices := t.EncodeIndicesBatch(b)
	numIndices, _ := indices.Dims()
	for row := 0; row < numIndices; row++ {
		colIndices := indices.RawRowView(row)
		for i := range colIndices {
			tileCoded.Set(int(colIndices[i]), i, 1.0)
		}
	}

	return tileCoded
}

// Encode encodes a single vector as a tile-coded vector
func (t *TileCoder) Encode(v mat.Vector) *mat.VecDense {
	tileCoded := mat.NewVecDense(t.VecLength(), nil)

	for _, index := range t.EncodeIndices(v) {
		tileCoded.SetVec(int(index), 1.0)
	}
	return tileCoded
}

// ToVector converts a vector of non-zero indices to a tile-coded
// vector
func (t *TileCoder) ToVector(v mat.Vector) *mat.VecDense {
	tileCoded := mat.NewVecDense(t.VecLength(), nil)
	for i := 0; i < v.Len(); i++ {
		tileCoded.SetVec(int(v.AtVec(i)), 1.0)
	}
	return tileCoded
}

// ToIndices converts a tile-coded vector to a vector of non-zero
// indices
func (t *TileCoder) ToIndices(v mat.Vector) *mat.VecDense {
	indices := make([]float64, 0, t.NumTilings())
	for i := 0; i < v.Len(); i++ {
		if v.AtVec(i) != 0.0 {
			indices = append(indices, float64(i))
		} else if v.AtVec(i) != 1.0 {
			panic("toIndices: vector is not a tile-coded vector")
		}
	}
	return mat.NewVecDense(t.NumTilings(), indices)
}

// String returns a string representation of a *TileCoder
func (t *TileCoder) String() string {
	bins := make([][]int, t.NumTilings())
	for i := 0; i < t.NumTilings(); i++ {
		bins[i] = t.tilings[i].bins
	}
	return fmt.Sprintf("Tilings %d  |  Tiles: %v", t.NumTilings(), bins)
}

// VecLength returns the number of features in a tile-coded vector
func (t *TileCoder) VecLength() int {
	baseVec := 0
	for i := 0; i < t.NumTilings(); i++ {
		// baseVec += prod(t.bins[i])
		baseVec += t.tilings[i].Tiles()
	}
	if t.includeBias {
		return baseVec + 1
	}
	return baseVec
}

// NumTilings returns the number of tilings the tile coder uses for
// encoding vectors
func (t *TileCoder) NumTilings() int {
	return len(t.tilings)
}

// prod calculates the product of all integers in a []int
func prod(i []int) int {
	prod := 1
	for _, v := range i {
		prod *= v
	}
	return prod
}

// Calculates how many features exist in the tile-coded representation
// before tiling number i
func (t *TileCoder) featuresBeforeTiling(i int) int {
	features := 0
	for j := 0; j < i; j++ {
		// features += prod(t.bins[j])
		features += t.tilings[j].Tiles()
	}
	return features
}

// encodeWithTiling returns the index of the tile coded feature vector
// which should be a 1.0 when the input vector v is encoded with tiling
// number tiling in the TileCoder.
func (t *TileCoder) encodeWithTiling(v mat.Vector, tiling int) int {
	bias := 0
	if t.includeBias {
		bias = 1
	}

	// indexOffset is the index into the tile-coded vector at which
	// the current tiling will start
	indexOffset := t.featuresBeforeTiling(tiling)
	index := t.tilings[tiling].Index(v)

	return indexOffset + index + bias
}

// encodeBatchWithTiling returns the indices of the tile coded feature
// vectors which should be a 1.0 when the input batch of vectors b is
// encoded with tiling number tiling. The indices for the vector at
// column i in b are also in column i in the returned matrix. Each
// column of b is considered a vector to tile code, while each row
// is considered a feature for each vector in the batch.
func (t *TileCoder) encodeBatchWithTiling(b *mat.Dense,
	tiling int) *mat.VecDense {
	// Check if using a bias unit, if so we will need to offset the
	// indices generated later
	bias := 0.
	if t.includeBias {
		bias = 1.
	}

	indexOffset := t.featuresBeforeTiling(tiling)
	index := t.tilings[tiling].IndexBatch(b)

	// Offset the 1.0 based on which tiling was used for the previous
	// iteration of coding and if a bias unit was used
	// A vector of 1.0's will be needed for calculations later
	rows, _ := b.Dims()
	ones := VecOnes(rows)
	index.AddScaledVec(index, float64(indexOffset)+bias, ones)

	return index
}
