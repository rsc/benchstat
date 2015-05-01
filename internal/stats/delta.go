// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// Delta is the Dirac delta function, centered at T.
type Delta struct {
	T float64
}

func (d Delta) At(x float64) float64 {
	if x == d.T {
		return math.Inf(1)
	}
	return 0
}

func (d Delta) AtEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	inf := math.Inf(1)
	for i, x := range xs {
		if x == d.T {
			res[i] = inf
		}
	}
	return res
}

func (d Delta) Bounds() (float64, float64) {
	return d.T - 1, d.T + 1
}

func (d Delta) Integrate() Func {
	return UnitStep{d.T}
}

// UnitStep is the Heaviside step function, centered at T.  This uses
// the convention that f(T) == 1, so it is the cumulative distribution
// function of the delta function.
type UnitStep struct {
	T float64
}

func (u UnitStep) At(x float64) float64 {
	if x >= u.T {
		return 1
	}
	return 0
}

func (u UnitStep) AtEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	for i, x := range xs {
		if x >= u.T {
			res[i] = 1
		}
	}
	return res
}

func (u UnitStep) Bounds() (float64, float64) {
	return u.T - 1, u.T + 1
}

func (u UnitStep) Integrate() Func {
	return nil
}
