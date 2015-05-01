// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"testing"
)

func TestStdNormal(t *testing.T) {
	d := StdNormal
	if e, g := 1/math.Sqrt(2*math.Pi), d.At(0); !aeq(e, g) {
		t.Errorf("bad value at 0: expected %g, got %g", e, g)
	}
	if e, g := 1/math.Sqrt(2*math.Pi)*math.Exp(-0.5), d.At(1); !aeq(e, g) {
		t.Errorf("bad value at 1: expected %g, got %g", e, g)
	}
	if e, g := 1/math.Sqrt(2*math.Pi)*math.Exp(-0.5), d.At(-1); !aeq(e, g) {
		t.Errorf("bad value at -1: expected %g, got %g", e, g)
	}
	if e, g := 0.0, d.At(-10000); !aeq(e, g) {
		t.Errorf("bad value at low tail: expected %g, got %g", e, g)
	}
	if e, g := 0.0, d.At(10000); !aeq(e, g) {
		t.Errorf("bad value at high tail: expected %g, got %g", e, g)
	}
}

func TestStdNormalIntegral(t *testing.T) {
	d := StdNormal.Integrate()
	if e, g := 0.5, d.At(0); !aeq(e, g) {
		t.Errorf("bad value at 0: expected %g, got %g", e, g)
	}
	if e, g := 0.0, d.At(-10000); !aeq(e, g) {
		t.Errorf("bad value at low tail: expected %g, got %g", e, g)
	}
	if e, g := 1.0, d.At(10000); !aeq(e, g) {
		t.Errorf("bad value at high tail: expected %g, got %g", e, g)
	}
}
