//go:build !windows

package platformversion

func Current() Version {
	return Version{}
}
