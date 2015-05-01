// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "testing"

func TestTTest(t *testing.T) {
	s1 := Sample{Xs: []float64{2, 1, 3, 4}}
	s2 := Sample{Xs: []float64{6, 5, 7, 9}}

	check := func(want, got *TTestResult) {
		if !aeq(want.T, got.T) || !aeq(want.P, got.P) || !aeq(want.DoF, got.DoF) {
			t.Errorf("want %+v, got %+v", want, got)
		}
	}

	var r *TTestResult

	r, _ = TwoSampleTTest(s1, s1)
	check(&TTestResult{0, 1, 6}, r)
	r, _ = TwoSampleWelchTTest(s1, s1)
	check(&TTestResult{0, 1, 6}, r)

	r, _ = TwoSampleTTest(s1, s2)
	check(&TTestResult{-3.9703446152237674, 0.0073640592242113214, 6}, r)
	r, _ = TwoSampleWelchTTest(s1, s2)
	check(&TTestResult{-3.9703446152237674, 0.0085128631313781695, 5.584615384615385}, r)

	r, _ = PairedTTest(s1.Xs, s2.Xs, 0)
	check(&TTestResult{17, 0.00044334353831207749, 3}, r)

	r, _ = OneSampleTTest(s1, 0)
	check(&TTestResult{3.872983346207417, 0.030466291662170977, 3}, r)
	r, _ = OneSampleTTest(s1, 2.5)
	check(&TTestResult{0, 1, 3}, r)
}
