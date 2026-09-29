// Command dotfiles manages this dotfiles repo: install and doctor.
//
// Usage:
//
//	go run ./cmd/dotfiles install [--dry-run]
//	go run ./cmd/dotfiles doctor
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "install":
		dry := false
		for _, a := range os.Args[2:] {
			if a == "--dry-run" {
				dry = true
			} else {
				fmt.Fprintf(os.Stderr, "unknown flag: %s\n", a)
				usage()
				os.Exit(2)
			}
		}
		if err := install(dry); err != nil {
			errMsg(err.Error())
			os.Exit(1)
		}
	case "doctor":
		if len(os.Args) > 2 {
			fmt.Fprintln(os.Stderr, "doctor takes no flags")
			usage()
			os.Exit(2)
		}
		if !doctor() {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `dotfiles — manage this dotfiles repo

Usage:
  go run ./cmd/dotfiles install [--dry-run]   wire up dotfiles on a new machine
  go run ./cmd/dotfiles doctor                 check machine state, report gaps`)
}
