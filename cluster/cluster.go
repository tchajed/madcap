package cluster

// A point in a metric space that can compute its distance to other points
type Point interface {
	Vector() []float64
	Distance(other Point) float64
}

// Cluster the given points into k clusters, returning a list of assignments
// with the semantics assignments[i] is the cluster index in [0, k) of points[i]
func Kmeans(points []Point, k int) (assignments []int) {
	assignments = make([]int, len(points))
	return
}
