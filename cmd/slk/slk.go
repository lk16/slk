// main is the package containing the entry point of slk
package main

import (
	"os"

	"github.com/lk16/slk/internal/slk"
)

func main() {
	slk.NewSlk(os.Args[1:]).Run()
}
