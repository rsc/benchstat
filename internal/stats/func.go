// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

// Func represents a continuous function.
type Func interface {
	// At returns the value of this Func at x.
	At(x float64) float64

	// AtEach returns the value of this Func at each x in xs.
	AtEach(xs []float64) []float64

	// Bounds returns reasonable bounds for plotting this Func.
	//
	// If this Func is a PDF, the total weight outside of these
	// bounds should be approximately 0.  If this Func is a CDF,
	// the value at the lower bound should be approximately 0 and
	// the value at the upper bound should be approximately 1.
	Bounds() (low float64, high float64)

	// Integrate returns a Func f where f.At(x) is the integral of
	// this function from -inf to x.
	//
	// If Integrate is not implemented, it should return nil.
	Integrate() Func
}

// atEach is a generic implementation of Func.AtEach.
func atEach(f Func, xs []float64) []float64 {
	// TODO(austin) Parallelize
	res := make([]float64, len(xs))
	for i, x := range xs {
		res[i] = f.At(x)
	}
	return res
}
