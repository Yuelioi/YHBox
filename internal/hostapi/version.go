// Package hostapi owns the compatibility generation exposed by Yotta hosts to
// Node Packages. It evolves independently from the Yotta product release.
package hostapi

const (
	Current      = "1.0"
	NextMajor    = "2.0"
	CurrentMajor = 1
	CurrentMinor = 0
)

func Supported() []string {
	return []string{Current}
}
