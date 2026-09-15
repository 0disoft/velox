//go:build ignore

package main

import (
	"os"

	"github.com/tc-hib/winres"
)

func main() {
	if len(os.Args) != 3 {
		panic("expected input.ico output.syso")
	}
	input, err := os.Open(os.Args[1])
	check(err)
	defer input.Close()
	icon, err := winres.LoadICO(input)
	check(err)
	resources := winres.ResourceSet{}
	check(resources.SetIcon(winres.ID(1), icon))
	check(resources.SetIcon(winres.ID(11), icon))
	output, err := os.Create(os.Args[2])
	check(err)
	check(resources.WriteObject(output, winres.ArchAMD64))
	check(output.Close())
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
