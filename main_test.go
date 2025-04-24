package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Git struct {
	remoteName string
	cloneUrl   string

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

func TestBranch(t *testing.T) {
	if testing.Verbose() {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	remoteRepoName := "test_repo.git"
	clonedRepoName1 := "test_cloned_repo_1"

	// Router needed for handling the specific URL
	router := GetRouter()
	ts := httptest.NewServer(router)
	defer ts.Close()

	cloneUrl := fmt.Sprintf("%s/repos/test/%s", ts.URL, remoteRepoName)

	git := Git{
		remoteName:  remoteRepoName,
		cloneUrl:    cloneUrl,
		tracePacket: testing.Verbose(),
		trace:       testing.Verbose(),
		curlVerbose: testing.Verbose(),
	}

	// Create a hosted repo
	err := CreateRepo(remoteRepoName)
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteRepo(remoteRepoName)

	contents, err := os.ReadFile(fmt.Sprintf("repositories/%s/HEAD", remoteRepoName))
	if err != nil {
		t.Fatal(err)
	}

	expected := "ref: refs/heads/main\n"
	if string(contents) != expected {
		t.Fatalf("HEAD contents -> expected: %s, got %s", expected, contents)
	}

	// Git clone the repo elsewhere
	fmt.Printf("cloneUrl: %v\n", cloneUrl)
	command, stdOut, stdErr := git.Command(
		"git",
		"clone",
		cloneUrl,
		fmt.Sprintf("repositories/%s", clonedRepoName1),
	)
	err = command.Run()

	log.Println("Git clone stdOut: ", stdOut.String())
	log.Println("Git clone stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}

	defer DeleteRepo(clonedRepoName1)

	command, stdOut, stdErr = git.Command(
		"git",
		"branch",
		"--show-current",
	)
	command.Dir = strings.Join([]string{"repositories", clonedRepoName1}, "/")
	err = command.Run()

	log.Println("Git branch stdOut: ", stdOut.String())
	log.Println("Git branch stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}

	expected = "main\n"
	if stdOut.String() != expected {
		t.Fatalf("branch name -> expected: %s, got %s", expected, stdOut.String())
	}
}

func TestClone(t *testing.T) {
	remoteRepoName := "test_repo.git"
	clonedRepoName1 := "test_cloned_repo_1"
	clonedRepoName2 := "test_cloned_repo_2"
	fileContents := "This is a readme"

	// Start server

	// Router needed for handling the specific URL
	router := GetRouter()

	ts := httptest.NewServer(router)
	defer ts.Close()

	cloneUrl := fmt.Sprintf("%s/repos/test/%s", ts.URL, remoteRepoName)

	git := Git{
		remoteName:  remoteRepoName,
		cloneUrl:    cloneUrl,
		tracePacket: testing.Verbose(),
		trace:       testing.Verbose(),
		curlVerbose: testing.Verbose(),
	}

	// Create a hosted repo
	err := CreateRepo(remoteRepoName)
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteRepo(remoteRepoName)

	// Git clone the repo elsewhere
	fmt.Printf("cloneUrl: %v\n", cloneUrl)
	command, stdOut, stdErr := git.Command(
		"git",
		"clone",
		cloneUrl,
		fmt.Sprintf("repositories/%s", clonedRepoName1),
	)
	err = command.Run()

	log.Println("Git clone stdOut: ", stdOut.String())
	log.Println("Git clone stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}

	defer DeleteRepo(clonedRepoName1)

	// Create new file, add, commit and push to remote
	err = os.WriteFile(fmt.Sprintf("repositories/%s/readme.md", clonedRepoName1), []byte(fileContents), 0644)
	if err != nil {
		t.Fatal(err)
	}

	command, stdOut, stdErr = git.Command(
		"git",
		"add",
		".",
	)
	command.Dir = fmt.Sprintf("repositories/%s", clonedRepoName1)
	err = command.Run()

	log.Println("Git add stdOut: ", stdOut.String())
	log.Println("Git add stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}

	command, stdOut, stdErr = git.Command(
		"git",
		"commit",
		"-m",
		"Initial commit",
	)
	command.Dir = fmt.Sprintf("repositories/%s", clonedRepoName1)
	err = command.Run()

	log.Println("Git commit stdOut: ", stdOut.String())
	log.Println("Git commit stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}

	command, stdOut, stdErr = git.Command(
		"git",
		"push",
	)
	command.Dir = fmt.Sprintf("repositories/%s", clonedRepoName1)
	err = command.Run()

	log.Println("Git push stdOut: ", stdOut.String())
	log.Println("Git push stdErr: ", stdErr.String())

	if err != nil {
		fmt.Println("Error during push")
		t.Fatal(err)
	}

	// Clone repo elsewhere and verify contents

	// Git clone the repo elsewhere
	command, stdOut, stdErr = git.Command(
		"git",
		"clone",
		cloneUrl,
		fmt.Sprintf("repositories/%s", clonedRepoName2),
	)
	err = command.Run()

	log.Println("Git clone2 stdOut: ", stdOut.String())
	log.Println("Git clone2 stdErr: ", stdErr.String())

	if err != nil {
		t.Fatal(err)
	}
	defer DeleteRepo(clonedRepoName2)

	contents, err := os.ReadFile(fmt.Sprintf("repositories/%s/readme.md", clonedRepoName2))
	if err != nil {
		t.Fatal(err)
	}

	if string(contents) != fileContents {
		t.Fail()
	}
}

func TestCreateRepo(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		repoName string
		wantErr  bool
	}{
		{
			repoName: "test",
			wantErr:  true,
		},
		{
			repoName: "test.git",
			wantErr:  false,
		},
		{
			repoName: "test repository with spaces.git",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := CreateRepo(tt.repoName)
			defer DeleteRepo(tt.repoName)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CreateRepo() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CreateRepo() succeeded unexpectedly")
			}
		})
	}
}
