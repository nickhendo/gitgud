package git

import (
	"fmt"
	"gitgud/config"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func NewRemoteRepository(baseURL, orgName, repoName string) (GitRemoteRepository, error) {
	if strings.HasSuffix(repoName, ".git") {
		return GitRemoteRepository{}, fmt.Errorf("%s must not end with '.git' as it is added automatically", repoName)
	}

	if strings.Contains(repoName, " ") {
		return GitRemoteRepository{}, fmt.Errorf("bare repository name must not contain spaces")
	}

	if strings.Contains(orgName, " ") {
		return GitRemoteRepository{}, fmt.Errorf("org name must not contain spaces")
	}

	fullRepoName := repoName + ".git"
	return GitRemoteRepository{
		Name:     repoName,
		OrgName:  orgName,
		FullName: fullRepoName,
		CloneURL: fmt.Sprintf("%s/git/%s/%s", baseURL, orgName, fullRepoName),

		DefaultBranch: config.Settings.DefaultBranch,

		GitRepository: GitRepository{
			FullPath:    strings.Join([]string{config.Settings.RepositoriesLocation, orgName, fullRepoName}, "/"),
			tracePacket: config.Settings.Debug,
			trace:       config.Settings.Debug,
			curlVerbose: config.Settings.Debug,
		},
	}, nil
}

type GitRepository struct {
	// Location of repo on filesystem
	FullPath string

	// Debug settings

	tracePacket bool
	trace       bool
	curlVerbose bool
}

type GitRemoteRepository struct {
	GitRepository

	Name          string
	OrgName       string
	FullName      string
	DefaultBranch string
	CloneURL      string
}

type GitClonedRepository struct {
	GitRepository
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

type File struct {
	Name string
}

type EmptyRepositoryError struct {
	BranchName string
}

func (e EmptyRepositoryError) Error() string {
	return fmt.Sprintf("repository is empty on branch: %s", e.BranchName)
}

// Return a slice of files from the given branch of the GitRepository
func (g GitRepository) GetFiles(branchName string) ([]File, error) {
	slog.Debug("getting files...")

	command, stdOut, stdErr := g.Command(
		"git",
		"ls-tree",
		branchName,
		"--full-tree",
		"-r",
		"--name-only",
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())
	slog.Error(stdErr.String())

	if err != nil {
		if err.Error() == "exit status 128" && strings.Contains(stdErr.String(), fmt.Sprintf("fatal: Not a valid object name %s\n", branchName)) {
			return []File{}, EmptyRepositoryError{branchName}
		}

		return []File{}, fmt.Errorf("failure looking for files in git repo: %s (%w)", stdErr.String(), err)
	}

	slog.Debug("files retrieved.")

	files := strings.Split(stdOut.String(), "\n")
	files = files[:len(files)-1]
	fileList := []File{}
	for _, fileName := range files {
		fileList = append(fileList, File{fileName})
	}

	return fileList, nil
}

// Create a new git remote (bare) repository at the configured FullPath.
// Overwrite the DefaultBranch before calling this if required.
func (g GitRemoteRepository) CreateBareRepo() error {
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

func (g GitRemoteRepository) CallService(service string, advertiseRefs bool) *exec.Cmd {
	slog.Debug("calling service", "service", service, "path", g.FullPath)

	var command *exec.Cmd
	if advertiseRefs {
		command = exec.Command(
			"git",
			strings.Replace(service, "git-", "", 1),
			"--stateless-rpc",
			"--advertise-refs",
			"--http-backend-info-refs",
			".",
		)
	} else {
		command = exec.Command(
			"git",
			strings.Replace(service, "git-", "", 1),
			"--stateless-rpc",
			".",
		)
	}

	slog.Debug("command", "args", command)

	command.Dir = g.FullPath

	command.Env = append(command.Env, "GIT_PROTOCOL=version=2")

	if g.tracePacket {
		command.Env = append(command.Env, "GIT_TRACE_PACKET=1")
	}

	if g.trace {
		command.Env = append(command.Env, "GIT_TRACE=1")
	}

	if g.curlVerbose {
		command.Env = append(command.Env, "GIT_CURL_VERBOSE=1")
	}

	return command
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

func (g GitRemoteRepository) Clone(destination string) (GitClonedRepository, error) {
	clonePath := strings.Join([]string{config.Settings.ClonesLocation, g.OrgName, destination}, "/")
	slog.Debug("cloning repository", "repo", g.CloneURL, "dest", clonePath)

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
		return GitClonedRepository{}, fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Debug("repository cloned.")

	clonedRepo := GitClonedRepository{
		g.GitRepository,
	}
	clonedRepo.FullPath = clonePath

	return clonedRepo, nil
}

func (g GitClonedRepository) AddAll() error {
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

func (g GitClonedRepository) GetConfig() (string, error) {
	slog.Debug("getting config...")

	command, stdOut, stdErr := g.Command(
		"git",
		"config",
		"--list",
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		return "", fmt.Errorf("failed to get config: %s (%w)", stdErr, err)
	}

	slog.Debug("Config retrieved.")

	return stdOut.String(), nil
}

func (g GitClonedRepository) SetConfig(key, value string) error {
	slog.Debug("setting config...", "key", key, "value", value)

	command, stdOut, stdErr := g.Command(
		"git",
		"config",
		key,
		value,
	)
	command.Dir = g.FullPath

	err := command.Run()
	slog.Debug(stdOut.String())

	if err != nil {
		return fmt.Errorf("failed to set config: %s (%w)", stdErr, err)
	}

	slog.Debug("Config set.")

	return nil
}

func (g GitClonedRepository) Commit(message string) error {
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

func (g GitClonedRepository) Push() error {
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
