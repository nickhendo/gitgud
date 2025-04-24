package main

import (
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"gitgud/git"
)

func TestDefaultBranch(t *testing.T) {
	if testing.Verbose() {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	remoteRepoName := "test_repo"
	clonedRepoName := "test_cloned_repo"

	router := GetRouter()
	ts := httptest.NewServer(router)
	defer ts.Close()

	testRepo, err := git.NewRemoteRepository(ts.URL, "test_org", remoteRepoName)
	if err != nil {
		t.Fatal(err)
	}

	err = testRepo.CreateBareRepo()
	if err != nil {
		t.Fatal(err)
	}
	defer testRepo.DeleteRepo()

	contents, err := os.ReadFile(fmt.Sprintf("%s/HEAD", testRepo.FullPath))
	if err != nil {
		t.Fatal(err)
	}

	expected := "ref: refs/heads/main\n"
	if string(contents) != expected {
		t.Fatalf("HEAD contents -> expected: %s, got %s", expected, contents)
	}

	clonedRepo, err := testRepo.Clone(clonedRepoName)
	if err != nil {
		t.Fatal(err)
	}
	defer clonedRepo.DeleteRepo()

	branchName, err := clonedRepo.GetBranch()
	if err != nil {
		t.Fatal(err)
	}

	expected = "main"
	if branchName != expected {
		t.Fatalf("branch name -> expected: %s, got %s", expected, branchName)
	}
}

func TestClone(t *testing.T) {
	remoteRepoName := "test_repo"
	clonedRepoName1 := "test_cloned_repo_1"
	clonedRepoName2 := "test_cloned_repo_2"
	fileContents := "This is a readme"

	router := GetRouter()

	ts := httptest.NewServer(router)
	defer ts.Close()

	testRepo, err := git.NewRemoteRepository(ts.URL, "test_org", remoteRepoName)
	if err != nil {
		t.Fatal(err)
	}

	err = testRepo.CreateBareRepo()
	if err != nil {
		t.Fatal(err)
	}
	defer testRepo.DeleteRepo()

	clonedRepo, err := testRepo.Clone(clonedRepoName1)
	if err != nil {
		t.Fatal(err)
	}
	defer clonedRepo.DeleteRepo()

	// Create new file, add, commit and push to remote
	// TODO: Look at file perms
	err = os.WriteFile(fmt.Sprintf("%s/readme.md", clonedRepo.FullPath), []byte(fileContents), 0750)
	if err != nil {
		t.Fatal(err)
	}

	err = clonedRepo.AddAll()
	if err != nil {
		t.Fatal(err)
	}

	err = clonedRepo.Commit("Initial commit")
	if err != nil {
		t.Fatal(err)
	}

	err = clonedRepo.Push()
	if err != nil {
		t.Fatal(err)
	}

	// Clone repo elsewhere and verify contents

	otherClonedRepo, err := testRepo.Clone(clonedRepoName2)
	if err != nil {
		t.Fatal(err)
	}

	defer otherClonedRepo.DeleteRepo()

	contents, err := os.ReadFile(fmt.Sprintf("%s/readme.md", otherClonedRepo.FullPath))
	if err != nil {
		t.Fatal(err)
	}

	if string(contents) != fileContents {
		t.Fail()
	}
}
