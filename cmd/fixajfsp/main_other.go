//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "fixajfsp läuft nur unter Windows.")
	os.Exit(1)
}
