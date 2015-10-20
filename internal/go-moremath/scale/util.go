// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scale

// clamp clamps x to the range [0, 1].
func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// autoScale returns the smallest m for which fn(m) <= n. This is
// intended to be used for auto-scaling tick values, where fn maps
// from a tick "level" to the number of ticks at that level in the
// scale's input range.
//
// fn must be a monotonically decreasing function.
func autoScale(n int, fn func(level int) int, guess int) int {
	m := guess
	if fn(m) <= n {
		for m--; fn(m) <= n; m-- {
		}
		return m + 1
	} else {
		for m++; fn(m) > n; m++ {
		}
		return m
	}
}
