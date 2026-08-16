package evalsandbox

import "regexp"

var (
	trialIDPattern         = regexp.MustCompile(`^trial-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$`)
	seriesIDPattern        = regexp.MustCompile(`^series-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$`)
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
)
