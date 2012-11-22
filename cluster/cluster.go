package cluster

// A point in a metric space that can compute its distance to other points
type Point interface {
	Vector() []float64
	Distance(other Point) float64
}
