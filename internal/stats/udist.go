// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// A UDist is the discrete probability distribution of the
// Mann-Whitney U statistic for a pair of samples of sizes N1 and N2.
//
// The details of computing this distribution with no ties can be
// found in Mann, Henry B.; Whitney, Donald R. (1947). "On a Test of
// Whether one of Two Random Variables is Stochastically Larger than
// the Other". Annals of Mathematical Statistics 18 (1): 50–60.
// Computing this distribution in the presence of ties is described in
// Klotz, J. H. (1966). "The Wilcoxon, Ties, and the Computer".
// Journal of the American Statistical Association 61 (315): 772-787
// and Cheung, Ying Kuen; Klotz, Jerome H. (1997). "The Mann Whitney
// Wilcoxon Distribution Using Linked Lists". Statistica Sinica 7:
// 805-813 (the former paper contains details that are glossed over in
// the latter paper but has mathematical typesetting issues, so it's
// easiest to get the context from the former paper and the details
// from the latter).
type UDist struct {
	N1, N2 int

	// T is the count of the number of ties at each rank in the
	// input distributions. T may be nil, in which case it is
	// assumed there are no ties (which is equivalent to an M+N
	// slice of 1s). It must be the case that Sum(T) == M+N.
	T []int
}

// hasTies returns true if d has any tied samples.
func (d UDist) hasTies() bool {
	for _, t := range d.T {
		if t > 1 {
			return true
		}
	}
	return false
}

// p returns the p_{d.N1,d.N2} function defined by Mann, Whitney 1947
// for values of U from 0 up to and including the U argument.
//
// This algorithm runs in Θ(N1*N2*U) = O(N1²N2²) time and is quite
// fast for small values of N1 and N2. However, it does not handle ties.
func (d UDist) p(U int) []float64 {
	// This is a dynamic programming implementation of the
	// recursive recurrence definition given by Mann and Whitney:
	//
	//   p_{n,m}(U) = (n * p_{n-1,m}(U-m) + m * p_{n,m-1}(U)) / (n+m)
	//   p_{n,m}(U) = 0                           if U < 0
	//   p_{0,m}(U) = p{n,0}(U) = 1 / nCr(m+n, n) if U = 0
	//                          = 0               if U > 0
	//
	// (Note that there is a typo in the original paper. The first
	// recursive application of p should be for U-m, not U-M.)
	//
	// Since p{n,m} only depends on p{n-1,m} and p{n,m-1}, we only
	// need to store one "plane" of the three dimensional space at
	// a time.
	//
	// Furthermore, p_{n,m} = p_{m,n}, so we only construct values
	// for n <= m and obtain the rest through symmetry.
	//
	// We organize the computed values of p as followed:
	//
	//       n →   N
	//     m *
	//     ↓ * *
	//       * * *
	//       * * * *
	//       * * * *
	//     M * * * *
	//
	// where each * is a slice indexed by U. The code below
	// computes these left-to-right, top-to-bottom, so it only
	// stores one row of this matrix at a time. Furthermore,
	// computing an element in a given U slice only depends on the
	// same and smaller values of U, so we can overwrite the U
	// slice we're computing in place as long as we start with the
	// largest value of U. Finally, even though the recurrence
	// depends on (n,m) above the diagonal and we use symmetry to
	// mirror those across the diagonal to (m,n), the mirrored
	// indexes are always available in the current row, so this
	// mirroring does not interfere with our ability to recycle
	// state.

	N, M := d.N1, d.N2
	if N > M {
		N, M = M, N
	}

	memo := make([][]float64, N+1)
	for n := range memo {
		memo[n] = make([]float64, U+1)
	}

	for m := 0; m <= M; m++ {
		// Compute p_{0,m}. This is zero except for U=0.
		memo[0][0] = 1

		// Compute the remainder of this row.
		nlim := N
		if m < nlim {
			nlim = m
		}
		for n := 1; n <= nlim; n++ {
			lp := memo[n-1] // p_{n-1,m}
			var rp []float64
			if n <= m-1 {
				rp = memo[n] // p_{n,m-1}
			} else {
				rp = memo[m-1] // p{m-1,n} and m==n
			}

			// For a given n,m, U is at most n*m.
			//
			// TODO: Actually, it's at most ⌈n*m/2⌉, but
			// then we need to use more complex symmetries
			// in the inner loop below.
			ulim := n * m
			if U < ulim {
				ulim = U
			}

			out := memo[n] // p_{n,m}
			nplusm := float64(n + m)
			for U1 := ulim; U1 >= 0; U1-- {
				l := 0.0
				if U1-m >= 0 {
					l = float64(n) * lp[U1-m]
				}
				r := float64(m) * rp[U1]
				out[U1] = (l + r) / nplusm
			}
		}
	}
	return memo[N]
}

type ukey struct {
	n1   int // size of first sample
	twoU int // 2*U statistic for this permutation
}

// This computes the CDF of the Mann-Whitney U distribution in the
// presence of ties using the computation from Cheung, Ying Kuen;
// Klotz, Jerome H. (1997). "The Mann Whitney Wilcoxon Distribution
// Using Linked Lists". Statistica Sinica 7: 805-813, with much
// guidance from appendix L of Klotz, A Computational Approach to
// Statistics.
//
// makeUmemo constructs the memoization table for the cumulative
// distribution of 2*U <= twoU, for sample sizes n1 and sum(t)-n1, and
// tie vector t. The result is a table memo[K][ukey{n1, 2*U}]=pr,
// where K is the number of ranks, n1 is the size of the first sample,
// U is the U statistic. pr is the probability of a permutation of a
// sample of size n1 in a ranking with tie vector t[:K] having a U
// statistic <= U.
func makeUmemo(twoU, n1 int, t []int) []map[ukey]float64 {
	// Another candidate for a fast implementation is van de Wiel,
	// "The split-up algorithm: a fast symbolic method for
	// computing p-values of distribution-free statistics". This
	// is what's used by R's coin package. It's a comparatively
	// recent publication, so it's presumably faster (or perhaps
	// just more general) than previous techniques, but I can't
	// get my hands on the paper.

	K := len(t)

	// Compute a coefficients. The a slice is indexed by k (a[0]
	// is unused).
	a := make([]int, K+1)
	a[1] = t[0]
	for k := 2; k <= K; k++ {
		a[k] = a[k-1] + t[k-2] + t[k-1]
	}

	// Create the memo table for the probability function. The pr
	// slice is indexed by k (pr[0] is unused).
	//
	// In "The Mann Whitney Distribution Using Linked Lists", they
	// use linked lists (*gasp*) for this, but within each K it's
	// really just a memoization table, so it's faster to use a
	// map. The outer structure is a slice indexed by k because we
	// need to find all memo entries with certain values of k.
	//
	// TODO: Compute the A function and normalize it to a
	// probability at the end. This should be much cheaper to
	// compute.
	//
	// TODO: The n1 and twoU values in the ukeys follow strict
	// patterns. For each K value, the n1 values are every integer
	// between two bounds. For each (K, n1) value, the twoU values
	// are every integer multiple of a certain base between two
	// bounds. It might be worth turning these into directly
	// indexible slices.
	pr := make([]map[ukey]float64, K+1)
	pr[K] = map[ukey]float64{ukey{n1: n1, twoU: twoU}: 0}

	// Compute memo table (k, n1, twoU) triples from high K values
	// to low K values. This drives the recurrence relation
	// downward to figure out all of the needed argument triples.
	//
	// TODO: Is it possible to generate this table bottom-up? If
	// so, this could be a pure dynamic programming algorithm and
	// we could discard the K dimension. We could at least store
	// the inputs in a more compact representation that replaces
	// the twoU dimension with an interval and a step size (as
	// suggested by Cheung, Klotz, not that they make it at all
	// clear *why* they're suggesting this).
	tsum := sumint(t) // always ∑ t[0:k]
	for k := K - 1; k >= 2; k-- {
		tsum -= t[k]
		pr[k] = make(map[ukey]float64)

		// Construct pr[k] from pr[k+1].
		for pr_kplus1 := range pr[k+1] {
			rkLow := maxint(0, pr_kplus1.n1-tsum)
			rkHigh := minint(pr_kplus1.n1, t[k])
			for rk := rkLow; rk <= rkHigh; rk++ {
				twoU_k := pr_kplus1.twoU - rk*(a[k+1]-2*pr_kplus1.n1+rk)
				n1_k := pr_kplus1.n1 - rk
				// TODO: Slice t instead of passing k?
				if twoUmin(k, n1_k, t, a) <= twoU_k && twoU_k <= twoUmax(k, n1_k, t, a) {
					key := ukey{n1: n1_k, twoU: twoU_k}
					pr[k][key] = 0
				}
			}
		}
	}

	// Fill probabilities in memo table from low K values to high
	// K values. This unwinds the recurrence relation.

	// Start with K==2 base case.
	//
	// TODO: Later computations depend on these, but these don't
	// depend on anything (including each other), so if K==2, we
	// can skip the memo table altogether.
	if K < 2 {
		panic("K < 2")
	}
	N_2 := t[0] + t[1]
	for pr_2i := range pr[2] {
		x := (pr_2i.twoU - pr_2i.n1*(t[0]-pr_2i.n1)) / N_2
		dist := HypergeometicDist{N: N_2, K: t[1], Draws: pr_2i.n1}
		pr[2][pr_2i] = dist.CDF(float64(x))
	}

	// Derive probabilities for the rest of the memo table.
	tsum = t[0] // always ∑ t[0:k-1]
	for k := 3; k <= K; k++ {
		tsum += t[k-2]
		N_k := tsum + t[k-1]

		// Compute pr[k] probabilities from pr[k-1] probabilities.
		for pr_ki := range pr[k] {
			prsum := 0.0
			dist := HypergeometicDist{N: N_k, K: t[k-1], Draws: pr_ki.n1}
			rkLow := maxint(0, pr_ki.n1-tsum)
			rkHigh := minint(pr_ki.n1, t[k-1])
			for rk := rkLow; rk <= rkHigh; rk++ {
				twoU_k := pr_ki.twoU - rk*(a[k]-2*pr_ki.n1+rk)
				n1_k := pr_ki.n1 - rk
				twoUmin := twoUmin(k-1, n1_k, t, a)
				twoUmax := twoUmax(k-1, n1_k, t, a)
				if twoUmin <= twoU_k && twoU_k <= twoUmax {
					pr1 := pr[k-1][ukey{n1: n1_k, twoU: twoU_k}]
					prsum += pr1 * dist.PMF(float64(rk))
				} else if twoUmax < twoU_k {
					prsum += dist.PMF(float64(rk))
				}
			}
			pr[k][pr_ki] = prsum
		}
	}

	return pr
}

func twoUmin(K, n1 int, t, a []int) int {
	twoU := -n1 * n1
	n1_k := n1
	for k := 1; k <= K; k++ {
		twoU_k := minint(n1_k, t[k-1])
		twoU += twoU_k * a[k]
		n1_k -= twoU_k
	}
	return twoU
}

func twoUmax(K, n1 int, t, a []int) int {
	twoU := -n1 * n1
	n1_k := n1
	for k := K; k > 0; k-- {
		twoU_k := minint(n1_k, t[k-1])
		twoU += twoU_k * a[k]
		n1_k -= twoU_k
	}
	return twoU
}

func (d UDist) PMF(U float64) float64 {
	if U < 0 || U >= 0.5+float64(d.N1*d.N2) {
		return 0
	}

	if d.hasTies() {
		// makeUmemo computes the CDF directly. Take its
		// difference to get the PMF.
		p1, ok1 := makeUmemo(int(2*U)-1, d.N1, d.T)[len(d.T)][ukey{d.N1, int(2*U) - 1}]
		p2, ok2 := makeUmemo(int(2*U), d.N1, d.T)[len(d.T)][ukey{d.N1, int(2 * U)}]
		if !ok1 || !ok2 {
			panic("makeUmemo did not return expected memoization table")
		}
		return p2 - p1
	}

	// There are no ties. Use the fast algorithm. U must be integral.
	Ui := int(math.Floor(U))
	// TODO: Use symmetry to minimize U
	return d.p(Ui)[Ui]
}

func (d UDist) CDF(U float64) float64 {
	if U < 0 {
		return 0
	} else if U >= float64(d.N1*d.N2) {
		return 1
	}

	if d.hasTies() {
		// TODO: Minimize U?
		p, ok := makeUmemo(int(2*U), d.N1, d.T)[len(d.T)][ukey{d.N1, int(2 * U)}]
		if !ok {
			panic("makeUmemo did not return expected memoization table")
		}
		return p
	}

	// There are no ties. Use the fast algorithm. U must be integral.
	Ui := int(math.Floor(U))
	// The distribution is symmetric around U = m * n / 2. Sum up
	// whichever tail is smaller.
	flip := Ui >= (d.N1*d.N2+1)/2
	if flip {
		Ui = d.N1*d.N2 - Ui - 1
	}
	pdfs := d.p(Ui)
	p := 0.0
	for _, pdf := range pdfs[:Ui+1] {
		p += pdf
	}
	if flip {
		p = 1 - p
	}
	return p
}

func (d UDist) Step() float64 {
	return 0.5
}

func (d UDist) Bounds() (float64, float64) {
	// TODO: More precise bounds when there are ties.
	return 0, float64(d.N1 * d.N2)
}
