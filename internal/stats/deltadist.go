// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

// Delta is the Dirac delta function, centered at T, with total area
// 1.
//
// The CDF of the Dirac delta function is the Heaviside step function,
// centered at T. Specifically, f(T) == 1.
type Delta struct {
	T float64
}

func (d Delta) PDF(x float64) float64 {
	if x == d.T {
		return inf
	}
	return 0
}

func (d Delta) PDFEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	for i, x := range xs {
		if x == d.T {
			res[i] = inf
		}
	}
	return res
}

func (d Delta) CDF(x float64) float64 {
	if x >= d.T {
		return 1
	}
	return 0
}

func (d Delta) CDFEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	for i, x := range xs {
		res[i] = d.CDF(x)
	}
	return res
}

func (d Delta) InvCDF(y float64) float64 {
	if y < 0 || y > 1 {
		return nan
	}
	return d.T
}

func (d Delta) InvCDFEach(ys []float64) []float64 {
	res := make([]float64, len(ys))
	for i, y := range ys {
		res[i] = d.InvCDF(y)
	}
	return res
}

func (d Delta) Bounds() (float64, float64) {
	return d.T - 1, d.T + 1
}
