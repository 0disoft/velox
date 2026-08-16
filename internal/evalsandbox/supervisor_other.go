//go:build !windows

package evalsandbox

import "fmt"

func Run(Config) (Receipt, error) {
	return Receipt{}, fmt.Errorf("evaluation sandbox is supported only on Windows")
}

func SafeSummary(receipt Receipt) string {
	return receipt.SchemaVersion
}
