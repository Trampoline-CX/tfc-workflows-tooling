package environment

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/tfci/internal/logging"
)

const EOF = "\n"

// Sourced from: https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
type GitHubContext struct {
	// A unique number for each workflow run within a repository. This number does not change if you re-run the workflow run
	runId string
	// A unique number for each run of a particular workflow in a repository. This number begins at 1 for the workflow's first run, and increments with each new run. This number does not change if you re-run the workflow run
	runNumber string
	// The commit SHA that triggered the workflow. The value of this commit SHA depends on the event that triggered the workflow.
	commitSHA string
	// The name of the person or app that initiated the workflow. For example, octocat.
	actor string
	// The owner and repository name. For example, octocat/Hello-World
	repository string
	// The short ref name of the branch or tag that triggered the workflow run. This value matches the branch or tag name shown on GitHub
	refName string
	// The type of ref that triggered the workflow run. Valid values are branch or tag.
	refType string
	// The path to a temporary directory on the runner. This directory is emptied at the beginning and end of each job. Note that files will not be removed if the runner's user account does not have permission to delete them.
	runnerTemp string
	// path to output file for GitHub Actions
	githubOutput string
	// data accumulated for output
	output OutputMap
	// unique delimiter for multiline outputs
	fileDelimeter string
}

func (gh *GitHubContext) ID() string {
	return fmt.Sprintf("gha-%s-%s", gh.runId, gh.runNumber)
}

func (gh *GitHubContext) SHA() string {
	return gh.commitSHA
}
func (gh *GitHubContext) SHAShort() string {
	if len(gh.commitSHA) > 7 {
		return gh.commitSHA[:7]
	}
	return gh.commitSHA
}

func (gh *GitHubContext) Author() string {
	return gh.actor
}

func (gh *GitHubContext) WriteDir() string {
	return gh.runnerTemp
}

func (gh *GitHubContext) SetOutput(output OutputMap) {
	if gh.output == nil {
		gh.output = make(map[string]OutputWriter)
	}

	maps.Copy(gh.output, output)
}

func (gh *GitHubContext) CloseOutput() (retErr error) {
	if gh.githubOutput == "" {
		logging.Error("GITHUB_OUTPUT environment variable not set")
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	file, err := os.OpenFile(gh.githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.Error("Failed to open GitHub output file", "error", err)
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.Error("Failed to close GitHub output file", "error", err)
			retErr = err
		}
	}()

	logging.Debug("Writing outputs to GitHub output file", "count", len(gh.output))

	for key, value := range gh.output {
		strValue := value.String()

		// Log each output value for troubleshooting
		logging.Debug("Setting GitHub output", "key", key, "value", strValue)

		var outputLine string
		if value.MultiLine() || strings.Contains(strValue, "\n") {
			outputLine = fmt.Sprintf("%s<<%s%s%s%s%s%s",
				key,
				gh.fileDelimeter,
				EOF,
				strValue,
				EOF,
				gh.fileDelimeter,
				EOF)
		} else {
			outputLine = fmt.Sprintf("%s=%s%s", key, strValue, EOF)
		}

		if _, err := file.WriteString(outputLine); err != nil {
			logging.Error("Failed to write output", "key", key, "error", err)
			retErr = err
			return
		}

		logging.Debug("Successfully wrote output", "key", key)
	}

	// Ensure data is flushed to disk before returning
	if err := file.Sync(); err != nil {
		logging.Error("Failed to sync GitHub output file", "error", err)
		if retErr == nil {
			retErr = err
		}
	}

	// Write to stdout as well for debugging in GitHub Actions logs
	for key, value := range gh.output {
		fmt.Printf("::set-output name=%s::%s\n", key, value.String())
	}

	gh.output = make(map[string]OutputWriter)
	return
}

func newGitHubContext(getenv GetEnv) *GitHubContext {
	runId := getenv("GITHUB_RUN_ID")
	runNumber := getenv("GITHUB_RUN_NUMBER")
	githubOutput := getenv("GITHUB_OUTPUT")

	// Log all GitHub environment variables for debugging
	logging.Debug("GitHub environment variables", 
		"GITHUB_RUN_ID", runId,
		"GITHUB_RUN_NUMBER", runNumber,
		"GITHUB_OUTPUT", githubOutput,
		"GITHUB_SHA", getenv("GITHUB_SHA"),
		"GITHUB_ACTOR", getenv("GITHUB_ACTOR"),
		"GITHUB_REPOSITORY", getenv("GITHUB_REPOSITORY"),
		"GITHUB_REF_NAME", getenv("GITHUB_REF_NAME"),
		"GITHUB_REF_TYPE", getenv("GITHUB_REF_TYPE"))

	ghCtx := &GitHubContext{
		runId:        runId,
		runNumber:    runNumber,
		commitSHA:    getenv("GITHUB_SHA"),
		actor:        getenv("GITHUB_ACTOR"),
		repository:   getenv("GITHUB_REPOSITORY"),
		refName:      getenv("GITHUB_REF_NAME"),
		refType:      getenv("GITHUB_REF_TYPE"),
		githubOutput: githubOutput,
		runnerTemp:   getenv("RUNNER_TEMP"),
		output:       make(map[string]OutputWriter),
	}

	ghCtx.fileDelimeter = fmt.Sprintf("GHDELIM_%s_%s_%d", runId, runNumber, os.Getpid())

	if ghCtx.githubOutput == "" {
		logging.Warn("GITHUB_OUTPUT environment variable is not set. Outputs will not be available in GitHub Actions.")

		// Fallback to legacy GITHUB_ENV if available (for older Actions versions)
		legacyEnv := getenv("GITHUB_ENV")
		if legacyEnv != "" {
			logging.Info("Using GITHUB_ENV as fallback for outputs", "path", legacyEnv)
			ghCtx.githubOutput = legacyEnv
		}
	}

	return ghCtx
}
