// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

// Miscellaneous helper algorithms

import (
	"fmt"
	"math"
)

// sign returns the sign of x: -1 if x < 0, 0 if x == 0, 1 if x > 0.
// If x is NaN, it returns NaN.
func sign(x float64) float64 {
	if x == 0 {
		return 0
	} else if x < 0 {
		return -1
	} else if x > 0 {
		return 1
	}
	return nan
}

func maxint(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minint(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sumint(xs []int) int {
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return sum
}

// lchoose returns math.Log(choose(n, k)).
func lchoose(n, k int) float64 {
	a, _ := math.Lgamma(float64(n + 1))
	b, _ := math.Lgamma(float64(k + 1))
	c, _ := math.Lgamma(float64(n - k + 1))
	return a - b - c
}

// choose returns the binomial coefficient of n and k.
func choose(n, k int) int {
	return int(math.Exp(lchoose(n, k)) + 0.5)
}

// atEach returns f(x) for each x in xs.
func atEach(f func(float64) float64, xs []float64) []float64 {
	// TODO(austin) Parallelize
	res := make([]float64, len(xs))
	for i, x := range xs {
		res[i] = f(x)
	}
	return res
}

// bisect returns an x in [low, high] such that |f(x)| <= tolerance
// using the bisection method.
//
// f(low) and f(high) must have opposite signs.
//
// If f does not have a root in this interval (e.g., it is
// discontiguous), this returns the X of the apparent discontinuity
// and false.
func bisect(f func(float64) float64, low, high, tolerance float64) (float64, bool) {
	flow, fhigh := f(low), f(high)
	if -tolerance <= flow && flow <= tolerance {
		return low, true
	}
	if -tolerance <= fhigh && fhigh <= tolerance {
		return high, true
	}
	if sign(flow) == sign(fhigh) {
		panic(fmt.Sprintf("root of f is not bracketed by [low, high]; f(%g)=%g f(%g)=%g", low, flow, high, fhigh))
	}
	for {
		mid := (high + low) / 2
		fmid := f(mid)
		if -tolerance <= fmid && fmid <= tolerance {
			return mid, true
		}
		if mid == high || mid == low {
			return mid, false
		}
		if sign(fmid) == sign(flow) {
			low = mid
			flow = fmid
		} else {
			high = mid
			fhigh = fmid
		}
	}
}

// series returns the sum of the series f(0), f(1), ...
//
// This implementation is fast, but subject to round-off error.
func series(f func(float64) float64) float64 {
	y, yp := 0.0, 1.0
	for n := 0.0; y != yp; n++ {
		yp = y
		y += f(n)
	}
	return y
}
