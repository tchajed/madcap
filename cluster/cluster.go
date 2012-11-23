package cluster

import (
	"math"
	"math/rand"
)

// A point in a metric space
type Point interface {
	Vector() []float64
}

type Points interface {
	Vector(i int) []float64
	Dim() int
	Len() int
}

type slicePoint []float64

func (p slicePoint) Vector() []float64 {
	return []float64(p)
}

type DistanceFunc func(first, second Point) float64

// Compute the (squared) Euclidean distance between two points; assumes same
// length, or at least len(first.Vector()) >= len(second.Vector())
func EuclideanDistanceF(first, second Point) float64 {
	var distance float64
	firstVec := first.Vector()
	secondVec := second.Vector()
	for i, f := range firstVec {
		delta := f - secondVec[i]
		distance += delta * delta
	}
	return distance
}

var EuclideanDistance = DistanceFunc(EuclideanDistanceF)

// Destructive operation that normalizes each dimension of points to be on a
// [0,1] scale.
func normalize(points Points) {
	min, max := pointsRange(points)
	for i := 0; i < points.Len(); i++ {
		point := points.Vector(i)
		for d := range point {
			point[d] -= min[d]
			dimRange := max[d] - min[d]
			if dimRange != 0.0 {
				point[d] /= dimRange
			}
			if math.IsNaN(point[d]) {
				point[d] = 0.0
			}
			if math.IsInf(point[d], 0) {
				point[d] = 1.0
			}
		}
	}
}

func pointsRange(points Points) (min, max []float64) {
	dims := points.Dim()
	min = make([]float64, dims)
	max = make([]float64, dims)
	copy(min, points.Vector(0))
	copy(max, points.Vector(0))
	// find the range in each dimension to generate random initial positions
	for i := 0; i < points.Len(); i++ {
		vec := points.Vector(i)
		for d, v := range vec {
			if v < min[d] {
				min[d] = v
			}
			if v > max[d] {
				max[d] = v
			}
		}
	}
	return
}

func initVector(d int, v float64) []float64 {
	vector := make([]float64, d)
	for i := range vector {
		vector[i] = v
	}
	return vector
}

func randomCenter(min, max []float64) slicePoint {
	vector := make([]float64, len(min))
	for d := 0; d < len(vector); d++ {
		vector[d] = min[d] + rand.Float64()*(max[d]-min[d])
	}
	return slicePoint(vector)
}

// Compute the center of each cluster based on some assignments. Also returns a
// list of clusters that no longer have points in them.
func meanCenters(points Points, assignments []int, k int) (centers []slicePoint, nilClusters []int) {
	// number of points in each cluster
	numPoints := make([]int, k)
	centers = make([]slicePoint, k)
	nilClusters = make([]int, 0)
	dim := points.Dim()
	for i := range centers {
		centers[i] = slicePoint(make([]float64, dim))
	}
	for i, cluster := range assignments {
		center := centers[cluster]
		point := points.Vector(i)
		numPoints[cluster]++
		// add new point
		for d := range center {
			center[d] += point[d]
		}
	}
	// divide by number of items to produce mean points
	for cluster := range centers {
		if numPoints[cluster] == 0 {
			nilClusters = append(nilClusters, cluster)
		} else {
			for d := range centers[cluster] {
				centers[cluster][d] /= float64(numPoints[cluster])
			}
		}
	}
	return
}

func findClosest(centers []slicePoint, p Point, distF DistanceFunc) int {
	var minIndex int = -1
	var minDistance float64 = math.MaxFloat64
	for i, center := range centers {
		distance := distF(center, p)
		if distance < minDistance {
			minIndex, minDistance = i, distance
		}
	}
	return minIndex
}

// Cluster the given points into k clusters, returning a list of assignments
// with the semantics assignments[i] is the cluster index in [0, k) of points[i].
// The dimensionality is taken to be the dimension of the first point.
func Kmeans(points Points, k int, distanceFunc DistanceFunc) (assignments []int, cost float64) {
	if points.Len() == 0 {
		return
	}
	assignments = make([]int, points.Len())
	normalize(points)
	min, max := pointsRange(points)
	centers := make([]slicePoint, k)
	for i := 0; i < k; i++ {
		centers[i] = randomCenter(min, max)
	}
	// repeat a few times for convergence
	for iter := 0; iter < 10; iter++ {
		// assign samples to closest mean
		for xi := 0; xi < points.Len(); xi++ {
			x := slicePoint(points.Vector(xi))
			assignments[xi] = findClosest(centers, x, distanceFunc)
			if assignments[xi] < 0 {
				assignments[xi] = rand.Int() % k
			}
		}
		// get new centers
		centers, nilClusters := meanCenters(points, assignments, k)
		// re-initialize these clusters
		for _, cluster := range nilClusters {
			centers[cluster] = randomCenter(min, max)
		}
		if len(nilClusters) > 0 {
			for _, i := range nilClusters {
				centers[i] = randomCenter(min, max)
			}
		}
	}
	numPoints := make([]int, k)
	for i, cluster := range assignments {
		cost += distanceFunc(slicePoint(points.Vector(i)), centers[cluster])
		numPoints[cluster]++
	}
	var sumSquaredPoints float64
	for _, num := range numPoints {
		sumSquaredPoints += float64(num * num)
	}
	// encourages an even spread by penalizing piling all points into one cluster
	cost *= sumSquaredPoints
	return
}
