// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Benchstat computes and compares statistics about benchmarks.
//
// Usage:
//
//	benchstat [-delta-test name] [-html] old.txt [new.txt] [more.txt ...]
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
// Next to the percent change, benchstat shows the p-value and sample
// sizes from a test of the two distributions of benchmark times.
// Small p-values indicate that the two distributions are significantly different.
// If the test indicates that there was no significant change between the two
// benchmarks (defined as p > 0.05), benchstat displays a single ~ instead of
// the percent change.
//
// The -delta-test option controls which significance test is applied:
// utest (Mann-Whitney U-test), ttest (two-sample Welch t-test), or none.
// The default is the U-test, sometimes also referred to as the Wilcoxon rank
// sum test.
//
// If invoked on more than two input files, benchstat prints the per-benchmark
// statistics for all the files, showing one column of statistics for each file,
// with no column for percent change or statistical significance.
//
// The -html option causes benchstat to print the results as an HTML table.
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
//	name        time/op
//	GobEncode   13.6ms ± 1%
//	JSONEncode  32.1ms ± 1%
//	$
//
// If run with two input files, benchstat summarizes and compares:
//
//	$ benchstat old.txt new.txt
//	name        old time/op  new time/op  delta
//	GobEncode   13.6ms ± 1%  11.8ms ± 1%  -13.31% (p=0.016 n=4+5)
//	JSONEncode  32.1ms ± 1%  31.8ms ± 1%     ~    (p=0.286 n=4+5)
//	$
//
// Note that the JSONEncode result is reported as
// statistically insignificant instead of a -0.93% delta.
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"rsc.io/benchstat/internal/stats"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: benchstat [options] old.txt [new.txt] [more.txt ...]\n")
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	flagDeltaTest = flag.String("delta-test", "utest", "significance `test` to apply to delta: utest, ttest, or none")
	flagHTML      = flag.Bool("html", false, "print results as an HTML table")
)

var deltaTestNames = map[string]func(old, new *Benchstat) (float64, error){
	"none":   notest,
	"u":      utest,
	"u-test": utest,
	"utest":  utest,
	"t":      ttest,
	"t-test": ttest,
	"ttest":  ttest,
}

func main() {
	log.SetPrefix("benchstats: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	deltaTest := deltaTestNames[strings.ToLower(*flagDeltaTest)]
	if flag.NArg() < 1 || deltaTest == nil {
		flag.Usage()
	}

	before := readFile(flag.Arg(0))
	var tables [][][]string
	switch flag.NArg() {
	case 1:
		for _, metric := range before.Metrics() {
			var table [][]string
			table = append(table, []string{"name", metric})
			for _, b := range before.Benchmarks {
				stat := b.Metric(metric)
				if stat == nil {
					continue
				}
				table = append(table, []string{b.Name, stat.Format(stat.Scaler())})
			}
			tables = append(tables, table)
		}

	case 2:
		after := readFile(flag.Arg(1))
		for _, metric := range before.Metrics() {
			var table [][]string
			for _, oldBench := range before.Benchmarks {
				newBench := after.ByName[oldBench.Name]
				if newBench == nil {
					continue
				}
				old := oldBench.Metric(metric)
				new := newBench.Metric(metric)
				if old == nil || new == nil {
					continue
				}
				if len(table) == 0 {
					table = append(table, []string{"name", "old " + metric, "new " + metric, "delta"})
				}

				pval, testerr := deltaTest(old, new)

				scaler := old.Scaler()
				row := []string{oldBench.Name, old.Format(scaler), new.Format(scaler), "~   "}
				if testerr == stats.ErrZeroVariance {
					row = append(row, "zero variance")
				} else if testerr == stats.ErrSampleSize {
					row = append(row, "too few samples")
				} else if testerr != nil {
					row = append(row, fmt.Sprintf("(%s)", testerr))
				} else if pval <= 0.05 {
					row[3] = fmt.Sprintf("%+.2f%%", ((new.Mean/old.Mean)-1.0)*100.0)
				}
				if len(row) == 4 && pval != -1 {
					row = append(row, fmt.Sprintf("(p=%0.3f n=%d+%d)", pval, len(old.RValues), len(new.RValues)))
				}
				table = append(table, row)
			}
			if len(table) > 0 {
				tables = append(tables, table)
			}
		}

	default:
		groups := []*Group{before}
		hdr := []string{"name \\ ", before.File}
		for _, file := range flag.Args()[1:] {
			group := readFile(file)
			groups = append(groups, group)
			hdr = append(hdr, group.File)
		}

		done := map[string]bool{}
		for _, group := range groups {
			for _, metric := range group.Metrics() {
				if done["metric:"+metric] {
					continue
				}
				done["metric:"+metric] = true

				var table [][]string
				thdr := append([]string{}, hdr...)
				thdr[0] += metric
				table = append(table, thdr)

				for _, bench := range group.Benchmarks {
					name := bench.Name
					if done["bench:"+metric+"/"+name] {
						continue
					}
					done["bench:"+metric+"/"+name] = true

					row := []string{name}
					var scaler func(*Benchstat) string
					for _, group := range groups {
						stat := group.ByName[name].Metric(metric)
						if stat == nil {
							row = append(row, "")
							continue
						}
						if scaler == nil {
							scaler = stat.Scaler()
						}
						row = append(row, stat.Format(scaler))
					}
					for row[len(row)-1] == "" {
						row = row[:len(row)-1]
					}
					table = append(table, row)
				}

				tables = append(tables, table)
			}
		}
	}

	numColumn := 0
	for _, table := range tables {
		for _, row := range table {
			if numColumn < len(row) {
				numColumn = len(row)
			}
		}
	}

	max := make([]int, numColumn)
	for _, table := range tables {
		for _, row := range table {
			for i, s := range row {
				n := utf8.RuneCountInString(s)
				if max[i] < n {
					max[i] = n
				}
			}
		}
	}

	var buf bytes.Buffer
	for i, table := range tables {
		if i > 0 {
			fmt.Fprintf(&buf, "\n")
		}

		if *flagHTML {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "<style>.benchstat tbody td:nth-child(1n+2) { text-align: right; padding: 0em 1em; }</style>\n")
			fmt.Fprintf(&buf, "<table class='benchstat'>\n")
			printRow := func(row []string, tag string) {
				fmt.Fprintf(&buf, "<tr>")
				for _, cell := range row {
					fmt.Fprintf(&buf, "<%s>%s</%s>", tag, html.EscapeString(cell), tag)
				}
				fmt.Fprintf(&buf, "\n")
			}
			printRow(table[0], "th")
			for _, row := range table[1:] {
				printRow(row, "td")
			}
			fmt.Fprintf(&buf, "</table>\n")
			continue
		}

		// headings
		row := table[0]
		for i, s := range row {
			switch i {
			case 0:
				fmt.Fprintf(&buf, "%-*s", max[i], s)
			default:
				fmt.Fprintf(&buf, "  %-*s", max[i], s)
			case len(row) - 1:
				fmt.Fprintf(&buf, "  %s\n", s)
			}
		}

		// data
		for _, row := range table[1:] {
			for i, s := range row {
				switch i {
				case 0:
					fmt.Fprintf(&buf, "%-*s", max[i], s)
				default:
					fmt.Fprintf(&buf, "  %*s", max[i], s)
				}
			}
			fmt.Fprintf(&buf, "\n")
		}
	}

	os.Stdout.Write(buf.Bytes())
}

func (b *Benchstat) TimeScaler() func(*Benchstat) string {
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
	return func(b *Benchstat) string {
		return fmt.Sprintf(format, b.Mean/1e9*scale)
	}
}

func (b *Benchstat) Scaler() func(*Benchstat) string {
	if b.Name == "time/op" {
		return b.TimeScaler()
	}

	var format string
	var scale float64
	var suffix string

	prescale := 1.0
	if b.Unit == "MB/s" {
		prescale = 1e6
	}

	switch x := b.Mean * prescale; {
	case x >= 99500000000000:
		format, scale, suffix = "%.0f", 1e12, "T"
	case x >= 9950000000000:
		format, scale, suffix = "%.1f", 1e12, "T"
	case x >= 995000000000:
		format, scale, suffix = "%.2f", 1e12, "T"
	case x >= 99500000000:
		format, scale, suffix = "%.0f", 1e9, "G"
	case x >= 9950000000:
		format, scale, suffix = "%.1f", 1e9, "G"
	case x >= 995000000:
		format, scale, suffix = "%.2f", 1e9, "G"
	case x >= 99500000:
		format, scale, suffix = "%.0f", 1e6, "M"
	case x >= 9950000:
		format, scale, suffix = "%.1f", 1e6, "M"
	case x >= 995000:
		format, scale, suffix = "%.2f", 1e6, "M"
	case x >= 99500:
		format, scale, suffix = "%.0f", 1e3, "k"
	case x >= 9950:
		format, scale, suffix = "%.1f", 1e3, "k"
	case x >= 995:
		format, scale, suffix = "%.2f", 1e3, "k"
	case x >= 99.5:
		format, scale, suffix = "%.0f", 1, ""
	case x >= 9.95:
		format, scale, suffix = "%.1f", 1, ""
	default:
		format, scale, suffix = "%.2f", 1, ""
	}

	if b.Unit == "B/op" {
		suffix += "B"
	}
	if b.Unit == "MB/s" {
		suffix += "B/s"
	}
	scale /= prescale

	return func(b *Benchstat) string {
		return fmt.Sprintf(format+suffix, b.Mean/scale)
	}
}

func (b *Benchstat) Format(scaler func(*Benchstat) string) string {
	diff := 1 - b.Min/b.Mean
	if d := b.Max/b.Mean - 1; d > diff {
		diff = d
	}
	return fmt.Sprintf("%s ±%3s", scaler(b), fmt.Sprintf("%.0f%%", diff*100.0))
}

// A Group is a collection of benchmark results for a specific version of the code.
type Group struct {
	File       string
	Benchmarks []*Benchmark
	ByName     map[string]*Benchmark
}

func (g *Group) Metrics() []string {
	have := map[string]bool{}
	var names []string
	for _, b := range g.Benchmarks {
		for _, r := range b.Runs {
			for _, m := range r.M {
				if !have[m.Name] {
					have[m.Name] = true
					names = append(names, m.Name)
				}
			}
		}
	}
	return names
}

// A Benchmark is a collection of runs of a specific benchmark.
type Benchmark struct {
	Name string
	Runs []Run
}

func (b *Benchmark) Metric(name string) *Benchstat {
	if b == nil {
		return nil
	}
	stat := &Benchstat{
		Name: name,
	}
	for _, r := range b.Runs {
		for _, m := range r.M {
			if m.Name == name {
				stat.Unit = m.Unit
				stat.Values = append(stat.Values, m.Val)
			}
		}
	}
	if stat.Values == nil {
		return nil
	}

	// Discard outliers.
	values := stats.Sample{Xs: stat.Values}
	q1, q3 := values.Percentile(0.25), values.Percentile(0.75)
	lo, hi := q1-1.5*(q3-q1), q3+1.5*(q3-q1)
	for _, value := range stat.Values {
		if lo <= value && value <= hi {
			stat.RValues = append(stat.RValues, value)
		}
	}

	// Compute statistics of remaining data.
	stat.Min, stat.Max = stats.Bounds(stat.RValues)
	stat.Mean = stats.Mean(stat.RValues)

	return stat
}

// A Benchstat is the metrics along one axis (e.g., ns/op or MB/s)
// for all runs of a specific benchmark.
type Benchstat struct {
	Name    string
	Unit    string
	Values  []float64 // metrics
	RValues []float64 // metrics with outliers removed
	Min     float64   // min of RValues
	Mean    float64   // mean of RValues
	Max     float64   // max of RValues
}

// A Run records the result of a single benchmark run.
type Run struct {
	N int
	M []Metric
}

type Metric struct {
	Name string
	Unit string
	Val  float64
}

// readFile reads a benchmark group from a file.
func readFile(file string) *Group {
	g := &Group{File: file, ByName: make(map[string]*Benchmark)}

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
		if n == 0 {
			continue
		}
		r := Run{N: n}
		for i := 2; i+2 <= len(f); i += 2 {
			val, err := strconv.ParseFloat(f[i], 64)
			if err != nil {
				continue
			}
			unit := f[i+1]
			name := unit
			switch unit {
			case "ns/op":
				name = "time/op"
			case "B/op":
				name = "alloc/op"
			case "MB/s":
				name = "speed"
			}
			r.M = append(r.M, Metric{Name: name, Unit: unit, Val: val})
		}
		if len(r.M) == 0 {
			continue
		}
		b := g.ByName[name]
		if b == nil {
			b = &Benchmark{Name: name}
			g.ByName[name] = b
			g.Benchmarks = append(g.Benchmarks, b)
		}
		b.Runs = append(b.Runs, r)
	}

	return g
}

// Significance tests.

func notest(old, new *Benchstat) (pval float64, err error) {
	return -1, nil
}

func ttest(old, new *Benchstat) (pval float64, err error) {
	t, err := stats.TwoSampleWelchTTest(stats.Sample{Xs: old.RValues}, stats.Sample{Xs: new.RValues}, stats.LocationDiffers)
	if err != nil {
		return -1, err
	}
	return t.P, nil
}

func utest(old, new *Benchstat) (pval float64, err error) {
	u, err := stats.MannWhitneyUTest(old.RValues, new.RValues, stats.LocationDiffers)
	if err != nil {
		return -1, err
	}
	return u.P, nil
}
