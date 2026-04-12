package main

import (
	"fmt"
	"os"

	"github.com/APICerberus/APICerebrus/internal/cli"
)

// osExit allows mocking os.Exit for testing
var osExit = os.Exit

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}
