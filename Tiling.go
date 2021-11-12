package gotile

import (
	"fmt"
	"math"

	"github.com/samuelfneumann/goutils/floatutils"
	"github.com/samuelfneumann/goutils/matutils"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/spatial/r1"
	"gonum.org/v1/gonum/stat/distmv"
	"gonum.org/v1/gonum/stat/samplemv"
)

// Default offset divisor. See NewTiling for more details.
const OffsetDiv float64 = 1.5

// Tiling is a grid of tiles over some space in ℝ^n
type Tiling struct {
	offsets    *mat.Dense // Offset of the tiling along each dimension
	bins       []int      // Number of bins along each dimension
	binLengths []float64  // Length of bins along each dimension
	minDims    mat.Vector
	seed       uint64
}

// NewTiling returns a new tiling from minDims to maxDims along each
// dimension. The tiling will have bins[i] bins along dimension i,
// and each dimension can have a different number of bins. The
// offset from the origin in each dimension of the tiling is controlled
// by offsetDiv.
//
// For each dimension, tilings are offset from
// the origin by randomly sampling from a uniform distribution with
// support [-tiling width/OffsetDiv, tiling width/OffsetDiv]^k, where
// k is the number of dimension of the tiling or state space. Each
// dimension of the tiling may be offset from the origin by a different
// amount.
func NewTiling(minDims, maxDims mat.Vector, bins []int,
	seed uint64, offsetDiv float64) (*Tiling, error) {
	// Error checking
	if minDims.Len() != maxDims.Len() {
		msg := fmt.Sprintf("newTiing: cannot specify minimum with fewer "+
			"dimensions than maximum: %d != %d", minDims.Len(), maxDims.Len())
		return nil, fmt.Errorf(msg)
	}
	if len(bins) == 0 {
		msg := "newTiling: cannot have less than 1 bin per dimension"
		return nil, fmt.Errorf(msg)
	}
	if len(bins) != minDims.Len() {
		msg := fmt.Sprintf("newTiling: there should be a single number of bins for "+
			"each dimension: \n\thave(%d) \n\twant (%d)", len(bins),
			minDims.Len())
		return nil, fmt.Errorf(msg)
	}

	// Calculate the length of bins and the Tiling offset bounds
	var bounds []r1.Interval

	TilingBinLengths := make([]float64, minDims.Len())
	binLengths := TilingBinLengths

	for i := 0; i < minDims.Len(); i++ {
		// Calculate the length of bins
		binLength := (maxDims.AtVec(i) - minDims.AtVec(i))
		binLength /= float64(bins[i])
		bound := binLength / OffsetDiv // Bounds Tiling offsets

		binLengths[i] = binLength
		bounds = append(bounds, r1.Interval{Min: -bound, Max: bound})
	}

	// Create RNG for uniform sampling of Tiling offsets
	source := rand.NewSource(seed)
	u := distmv.NewUniform(bounds, source)
	sampler := samplemv.IID{Dist: u}

	// Calculate offsets
	offsets := mat.NewDense(1, len(bounds), nil)
	sampler.Sample(offsets)

	return &Tiling{offsets, bins, binLengths, minDims, seed}, nil
}

// Index will return the index of the tile within which v falls
func (t *Tiling) Index(v mat.Vector) int {
	index := 0

	// Tile code the vector based on the current Tiling
	// We loop through each feature to calculate the tile index to
	// set to 1.0 along this feature dimension
	for i := len(t.bins) - 1; i > -1; i-- {
		// Offset the Tiling
		data := v.AtVec(i) + t.offsets.At(0, i)

		// Calculate the index of the tile along the current feature
		// dimension in which the feature falls
		tile := math.Floor((data - t.minDims.AtVec(i)) / t.binLengths[i])

		// Clip tile to within Tiling bounds
		tile = floatutils.Clip(tile, 0.0, float64(t.bins[i]-1))

		// Calculate the index into the tile-coded representation
		// that should be 1.0 for this Tiling
		tileIndex := int(tile)
		if i == len(t.bins)-1 {
			index += tileIndex
		} else {
			index += tileIndex * t.bins[i+1]
		}
	}
	return index
}

// IndexBatch returns the indices within which each vector in a batch
// of vectors falls. The batch of vectors b should be such that each
// columns is a vector to tile code, and each row corresponds to a
// single feature for each vector in the batch. That is, b should
// be of the following form:
//
//		v⃗		 =	 [v⃗_1  v⃗_2  ...  v⃗_n]
//			    =	⎡v_11  v_21  ...  v_n1 ⎤
//			 		⎢v_12  v_22  ...  v_n2 ⎥
//					⎢...   ...   ...  ...  ⎥
//					⎣v_1m  v_2m  ...  v_nm ⎦
//		v⃗_i	 =	 sample/vector i in the batch
//		v_ij	=	coordinate/feature j of sample vector i
func (t *Tiling) IndexBatch(b *mat.Dense) *mat.VecDense {
	rows, _ := b.Dims()

	// A vector of 1.0's will be needed for calculations later
	ones := matutils.VecOnes(rows)

	data := mat.NewVecDense(rows, nil)

	index := mat.NewVecDense(rows, nil)

	for i := len(t.bins) - 1; i > -1; i-- {
		// Clone the next batch of features into the data vector
		data.CloneFromVec(b.RowView(i))

		// Offset the Tiling
		data.AddScaledVec(data, t.offsets.At(0, i), ones)

		// Calculate which tile each feature is in along the current
		// dimension. Subtracting the minimum dimension will ensure that
		// the data is between [0, 1] before multiplying by the bin
		// length in VecFloor. The integer value of this * binLength
		// is the tile along the current dimension that the feature is in:
		//
		// binLengths[i] = max - min / binLength
		// (data - min) / ((max - min) / binLength) =
		// = ((data - min) / (max - min)) * binLength = IND
		// int(IND) == index into Tiling along current dimension
		data.AddScaledVec(data, -t.minDims.AtVec(i), ones)
		matutils.VecFloor(data, t.binLengths[i])

		// If out-of-bounds, use the last tile
		matutils.VecClip(data, 0.0, float64(t.bins[i]-1))

		// Calculate the index into the tile-coded representation
		// that should be 1.0 for this Tiling
		if i == len(t.bins)-1 {
			index.AddVec(index, data)
		} else {
			index.AddScaledVec(index, float64(t.bins[i+1]), data)
		}
	}

	return index
}

// Tiles returns the number of tiles in the tiling
func (t *Tiling) Tiles() int {
	return prod(t.bins)
}
