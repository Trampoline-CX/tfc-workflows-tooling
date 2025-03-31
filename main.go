// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/tfci/internal/environment"
	"github.com/hashicorp/tfci/internal/logging"
	"github.com/hashicorp/tfci/version"
	"github.com/mitchellh/cli"
)

var (
	Ui     cli.Ui
	appCtx context.Context
	env    *environment.CI
)

func main() {
	// load env
	env = environment.NewCIContext()

	// setup logging
	logging.SetupLogger(&logging.LoggerOptions{
		PlatformType: string(env.PlatformType),
	})

	// Ensure logs are flushed on exit
	defer func() {
		if err := logging.Sync(); err != nil {
			// Don't use logging here to avoid circular references
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Ui settings
	Ui = &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
			Reader:      os.Stdin,
		},
	}

	appCtx = context.Background()

	os.Exit(realMain())
}

func realMain() int {
	logging.Info("Starting application",
		"version", version.GetVersion(),
		"go_version", runtime.Version())

	logging.Debug("Preparing runner")
	cliRunner, runError := newCliRunner()
	if runError != nil {
		logging.Error("Failed to create CLI runner", "error", runError)
		Ui.Error(runError.Error())
		return 1
	}

	logging.Debug("Running command")
	exitCode, err := cliRunner.Run()
	if err != nil {
		logging.Error("Command execution failed", "error", err)
		Ui.Error(err.Error())
		return 1
	}

	return exitCode
}
