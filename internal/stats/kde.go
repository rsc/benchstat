// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"fmt"
	"math"
)

// TODO: Consider moving this to stats/kde.  Then I could write things
// like kde.Setup{Bandwidth: kde.Scott}.FromSample(sample)

// TODO: Different bandwidth estimators need different inputs.  Can I
// unify this better?

// KDE represents options for constructing a kernel density estimate.
//
// Kernel density estimation is a method for constructing an estimate
// ƒ̂(x) of a unknown distribution ƒ(x) given a sample from that
// distribution.  Unlike many techniques, kernel density estimation is
// non-parametric: in general, it doesn't assume any particular true
// distribution (note, however, that the resulting distribution
// depends deeply on the selected bandwidth, and many bandwidth
// estimation techniques assume normal reference rules).
//
// A kernel density estimate is similar to a histogram, except that it
// is a smooth probability estimate and does not require choosing a
// bin size and discretizing the data.
//
// To construct a kernel density estimate, create an instance of KDE
// and then use the appropriate method to supply data.
//
// The default (zero) value of KDE is a reasonable default
// configuration.
type KDE struct {
	// Kernel is the kernel to use for the KDE.
	Kernel Kernel

	// Bandwidth is the bandwidth estimator to use for the KDE.
	//
	// If this is nil, the default bandwidth estimator is used.
	// Currently this is Scott, but this may change as better
	// estimators are implemented.
	Bandwidth BandwidthEstimator

	// BoundaryMethod is the boundary correction method to use for
	// the KDE.
	BoundaryMethod BoundaryMethod

	// [BoundaryMin, BoundaryMax) specify a bounded support for
	// the KDE.  This is ignored if BoundaryMethod is
	// BoundaryNone.
	//
	// To specify a half-bounded support, set Min to math.Inf(-1)
	// or Max to math.Inf(1).
	//
	// If these are both 0 (their default values), no boundary
	// correction is performed.
	BoundaryMin float64
	BoundaryMax float64
}

type BandwidthEstimator interface {
	// Bandwidth returns the bandwidth estimate for a sample.
	Bandwidth(s Sample) float64

	// HistBandwidth returns the bandwidth estimate for the
	// samples in hist.
	HistBandwidth(hist Histogram, ss *StreamStats) float64
}

// Silverman is a bandwidth estimator implementing Silverman's Rule of
// Thumb.  It's fast, but not very robust to outliers.
//
// Silverman, B. W. (1986) Density Estimation.
var Silverman silverman

type silverman struct{}

func (silverman) compute(stddev, count float64) float64 {
	return 1.06 * stddev * math.Pow(count, -1.0/5)
}

func (bw silverman) Bandwidth(s Sample) float64 {
	return bw.compute(s.StdDev(), s.Weight())
}

func (bw silverman) HistBandwidth(hist Histogram, ss *StreamStats) float64 {
	return bw.compute(ss.StdDev(), ss.Weight())
}

// Scott is a bandwidth estimator implementing Scott's Rule.  This is
// generally robust to outliers: it chooses the minimum between the
// sample's standard deviation and an robust estimator of a Gaussian
// distribution's standard deviation.
//
// Scott, D. W. (1992) Multivariate Density Estimation: Theory,
// Practice, and Visualization.
var Scott scott

type scott struct{}

func (scott) compute(stddev, iqr, count float64) float64 {
	hScale := 1.06 * math.Pow(count, -1.0/5)
	stdDev := stddev
	if stdDev < iqr/1.349 {
		// Use Silverman's Rule of Thumb
		return hScale * stdDev
	} else {
		// Use IQR/1.349 as a robust estimator of the standard
		// deviation of a Gaussian distribution.
		return hScale * (iqr / 1.349)
	}
}

func (bw scott) Bandwidth(s Sample) float64 {
	return bw.compute(s.StdDev(), s.IQR(), s.Weight())
}

func (bw scott) HistBandwidth(hist Histogram, ss *StreamStats) float64 {
	return bw.compute(ss.StdDev(), HistogramIQR(hist), ss.Weight())
}

// TODO(austin) Implement bandwidth estimator from Botev, Grotowski,
// Kroese. (2010) Kernel Density Estimation via Diffusion.

// FixedBandwidth is a bandwidth estimator that simply returns its
// value.
type FixedBandwidth float64

func (bw FixedBandwidth) Bandwidth(s Sample) float64 {
	return float64(bw)
}

func (bw FixedBandwidth) HistBandwidth(hist Histogram, ss *StreamStats) float64 {
	return float64(bw)
}

// Kernel represents a kernel to use for a KDE.
type Kernel int

//go:generate stringer -type=Kernel
const (
	GaussianKernel Kernel = iota

	// DeltaKernel is a Dirac delta function.  The PDF of such a
	// KDE is not well-defined, but the CDF will represent each
	// sample as an instantaneous increase.  This kernel ignores
	// bandwidth and never requires boundary correction.
	DeltaKernel
)

// BoundaryMethod represents a boundary correction method for
// constructing a KDE with bounded support.
type BoundaryMethod int

//go:generate stringer -type=BoundaryMethod
const (
	// BoundaryReflect reflects the density estimate at the
	// boundaries.  For example, for a KDE with support [0, inf),
	// this is equivalent to ƒ̂ᵣ(x)=ƒ̂(x)+ƒ̂(-x) for x>=0.  This is a
	// simple and fast technique, but enforces that ƒ̂ᵣ'(0)=0, so
	// it may not be applicable to all distributions.
	BoundaryReflect BoundaryMethod = iota

	// boundaryNone represents no boundary correction.
	//
	// This is used internally when the bounds are -/+inf.
	boundaryNone
)

// FromSample returns the probability density function of the kernel
// density estimate for s.
func (k KDE) FromSample(s Sample) Dist {
	if s.Weights != nil && len(s.Xs) != len(s.Weights) {
		panic("len(xs) != len(weights)")
	}

	// Compute bandwidth
	bw := k.Bandwidth
	if bw == nil {
		bw = Scott
	}
	h := bw.Bandwidth(s)

	// Construct kernel
	kernel := Dist(nil)
	switch k.Kernel {
	default:
		panic(fmt.Sprint("unknown kernel", k))
	case GaussianKernel:
		kernel = Normal{0, h}
	case DeltaKernel:
		kernel = Delta{0}
	}

	// Normalize boundaries
	bm := k.BoundaryMethod
	min, max := k.BoundaryMin, k.BoundaryMax
	if min == 0 && max == 0 {
		min, max = math.Inf(-1), math.Inf(1)
	}
	if math.IsInf(min, -1) && math.IsInf(max, 1) {
		bm = boundaryNone
	}

	return &kdeDist{kernel, s.Xs, s.Weights, bm, min, max}
}

// TODO: Instead of FromHistogram, make histogram able to create a
// weighted Sample and have a method that takes a sample and its
// statistics interface separately (or have the caller produce their
// own FixedBandwidth and expose the bandwidth estimators in terms of
// the statistics interfaces they each require).

// FromHistogram returns the probability density function of the kernel
// density estimate for hist.
//
// The returned KDE is necessarily approximate because of the
// histogram's bucketing of the samples.  However, as long as very few
// samples are outside the bounds of the histogram and the returned
// KDE is itself sampled at a coarser granularity than the granularity
// of the histogram, this approximation is quite good.  Assuming this
// approximation is sufficient, using a histogram to pre-process the
// data can significantly reduce the time and space required to
// construct a KDE.
//
// Note that the returned KDE may use the data from hist directly, so
// hist must not be modified until the caller is done with the KDE.
func (k KDE) FromHistogram(hist Histogram, ss *StreamStats) Dist {
	// Construct weighted samples from hist
	_, counts, _ := hist.Counts()
	xs, weights := make([]float64, len(counts)), make([]float64, len(counts))

	for bin, count := range counts {
		// Assume samples fall at the "center" of this bin
		xs[bin] = hist.BinToValue(float64(bin) + 0.5)
		weights[bin] = float64(count)
	}

	bw := k.Bandwidth
	if bw == nil {
		bw = Scott
	}

	kFixed := k
	kFixed.Bandwidth = FixedBandwidth(bw.HistBandwidth(hist, ss))
	return kFixed.FromSample(Sample{Xs: xs, Weights: weights})

	// TODO(austin) Somehow warn when too much weight is outside
	// histogram?
}

type kdeDist struct {
	kernel      Dist
	xs, weights []float64
	bm          BoundaryMethod
	min, max    float64 // Support bounds
}

// normalizedXs returns x - kde.xs.  Evaluating kernels shifted by
// kde.xs all at x is equivalent to evaluating one unshifted kernel at
// x - kde.xs.
func (kde *kdeDist) normalizedXs(x float64) []float64 {
	txs := make([]float64, len(kde.xs))
	for i, xi := range kde.xs {
		txs[i] = x - xi
	}
	return txs
}

func (kde *kdeDist) PDF(x float64) float64 {
	// Apply boundary
	if x < kde.min || x >= kde.max {
		return 0
	}

	y := func(x float64) float64 {
		// Shift kernel to each of kde.xs and evaluate at x
		ys := kde.kernel.PDFEach(kde.normalizedXs(x))

		// Kernel samples are weighted according to the weights of xs
		wys := Sample{Xs: ys, Weights: kde.weights}

		return wys.Sum() / wys.Weight()
	}
	switch kde.bm {
	default:
		panic("unknown boundary correction method")
	case boundaryNone:
		return y(x)
	case BoundaryReflect:
		if math.IsInf(kde.max, 1) {
			return y(x) + y(2*kde.min-x)
		} else if math.IsInf(kde.min, -1) {
			return y(x) + y(2*kde.max-x)
		} else {
			d := 2 * (kde.max - kde.min)
			w := 2 * (x - kde.min)
			return series(func(n float64) float64 {
				// Points >= x
				return y(x+n*d) + y(x+n*d-w)
			}) + series(func(n float64) float64 {
				// Points < x
				return y(x-(n+1)*d+w) + y(x-(n+1)*d)
			})
		}
	}
}

func (kde *kdeDist) PDFEach(xs []float64) []float64 {
	return atEach(kde.PDF, xs)
}

func (cdf *kdeDist) CDF(x float64) float64 {
	// Apply boundary
	if x < cdf.min {
		return 0
	} else if x >= cdf.max {
		return 1
	}

	y := func(x float64) float64 {
		// Shift kernel integral to each of cdf.xs and evaluate at x
		ys := cdf.kernel.CDFEach(cdf.normalizedXs(x))

		// Kernel samples are weighted according to the weights of xs
		wys := Sample{Xs: ys, Weights: cdf.weights}

		return wys.Sum() / wys.Weight()
	}
	switch cdf.bm {
	default:
		panic("unknown boundary correction method")
	case boundaryNone:
		return y(x)
	case BoundaryReflect:
		if math.IsInf(cdf.max, 1) {
			return y(x) - y(2*cdf.min-x)
		} else if math.IsInf(cdf.min, -1) {
			return y(x) + (1 - y(2*cdf.max-x))
		} else {
			d := 2 * (cdf.max - cdf.min)
			w := 2 * (x - cdf.min)
			return series(func(n float64) float64 {
				// Windows >= x-w
				return y(x+n*d) - y(x+n*d-w)
			}) + series(func(n float64) float64 {
				// Windows < x-w
				return y(x-(n+1)*d) - y(x-(n+1)*d-w)
			})
		}
	}
}

func (cdf *kdeDist) CDFEach(xs []float64) []float64 {
	return atEach(cdf.CDF, xs)
}

func (kde *kdeDist) InvCDF(x float64) float64 {
	panic("not implemented")
}

func (kde *kdeDist) InvCDFEach(cs []float64) []float64 {
	panic("not implemented")
}

func (cdf *kdeDist) Bounds() (low float64, high float64) {
	// TODO(austin) If this KDE came from a histogram, we'd better
	// not sample at a significantly higher rate than the
	// histogram.  Maybe we want to just return the bounds of the
	// histogram?

	// TODO(austin) It would be nice if this could be instructed
	// to include all original data points, even if they are in
	// the tail.  Probably that should just be up to the caller to
	// pass an axis derived from the bounds of the original data.

	// Use the lowest and highest samples as starting points
	lowX, highX := Sample{Xs: cdf.xs, Weights: cdf.weights}.Bounds()
	if lowX == highX {
		lowX -= 1
		highX += 1
	}

	// Find the end points that contain 99% of the CDF's weight.
	// Since bisect requires that the root be bracketed, start by
	// expanding our range if necessary.  TODO(austin) This can
	// definitely be done faster.
	const (
		lowY      = 0.005
		highY     = 0.995
		tolerance = 0.001
	)
	for cdf.CDF(lowX) > lowY {
		lowX -= highX - lowX
	}
	for cdf.CDF(highX) < highY {
		highX += highX - lowX
	}
	// Explicitly accept discontinuities, since we may be using a
	// discontiguous kernel.
	low, _ = bisect(func(x float64) float64 { return cdf.CDF(x) - lowY }, lowX, highX, tolerance)
	high, _ = bisect(func(x float64) float64 { return cdf.CDF(x) - highY }, lowX, highX, tolerance)

	// Expand width by 20% to give some margins
	width := high - low
	low, high = low-0.1*width, high+0.1*width

	// Limit to bounds
	low, high = math.Max(low, cdf.min), math.Min(high, cdf.max)

	return
}
