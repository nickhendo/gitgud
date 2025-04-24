package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type Git struct {
	remoteName    string
	cloneUrl      string
	defaultBranch string
	repoLocation  string

	tracePacket bool
	trace       bool
	curlVerbose bool
}

func (g Git) Command(name string, arg ...string) (*exec.Cmd, *strings.Builder, *strings.Builder) {
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

func (g Git) CreateBareRepo(repoName string) error {
	if !strings.HasSuffix(repoName, ".git") {
		return fmt.Errorf("bare repository name must end with '.git'")
	}

	if strings.Contains(repoName, " ") {
		return fmt.Errorf("bare repository name must not contain spaces")
	}

	repoPath := strings.Join([]string{g.repoLocation, repoName}, "/")
	slog.Debug("creating repository at", "path", repoPath)
	command, stdOut, stdErr := g.Command(
		"git",
		"init",
		"--bare",
		fmt.Sprintf("--initial-branch=%s", g.defaultBranch),
		repoPath,
	)
	err := command.Run()
	slog.Debug(stdOut.String())
	slog.Debug(stdErr.String())

	return err
}

func (g Git) DeleteRepo(repoName string) error {
	input := strings.Join([]string{g.repoLocation, repoName}, "/")
	slog.Debug("attempting to delete repository", "path", input)
	command, stdOut, stdErr := g.Command("rm", "-rf", input)
	err := command.Run()
	slog.Debug("DeleteRepo", "stdOut", stdOut.String())
	slog.Debug("DeleteRepo", "stdErr", stdErr.String())
	return err
}
