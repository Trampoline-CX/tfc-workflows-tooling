package environment

import (
	"fmt"
	"log"
	"maps"
	"os"
	"strings"
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
		log.Printf("[ERROR] GITHUB_OUTPUT environment variable not set")
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	file, err := os.OpenFile(gh.githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to open GitHub output file: %s", err)
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("[ERROR] Failed to close GitHub output file: %s", err)
			retErr = err
		}
	}()

	log.Printf("[DEBUG] Writing %d outputs to GitHub output file", len(gh.output))

	for key, value := range gh.output {
		strValue := value.String()

		// Log each output value for troubleshooting
		log.Printf("[DEBUG] Output value for '%s': '%s'", key, strValue)

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
			log.Printf("[ERROR] Failed to write output '%s': %s", key, err)
			retErr = err
			return
		}

		log.Printf("[DEBUG] Wrote output: %s", key)
	}

	// Ensure data is flushed to disk before returning
	if err := file.Sync(); err != nil {
		log.Printf("[ERROR] Failed to sync GitHub output file: %s", err)
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
	log.Printf("[DEBUG] GitHub environment - GITHUB_RUN_ID: %s", runId)
	log.Printf("[DEBUG] GitHub environment - GITHUB_RUN_NUMBER: %s", runNumber)
	log.Printf("[DEBUG] GitHub environment - GITHUB_OUTPUT: %s", githubOutput)
	log.Printf("[DEBUG] GitHub environment - GITHUB_SHA: %s", getenv("GITHUB_SHA"))
	log.Printf("[DEBUG] GitHub environment - GITHUB_ACTOR: %s", getenv("GITHUB_ACTOR"))
	log.Printf("[DEBUG] GitHub environment - GITHUB_REPOSITORY: %s", getenv("GITHUB_REPOSITORY"))
	log.Printf("[DEBUG] GitHub environment - GITHUB_REF_NAME: %s", getenv("GITHUB_REF_NAME"))
	log.Printf("[DEBUG] GitHub environment - GITHUB_REF_TYPE: %s", getenv("GITHUB_REF_TYPE"))

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
		log.Printf("[WARN] GITHUB_OUTPUT environment variable is not set. Outputs will not be available in GitHub Actions.")

		// Fallback to legacy GITHUB_ENV if available (for older Actions versions)
		legacyEnv := getenv("GITHUB_ENV")
		if legacyEnv != "" {
			log.Printf("[INFO] Using GITHUB_ENV as fallback for outputs: %s", legacyEnv)
			ghCtx.githubOutput = legacyEnv
		}
	}

	return ghCtx
}
