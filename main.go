// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Benchstat computes and compares statistics about benchmarks.
//
// Usage:
//
//	benchstat old.txt [new.txt]
//
// Each input file should contain the concatenated output of a number
// of runs of ``go test -bench.'' For each different benchmark listed in an input file,
// benchstat computes the mean, minimum, and maximum run time,
// after removing outliers using the interquartile range rule.
//
// If invoked on a single input file, benchstat prints the per-benchmark statistics
// for that file.
//
// If invoked on a pair of input files, benchstat adds to the output a column
// showing the statistics from the second file and a column showing the
// percent change in mean from the first to the second file.
// Next to the percent change, benchstat shows the p value from the
// two-sample Welch t-test, which measures the statistical significance
// of the difference. If the t-test indicates that the measured difference is
// not statistically significant (defined as p > 0.05), benchstat replaces
// the percent change with a single ~.
//
// Example
//
// Suppose we collect benchmark results from running ``go test -bench=Encode''
// five times before and after a particular change.
//
// The file old.txt contains:
//
//	BenchmarkGobEncode   	100	  13552735 ns/op	  56.63 MB/s
//	BenchmarkJSONEncode  	 50	  32395067 ns/op	  59.90 MB/s
//	BenchmarkGobEncode   	100	  13553943 ns/op	  56.63 MB/s
//	BenchmarkJSONEncode  	 50	  32334214 ns/op	  60.01 MB/s
//	BenchmarkGobEncode   	100	  13606356 ns/op	  56.41 MB/s
//	BenchmarkJSONEncode  	 50	  31992891 ns/op	  60.65 MB/s
//	BenchmarkGobEncode   	100	  13683198 ns/op	  56.09 MB/s
//	BenchmarkJSONEncode  	 50	  31735022 ns/op	  61.15 MB/s
//
// The file new.txt contains:
//
//	BenchmarkGobEncode   	 100	  11773189 ns/op	  65.19 MB/s
//	BenchmarkJSONEncode  	  50	  32036529 ns/op	  60.57 MB/s
//	BenchmarkGobEncode   	 100	  11942588 ns/op	  64.27 MB/s
//	BenchmarkJSONEncode  	  50	  32156552 ns/op	  60.34 MB/s
//	BenchmarkGobEncode   	 100	  11786159 ns/op	  65.12 MB/s
//	BenchmarkJSONEncode  	  50	  31288355 ns/op	  62.02 MB/s
//	BenchmarkGobEncode   	 100	  11628583 ns/op	  66.00 MB/s
//	BenchmarkJSONEncode  	  50	  31559706 ns/op	  61.49 MB/s
//	BenchmarkGobEncode   	 100	  11815924 ns/op	  64.96 MB/s
//	BenchmarkJSONEncode  	  50	  31765634 ns/op	  61.09 MB/s
//
// The order of the lines in the file does not matter, except that the
// output lists benchmarks in order of appearance.
//
// If run with just one input file, benchstat summarizes that file:
//
//	$ benchstat old.txt
//	name        mean
//	GobEncode   13.6ms × (1.00,1.01)
//	JSONEncode  32.1ms × (0.99,1.01)
//	$
//
// If run with two input files, benchstat summarizes and compares:
//
//	$ benchstat old.txt new.txt
//	name        old mean              new mean              delta
//	GobEncode   13.6ms × (1.00,1.01)  11.8ms × (0.99,1.01)  -13.31% (p=0.000)
//	JSONEncode  32.1ms × (0.99,1.01)  31.8ms × (0.99,1.01)     ~    (p=0.154)
//	$
//
// Note that the JSONEncode result is reported as
// statistically insignificant instead of a -1% delta.
//
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"rsc.io/benchstat/internal/stats"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: benchstat old.txt [new.txt]\n")
	os.Exit(2)
}

func main() {
	log.SetPrefix("benchstats: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 || flag.NArg() > 2 {
		flag.Usage()
	}

	before := readFile(flag.Arg(0))
	var out [][]string
	if flag.NArg() == 1 {
		out = append(out, []string{"name", "mean"})
		for _, old := range before.Benchmarks {
			out = append(out, []string{old.Name, old.Time(old.Scaler())})
		}
	} else {
		after := readFile(flag.Arg(1))
		out = append(out, []string{"name", "old mean            ", "new mean            ", "delta"})
		for _, old := range before.Benchmarks {
			new := after.ByName[old.Name]
			if new == nil {
				continue
			}

			ttest, err := stats.TwoSampleWelchTTest(stats.Sample{Xs: old.RTimes}, stats.Sample{Xs: new.RTimes})
			significant := false
			if err == nil {
				significant = ttest.P <= 0.05
			}

			scaler := old.Scaler()
			row := []string{old.Name, old.Time(scaler), new.Time(scaler), "~   "}
			if err == stats.ErrZeroVariance {
				row[3] = "zero variance"
			} else if err == stats.ErrSampleSize {
				row[3] = "too few samples"
			} else if err != nil {
				row[3] = fmt.Sprintf("(%s)", err)
			} else if significant {
				row[3] = fmt.Sprintf("%+.2f%%", ((new.Mean/old.Mean)-1.0)*100.0)
			}
			if ttest != nil {
				row[3] += fmt.Sprintf(" (p=%0.3f)", ttest.P)
			}
			out = append(out, row)
		}
	}

	max := []int{0, 0, 0, 0}
	for _, row := range out {
		for i, s := range row {
			n := utf8.RuneCountInString(s)
			if max[i] < n {
				max[i] = n
			}
		}
	}

	if flag.NArg() == 1 {
		row := out[0]
		fmt.Printf("%-*s  %s\n", max[0], row[0], row[1])
		for _, row := range out[1:] {
			fmt.Printf("%-*s  %*s\n", max[0], row[0], max[1], row[1])
		}
	} else {
		row := out[0]
		fmt.Printf("%-*s  %*s  %*s  %s\n", max[0], row[0], max[1], row[1], max[2], row[2], row[3])
		for _, row := range out[1:] {
			fmt.Printf("%-*s  %*s  %*s  %*s\n", max[0], row[0], max[1], row[1], max[2], row[2], max[3], row[3])
		}
	}
}

func (b *Benchmark) Scaler() func(*Benchmark) string {
	var format string
	var scale float64
	switch x := b.Mean / 1e9; {
	case x >= 99.5:
		format, scale = "%.0fs", 1
	case x >= 9.95:
		format, scale = "%.1fs", 1
	case x >= 0.995:
		format, scale = "%.2fs", 1
	case x >= 0.0995:
		format, scale = "%.0fms", 1000
	case x >= 0.00995:
		format, scale = "%.1fms", 1000
	case x >= 0.000995:
		format, scale = "%.2fms", 1000
	case x >= 0.0000995:
		format, scale = "%.0fµs", 1000*1000
	case x >= 0.00000995:
		format, scale = "%.1fµs", 1000*1000
	case x >= 0.000000995:
		format, scale = "%.2fµs", 1000*1000
	case x >= 0.0000000995:
		format, scale = "%.0fns", 1000*1000*1000
	case x >= 0.00000000995:
		format, scale = "%.1fns", 1000*1000*1000
	default:
		format, scale = "%.2fns", 1000*1000*1000
	}
	return func(b *Benchmark) string {
		return fmt.Sprintf(format, b.Mean/1e9*scale)
	}
}

func (b *Benchmark) Time(scaler func(*Benchmark) string) string {
	return fmt.Sprintf("%s × (%.2f,%.2f)", scaler(b), b.Min/b.Mean, b.Max/b.Mean)
}

// A Group is a collection of benchmark results for a specific version of the code.
type Group struct {
	Benchmarks []*Benchmark
	ByName     map[string]*Benchmark
}

// A Benchmark is a collection of runs of a specific benchmark.
type Benchmark struct {
	Name   string
	Runs   []Run
	Times  []float64 // Times of runs in nanoseconds
	RTimes []float64 // Times with outliers removed
	Min    float64   // minimum run
	Mean   float64
	Max    float64 // maximum run
}

// A Run records the result of a single benchmark run.
type Run struct {
	N  int
	Ns float64
}

// readFile reads a benchmark group from a file.
func readFile(file string) *Group {
	g := &Group{ByName: make(map[string]*Benchmark)}

	text, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	for _, line := range strings.Split(string(text), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		name := f[0]
		if !strings.HasPrefix(name, "Benchmark") {
			continue
		}
		name = strings.TrimPrefix(name, "Benchmark")
		n, _ := strconv.Atoi(f[1])
		var ns float64
		for i := 2; i+2 <= len(f); i += 2 {
			if f[i+1] == "ns/op" {
				ns, _ = strconv.ParseFloat(f[i], 64)
				break
			}
		}
		if n == 0 || ns == 0 {
			continue
		}
		b := g.ByName[name]
		if b == nil {
			b = &Benchmark{Name: name}
			g.ByName[name] = b
			g.Benchmarks = append(g.Benchmarks, b)
		}
		b.Runs = append(b.Runs, Run{n, ns})
		b.Times = append(b.Times, ns)
	}

	for _, b := range g.Benchmarks {
		// Discard outliers.
		times := stats.Sample{Xs: b.Times}
		q1, q3 := times.Percentile(0.25), times.Percentile(0.75)
		lo, hi := q1-1.5*(q3-q1), q3+1.5*(q3-q1)
		for _, time := range b.Times {
			if lo <= time && time <= hi {
				b.RTimes = append(b.RTimes, time)
			}
		}

		// Compute statistics of remaining data.
		b.Min, b.Max = stats.Bounds(b.RTimes)
		b.Mean = stats.Mean(b.RTimes)
	}

	return g
}
