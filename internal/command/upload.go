// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/logging"
)

type UploadConfigurationCommand struct {
	*Meta
	Workspace   string
	Directory   string
	Speculative bool
	Provisional bool
}

func (c *UploadConfigurationCommand) flags() *flag.FlagSet {
	f := c.flagSet("upload")

	f.StringVar(&c.Workspace, "workspace", "", "The name of the workspace to create the new configuration version in.")
	f.StringVar(&c.Directory, "directory", "", "Path to the configuration files on disk.")
	f.BoolVar(&c.Speculative, "speculative", false, "When true, this configuration version may only be used to create runs which are speculative, that is, can neither be confirmed nor applied.")
	f.BoolVar(&c.Provisional, "provisional", false, "When true, this configuration version does not immediately become the workspace's current configuration until a run referencing it is ultimately applied.")
	return f
}

func (c *UploadConfigurationCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	logging.Debug("Uploading configuration", 
		"workspace", c.Workspace,
		"directory", c.Directory,
		"speculative", c.Speculative,
		"provisional", c.Provisional)

	dirPath, dirError := filepath.Abs(c.Directory)
	if dirError != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("error resolving directory path %s", dirError.Error()))
		return 1
	}

	logging.Debug("Target directory for configuration upload", "path", dirPath)

	configVersion, cvError := c.cloud.UploadConfig(c.appCtx, cloud.UploadOptions{
		Workspace:              c.Workspace,
		Organization:           c.organization,
		ConfigurationDirectory: dirPath,
		Speculative:            c.Speculative,
		Provisional:            c.Provisional,
	})

	if cvError != nil {
		status := c.resolveStatus(cvError)
		c.addOutput("status", string(status))
		c.addConfigurationDetails(configVersion)
		c.writer.ErrorResult(fmt.Sprintf("error uploading configuration version to HCP Terraform: %s", cvError.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addConfigurationDetails(configVersion)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *UploadConfigurationCommand) addConfigurationDetails(config *tfe.ConfigurationVersion) {
	if config != nil {
		// Log to help debug the configuration version details
		logging.Debug("Configuration version details", 
			"id", config.ID, 
			"status", string(config.Status))
		
		// Add outputs that will be used by subsequent workflow steps
		c.addOutput("configuration_version_id", config.ID)
		c.addOutput("configuration_version_status", string(config.Status))
		
		// Explicitly log the output values to make troubleshooting easier
		fmt.Printf("::set-output name=configuration_version_id::%s\n", config.ID)
		fmt.Printf("::set-output name=configuration_version_status::%s\n", string(config.Status))
	} else {
		logging.Warn("Configuration version is nil, no outputs will be set")
	}

	c.addOutputWithOpts("payload", config, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})
}

func (c *UploadConfigurationCommand) Help() string {
	helpText := `
Usage: tfci [global options] upload [options]

	Creates and uploads a new configuration version for the provided workspace.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with HCP Terraform. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   HCP Terraform Organization Name.

Options:

	-workspace      The name of the HCP Terraform Workspace to create and upload the terraform configuration version in.

	-directory      Path to the terraform configuration files on disk.

	-speculative    When true, this configuration version may only be used to create runs which are speculative, that is, can neither be confirmed nor applied.

	-provisional    When true, this configuration version does not immediately become the workspace's current configuration until a run referencing it is ultimately applied.
	`
	return strings.TrimSpace(helpText)
}

func (c *UploadConfigurationCommand) Synopsis() string {
	return "Creates and uploads a new configuration version for the provided workspace"
}
