// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "testing"

func TestOneSample(t *testing.T) {
	// Unweighted, fixed bandwidth
	x := float64(5)
	kde := KDE{Bandwidth: FixedBandwidth(1)}.FromSample(Sample{Xs: []float64{x}})
	if e, g := StdNormal.PDF(0), kde.PDF(x); !aeq(e, g) {
		t.Errorf("bad PDF value at sample: expected %g, got %g", e, g)
	}
	if e, g := 0.0, kde.PDF(-10000); !aeq(e, g) {
		t.Errorf("bad PDF value at low tail: expected %g, got %g", e, g)
	}
	if e, g := 0.0, kde.PDF(10000); !aeq(e, g) {
		t.Errorf("bad PDF value at high tail: expected %g, got %g", e, g)
	}

	if e, g := 0.5, kde.CDF(x); !aeq(e, g) {
		t.Errorf("bad CDF value at sample: expected %g, got %g", e, g)
	}
	if e, g := 0.0, kde.CDF(-10000); !aeq(e, g) {
		t.Errorf("bad CDF value at low tail: expected %g, got %g", e, g)
	}
	if e, g := 1.0, kde.CDF(10000); !aeq(e, g) {
		t.Errorf("bad CDF value at high tail: expected %g, got %g", e, g)
	}

	low, high := kde.Bounds()
	if e, g := x-2, low; e < g {
		t.Errorf("bad low bound: expected %g, got %g", e, g)
	}
	if e, g := x+2, high; e > g {
		t.Errorf("bad high bound: expected %g, got %g", e, g)
	}
}

func TestTwoSamples(t *testing.T) {
	kde := KDE{Bandwidth: FixedBandwidth(2)}.FromSample(Sample{Xs: []float64{1, 3}})
	testFunc(t, "PDF", kde.PDF, map[float64]float64{
		0: 0.120395730,
		1: 0.160228251,
		2: 0.176032663,
		3: 0.160228251,
		4: 0.120395730})

	testFunc(t, "CDF", kde.CDF, map[float64]float64{
		0: 0.187672369,
		1: 0.329327626,
		2: 0.5,
		3: 0.670672373,
		4: 0.812327630})
}
