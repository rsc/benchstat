// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"fmt"
	"io"
	"math"
	"os"
)

type Plot struct {
	// F is the function to plot.
	F Func

	// X and Y are the X and Y axis configuration.
	X, Y Axis

	// Samples is the number of samples to use on the Y axis.  If
	// this is zero, a default value is used.
	Samples int
}

type Axis struct {
	// Lo and High specify the lower and upper bounds on this
	// Axis, respectively.  If these are both 0, this Axis is
	// autoscaled.
	Low, High float64

	// Log specifies a logarithmic scale of this base.  If this is
	// 0, the Axis uses a linear scale.
	Log float64
}

func (p *Plot) sample(defSamples int) (xs []float64, ys []float64) {
	if p.Samples != 0 {
		defSamples = p.Samples
	}

	if p.X.Log != 0 {
		logLo := math.Log(p.X.Low) / math.Log(p.X.Log)
		logHigh := math.Log(p.X.High) / math.Log(p.X.Log)
		xs = Logspace(logLo, logHigh, defSamples, p.X.Log)
	} else {
		xs = Linspace(p.X.Low, p.X.High, defSamples)
	}

	// If f is integrable, use the integral to get the average
	// function value around each sample.  This way the area under
	// the plotted curve is correct even if there are narrow
	// features.
	fint := p.F.Integrate()
	if fint != nil && len(xs) > 1 {
		ys = make([]float64, len(xs))
		w := (xs[1] - xs[0]) / 2
		left := fint.At(xs[0] - 0.5*w)
		for i, x := range xs {
			right := fint.At(x + 0.5*w)
			ys[i] = (right - left) / w
			left = right
		}
	} else {
		// Otherwise just take point samples
		ys = p.F.AtEach(xs)
	}
	return
}

// AutoScale sets autoscaled axes according to the function being
// plotted and returns p.
func (p *Plot) AutoScale() *Plot {
	if p.X.Low == 0 && p.X.High == 0 {
		p.X.Low, p.X.High = p.F.Bounds()
	}

	if p.Y.Low == 0 && p.Y.High == 0 {
		_, ys := p.sample(500)
		p.Y.Low, p.Y.High = Bounds(ys)
		if p.Y.Low == p.Y.High {
			p.Y.High += 1
		}
	}

	return p
}

// Values computes plottable values of p.F evenly spaced on the Y axis
// and returns the resulting X and Y coordinates.
func (p *Plot) Values() ([]float64, []float64) {
	return p.AutoScale().sample(100)
}

// p.ASCII() is shorthand for p.FASCII(os.Stdout).
func (p *Plot) ASCII() error {
	return p.FASCII(os.Stdout)
}

// FASCII prints a beautiful ASCII representation of p.F to w.
func (p *Plot) FASCII(w io.Writer) error {
	p.AutoScale()

	const width = 60
	xs, ys := p.sample(30)

	dots := make([]rune, 2+width)
	for i := range dots {
		dots[i] = '•'
	}

	// TODO(austin) Print Y axis
	lowY, highY := p.Y.Low, p.Y.High
	for i, y := range ys {
		label := fmt.Sprintf("%7.5g", xs[i])
		_, err := fmt.Fprintf(w, "%11s | %s\n", label, string(dots[:1+int(width*(y-lowY)/highY)]))
		if err != nil {
			return err
		}
	}
	return nil
}

// p.Table() is shorthand for p.FTable(os.Stdout).
func (p *Plot) Table() error {
	return p.FTable(os.Stdout)
}

// Table prints a table of "X Y" coordinates for p.F to w.
func (p *Plot) FTable(w io.Writer) error {
	p.AutoScale()

	xs, ys := p.sample(500)
	for i, y := range ys {
		if _, err := fmt.Fprintln(w, xs[i], y); err != nil {
			return err
		}
	}
	return nil
}
