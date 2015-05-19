// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// Normal is a normal (Gaussian) distribution with mean Mu and
// standard deviation Sigma.
type Normal struct {
	Mu, Sigma float64
}

// StdNormal is the standard normal distribution (Mu = 0, Sigma = 1)
var StdNormal = Normal{0, 1}

// 1/sqrt(2 * pi)
const invSqrt2Pi = 0.39894228040143267793994605993438186847585863116493465766592583

func (n Normal) PDF(x float64) float64 {
	z := x - n.Mu
	return math.Exp(-z*z/(2*n.Sigma*n.Sigma)) * invSqrt2Pi / n.Sigma
}

func (n Normal) PDFEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	if n.Mu == 0 && n.Sigma == 1 {
		// Standard normal fast path
		for i, x := range xs {
			res[i] = math.Exp(-x*x/2) * invSqrt2Pi
		}
	} else {
		a := -1 / (2 * n.Sigma * n.Sigma)
		b := invSqrt2Pi / n.Sigma
		for i, x := range xs {
			z := x - n.Mu
			res[i] = math.Exp(z*z*a) * b
		}
	}
	return res
}

func (n Normal) CDF(x float64) float64 {
	return (1 + math.Erf((x-n.Mu)/(n.Sigma*math.Sqrt2))) / 2
}

func (n Normal) CDFEach(xs []float64) []float64 {
	res := make([]float64, len(xs))
	a := 1 / (n.Sigma * math.Sqrt2)
	for i, x := range xs {
		res[i] = (1 + math.Erf((x-n.Mu)*a)) / 2
	}
	return res
}

func (n Normal) InvCDF(y float64) float64 {
	panic("not implemented")
}

func (n Normal) InvCDFEach(ys []float64) []float64 {
	panic("not implemented")
}

func (n Normal) Bounds() (float64, float64) {
	const stddevs = 3
	return n.Mu - stddevs*n.Sigma, n.Mu + stddevs*n.Sigma
}
