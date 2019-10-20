// main is the package containing the entry point of slk
package main

import (
	"os"

	"github.com/lk16/slk/internal/slk"
)

func main() {
	slack, err := slk.NewSlk(os.Args[1:])

	// TODO
	_ = slack
	_ = err
}
