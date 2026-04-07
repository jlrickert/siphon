package main

import (
	"context"
	_ "embed"
	"os"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/siphon/pkg/cli"
)

//go:embed LICENSE
var licenseText string

func main() {
	cli.LicenseText = licenseText

	ctx := context.Background()

	rt, err := toolkit.NewRuntime()
	if err != nil {
		os.Exit(1)
	}

	_, err = cli.Run(ctx, rt, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
}
