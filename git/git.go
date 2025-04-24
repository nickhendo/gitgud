package git

import (
	"fmt"
	"gitgud/config"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func NewRepository(baseURL, orgName, repoName string) (GitRepository, error) {
	if strings.HasSuffix(repoName, ".git") {
		return GitRepository{}, fmt.Errorf("bare repository name must not end with '.git' as it is added automatically")
	}

	if strings.Contains(repoName, " ") {
		return GitRepository{}, fmt.Errorf("bare repository name must not contain spaces")
	}

	if strings.Contains(orgName, " ") {
		return GitRepository{}, fmt.Errorf("org name must not contain spaces")
	}

	fullRepoName := repoName + ".git"
	return GitRepository{
		Name:     repoName,
		OrgName:  orgName,
		FullName: fullRepoName,
		CloneURL: fmt.Sprintf("%s/%s/%s", baseURL, orgName, fullRepoName),
		IsBare:   true,

		// Location of repo on filesystem
		FullPath:      strings.Join([]string{config.Settings.RepositoriesLocation, orgName, fullRepoName}, "/"),
		DefaultBranch: config.Settings.DefaultBranch,

		tracePacket: config.Settings.Debug,
		trace:       config.Settings.Debug,
		curlVerbose: config.Settings.Debug,
	}, nil
}

type GitRepository struct {
	Name          string
	OrgName       string
	FullName      string
	FullPath      string
	DefaultBranch string
	CloneURL      string
	IsBare        bool

	// Debug settings
	tracePacket bool
	trace       bool
	curlVerbose bool
}

func (g GitRepository) Command(name string, arg ...string) (*exec.Cmd, *strings.Builder, *strings.Builder) {
	command := exec.Command(name, arg...)

	if g.tracePacket {
		command.Env = append(command.Env, "GIT_TRACE_PACKET=1")
	}

	if g.trace {
		command.Env = append(command.Env, "GIT_TRACE=1")
	}

	if g.curlVerbose {
		command.Env = append(command.Env, "GIT_CURL_VERBOSE=1")
	}

	var stdOut strings.Builder
	var stdErr strings.Builder

	command.Stdout = &stdOut
	command.Stderr = &stdErr

	return command, &stdOut, &stdErr
}

// Create a new git remote (bare) repository at the configured FullPath.
// Overwrite the DefaultBranch before calling this if required.
func (g GitRepository) CreateBareRepo() error {
	slog.Debug("creating repository", "path", g.FullPath)

	command, stdOut, stdErr := g.Command(
		"git",
		"init",
		"--bare",
		fmt.Sprintf("--initial-branch=%s", g.DefaultBranch),
		g.FullPath,
	)

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return fmt.Errorf("failed to create bare repo: %w", err)
	}

	slog.Debug("repository created.")
	return nil
}

// Delete the remote repository if it exists.
func (g GitRepository) DeleteRepo() error {
	slog.Debug("attempting to delete", "path", g.FullPath)

	err := os.RemoveAll(g.FullPath)

	slog.Debug("deleted.")
	return fmt.Errorf("failed to delete repository: %w", err)
}

func (g GitRepository) GetBranch() (string, error) {
	slog.Debug("getting current branch...")

	command, stdOut, stdErr := g.Command(
		"git",
		"branch",
		"--show-current",
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	slog.Debug("branch retrieved.")

	// Remove trailing newline from branch output
	return stdOut.String()[:stdOut.Len()-1], nil
}

func (g GitRepository) Clone(destination string) (GitRepository, error) {
	if !g.IsBare {
		return GitRepository{}, fmt.Errorf("can only clone a bare repo")
	}
	slog.Debug("cloning repository", "repo", g.CloneURL, "dest", destination)

	clonePath := strings.Join([]string{config.Settings.ClonesLocation, g.OrgName, destination}, "/")
	command, stdOut, stdErr := g.Command(
		"git",
		"clone",
		g.CloneURL,
		clonePath,
	)

	err := command.Run()

	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return GitRepository{}, fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Debug("repository cloned.")

	clonedRepo := g
	clonedRepo.FullPath = clonePath
	clonedRepo.IsBare = false

	return clonedRepo, nil
}

func (g GitRepository) AddAll() error {
	if g.IsBare {
		return fmt.Errorf("can only work in cloned repo")
	}
	slog.Debug("adding all files...")

	command, stdOut, stdErr := g.Command(
		"git",
		"add",
		".",
	)
	command.Dir = g.FullPath

	err := command.Run()

	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return fmt.Errorf("failed to add all: %w", err)
	}

	slog.Debug("all files added.")

	return nil
}

func (g GitRepository) Commit(message string) error {
	if g.IsBare {
		return fmt.Errorf("can only work in cloned repo")
	}
	slog.Debug("committing...")

	command, stdOut, stdErr := g.Command(
		"git",
		"commit",
		"-m",
		message,
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return fmt.Errorf("failed to commit: %w", err)
	}

	slog.Debug("Committed.")

	return nil
}

func (g GitRepository) Push() error {
	if g.IsBare {
		return fmt.Errorf("can only work in cloned repo")
	}
	slog.Debug("pushing...")

	command, stdOut, stdErr := g.Command(
		"git",
		"push",
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		slog.Error(stdErr.String())
		return fmt.Errorf("failed to push: %w", err)
	}

	slog.Debug("pushed.")

	return nil
}
