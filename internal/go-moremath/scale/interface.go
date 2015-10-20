// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scale

// A Quantative scale is an invertible function from some continuous
// input range to an output domain of [0, 1].
type Quantitative interface {
	// Map maps from a value x in the input range to [0, 1]. If x
	// is outside the input range and clamping is enabled, x will
	// first be clamped to the input range.
	Map(x float64) float64

	// Unmap is the inverse of Map. That is, if x is in the input
	// range or clamping is disabled, x = Unmap(Map(x)). If
	// clamping is enabled and y is outside [0,1], the results are
	// undefined.
	Unmap(y float64) float64

	// SetClamp sets the clamping mode of this scale.
	SetClamp(bool)

	// Ticks returns a set of at most n major ticks, plus minor
	// ticks. These ticks will have "nice" values within the input
	// range. Both arrays are sorted in ascending order and minor
	// includes ticks in major.
	Ticks(n int) (major, minor []float64)

	// Nice expands the input range of this scale to "nice" values
	// for covering the input range with n major ticks. After
	// calling Nice(n), the first and last major ticks returned by
	// Ticks(n) will equal the lower and upper bounds of the input
	// range.
	Nice(n int)
}

// A QQ maps from a source Quantitative scale to a destination
// Quantitative scale.
type QQ struct {
	Src, Dest Quantitative
}

// Map maps from a value x in the source scale's input range to a
// value y in the destination scale's input range.
func (q QQ) Map(x float64) float64 {
	return q.Dest.Unmap(q.Src.Map(x))
}

// Unmap maps from a value y in the destination scale's input range to
// a value x in the source scale's input range.
func (q QQ) Unmap(x float64) float64 {
	return q.Src.Unmap(q.Dest.Map(x))
}
