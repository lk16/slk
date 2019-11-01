// main is the package containing the entry point of slk
package main

import (
	"fmt"
	"os"

	"github.com/lk16/slk/internal/slk"
)

func main() {
	slk, err := slk.NewSlk(os.Args[1:])
	if err != nil {
		fmt.Printf("slk initialization failed: %s\n", err.Error())
		os.Exit(1)
	}

	if err = slk.Run(); err != nil {
		fmt.Printf("slk crashed: %s\n", err.Error())
		os.Exit(1)
	}
}
