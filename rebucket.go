// Package rebucket implements the ReBucket duplicate crash report clustering algorithm
/*

http://research.microsoft.com/en-us/groups/sa/rebucket-icse2012.pdf

*/
package rebucket

import (
	"math"

	"github.com/bugsnag/bugsnag-go/errors"
)

func distance(e1, e2 *errors.Error, c, o float64) float64 {

	c1 := e1.StackFrames()
	c2 := e2.StackFrames()

	if len(c1) == 0 {
		return 1
	}

	if len(c2) == 0 {
		return 1
	}

	M := make([][]float64, len(c1)+1)
	for i := range M {
		M[i] = make([]float64, len(c2)+1)
	}

	for i := 1; i <= len(c1); i++ {
		for j := 1; j <= len(c2); j++ {
			var x float64
			// TODO(dgryski): better 'equality' comparison here
			if c1[i-1].Name == c2[j-1].Name {
				x = math.Exp(-c*fmin(i-1, j-1)) * math.Exp(-o*fabs(i-j))
			}
			M[i][j] = fmax3(M[i-1][j-1]+x, M[i-1][j], M[i][j-1])
		}
	}

	var sig float64

	for i := 0; i < min(len(c1), len(c2)); i++ {
		sig += math.Exp(-c * float64(i))
	}

	res := M[len(c1)][len(c2)] / sig
	return 1 - res

}

type Cluster struct {
	Idx []int
}

type pair struct {
	i, j int
}

type distanceCache map[pair]float64

func (dcache distanceCache) distance(p pair, e1, e2 *errors.Error, c, o float64) float64 {
	var d float64
	var ok bool
	if d, ok = dcache[p]; !ok {
		d = distance(e1, e2, c, o)
		dcache[p] = d
	}

	return d
}

func clusterDistance(errs []*errors.Error, c1, c2 Cluster, c, o float64, dCache distanceCache) float64 {

	maxd := math.Inf(-1)

	for _, i := range c1.Idx {
		for _, j := range c2.Idx {
			p := pair{i, j}
			d := dCache.distance(p, errs[i], errs[j], c, o)
			if d > maxd {
				maxd = d
			}
		}
	}

	return maxd
}

// ClusterErrors returns a set clusters of stacktraces in errs.  dthresh is the
// distance threshold to be considered 'similar', c is a coefficient for the
// distance to the top frame, o is a coefficient for the alignment offset.
func ClusterErrors(errs []*errors.Error, dthresh, c, o float64) []Cluster {

	// to start, every cluster contains only a single error
	clusters := make([]Cluster, len(errs))
	for i := range errs {
		clusters[i] = Cluster{Idx: []int{i}}
	}

	// TODO(dgryski): Need a better algorithm for this.
	// Until we get that, cache cluster distances

	dCache := make(distanceCache)

	var done bool
	for !done {
		var tomerge pair
		done = true
		minD := math.Inf(1)
		// find the closest two clusters, within the distance threshold
		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				d := clusterDistance(errs, clusters[i], clusters[j], c, o, dCache)
				if d < dthresh && d < minD {
					minD = d
					tomerge = pair{i, j}
					done = false
				}
			}
		}
		if !done {
			// add nodes from clusters[j] to clusters[i]
			clusters[tomerge.i].Idx = append(clusters[tomerge.i].Idx, clusters[tomerge.j].Idx...)
			// remove cluster[j]
			clusters[tomerge.j] = clusters[len(clusters)-1]
			clusters = clusters[:len(clusters)-1]
		}
	}

	// create a new slice to avoid the extra cruft left over in clusters
	cret := make([]Cluster, len(clusters))
	copy(cret, clusters)
	return cret
}

func fmax3(x, y, z float64) float64 {
	return math.Max(x, math.Max(y, z))
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func fmin(i, j int) float64 {
	return float64(min(i, j))
}

func fabs(i int) float64 {
	if i < 0 {
		return float64(-i)
	}
	return float64(i)
}
