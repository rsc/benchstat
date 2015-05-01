// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// Linspace returns num values spaced evenly between lo and hi,
// inclusive.
func Linspace(lo, hi float64, num int) []float64 {
	res := make([]float64, num)
	for i := 0; i < num; i++ {
		res[i] = lo + float64(i)*(hi-lo)/float64(num-1)
	}
	return res
}

// Logspace returns num values spaced evenly on a logarithmic scale
// between base**lo and base**hi, inclusive.
func Logspace(lo, hi float64, num int, base float64) []float64 {
	res := Linspace(lo, hi, num)
	for i, x := range res {
		res[i] = math.Pow(base, x)
	}
	return res
}

// Floors returns the floor of each element in xs.
func Floors(xs []float64) []int {
	res := make([]int, len(xs))
	for i, x := range xs {
		res[i] = int(x)
	}
	return res
}
