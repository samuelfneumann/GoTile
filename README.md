# GoTile
Tile coding vectors in Go

GoTile is a Go module for tile coding vectors in Go. The `TileCoder` 
struct is the workhorse of the module. It stores `Tiling`s and uses these 
tilings to encode a vector into some one-hot representation.
Some characteristics of the module:

* `Tiling`s in a `TileCoder` need not have the same number of bins per dimension. For example, one `Tiling` may have 3 bins in dimension `x` while another has 30 bins in dimension `x`. 

* The `TileCoder` can return either a one-hot encoded version of an input vector or only the non-zero indices of the input vector.

* Batch tile-coding is implemented efficiently. You can tile code a whole matrix, where each column is assumed to be a consecutive vector to tile code.

* Each `Tiling` in a `TileCoder` encodes input vectors concurrently. For example, if you have 100 `Tiling`s in a `TileCoder` and you call `Encode()` or `EncodeBatch()`, this will spawn 100 goroutines and each `Tiling` encodes the input vector(s) concurrently (there will be one goroutine per `Tiling`).

