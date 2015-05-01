// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// A TDist is a Student's t-distribution with V degrees of freedom.
type TDist struct {
	V float64
}

func lgamma(x float64) float64 {
	y, _ := math.Lgamma(x)
	return y
}

// beta returns the value of the complete beta function B(a, b).
func beta(a, b float64) float64 {
	// B(x,y) = Γ(x)Γ(y) / Γ(x+y)
	return math.Exp(lgamma(a) + lgamma(b) - lgamma(a+b))
}

// betainc returns the value of the regularized incomplete beta
// function Iₓ(a, b).
//
// Note that the "incomplete beta function" can be computed as
// betainc(x, a, b)*beta(a, b).
func betainc(x, a, b float64) float64 {
	// Based on Numerical Recipes in C, section 6.4. This uses the
	// continued fraction definition of I:
	//
	//  (xᵃ*(1-x)ᵇ)/(a*B(a,b)) * (1/(1+(d₁/(1+(d₂/(1+...))))))
	//
	// where B(a,b) is the beta function and
	//
	//  d_{2m+1} = -(a+m)(a+b+m)x/((a+2m)(a+2m+1))
	//  d_{2m}   = m(b-m)x/((a+2m-1)(a+2m))
	if x < 0 || x > 1 {
		panic("betainc: x must be in [0, 1]")
	}
	bt := 0.0
	if 0 < x && x < 1 {
		// Compute the coefficient before the continued
		// fraction.
		bt = math.Exp(lgamma(a+b) - lgamma(a) - lgamma(b) +
			a*math.Log(x) + b*math.Log(1-x))
	}
	if x < (a+1)/(a+b+2) {
		// Compute continued fraction directly.
		return bt * betacf(x, a, b) / a
	} else {
		// Compute continued fraction after symmetry transform.
		return 1 - bt*betacf(1-x, b, a)/b
	}
}

// betacf is the continued fraction component of the regularized
// incomplete beta function Iₓ(a, b).
func betacf(x, a, b float64) float64 {
	const maxIterations = 200
	const epsilon = 3e-14

	raiseZero := func(z float64) float64 {
		if math.Abs(z) < math.SmallestNonzeroFloat64 {
			return math.SmallestNonzeroFloat64
		}
		return z
	}

	c := 1.0
	d := 1 / raiseZero(1-(a+b)*x/(a+1))
	h := d
	for m := 1; m <= maxIterations; m++ {
		mf := float64(m)

		// Even step of the recurrence.
		numer := mf * (b - mf) * x / ((a + 2*mf - 1) * (a + 2*mf))
		d = 1 / raiseZero(1+numer*d)
		c = raiseZero(1 + numer/c)
		h *= d * c

		// Odd step of the recurrence.
		numer = -(a + mf) * (a + b + mf) * x / ((a + 2*mf) * (a + 2*mf + 1))
		d = 1 / raiseZero(1+numer*d)
		c = raiseZero(1 + numer/c)
		hfac := d * c
		h *= hfac

		if math.Abs(hfac-1) < epsilon {
			return h
		}
	}
	panic("betainc: a or b too big; failed to converge")
}

func (t TDist) At(x float64) float64 {
	return math.Exp(lgamma((t.V+1)/2)-lgamma(t.V/2)) /
		math.Sqrt(t.V*math.Pi) * math.Pow(1+(x*x)/t.V, -(t.V+1)/2)
}

func (t TDist) AtEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	factor := math.Exp(lgamma((t.V+1)/2)-lgamma(t.V/2)) /
		math.Sqrt(t.V*math.Pi)
	for i, x := range xs {
		res[i] = factor / math.Pow(1+(x*x)/t.V, (t.V+1)/2)
	}
	return res
}

func (t TDist) Bounds() (float64, float64) {
	return -4, 4
}

func (t TDist) Integrate() Func {
	return tCDF{t.V}
}

type tCDF struct {
	V float64
}

func (ti tCDF) At(x float64) float64 {
	if x == 0 {
		return 0.5
	} else if x > 0 {
		return 1 - 0.5*betainc(ti.V/(ti.V+x*x), ti.V/2, 0.5)
	} else if x < 0 {
		return 1 - ti.At(-x)
	} else {
		return math.NaN()
	}
}

func (ti tCDF) AtEach(xs []float64) []float64 {
	return atEach(ti, xs)
}

func (ti tCDF) Bounds() (float64, float64) {
	return -4, 4
}

func (ti tCDF) Integrate() Func {
	return nil
}
