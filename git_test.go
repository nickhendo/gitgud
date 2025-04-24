package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
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

	testRepo, err := NewRepository(ts.URL, "test_org", remoteRepoName)
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

	testRepo, err := NewRepository(ts.URL, "test_org", remoteRepoName)
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

func TestNewRepository(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		repoName string
		orgName  string
		wantErr  bool
	}{
		{
			name:     "valid repo and org name",
			repoName: "test",
			orgName:  "test_org",
			wantErr:  false,
		},
		{
			name:     "invalid repo name suffix",
			repoName: "test.git",
			orgName:  "test_org",
			wantErr:  true,
		},
		{
			name:     "invalid repo name with spaces",
			repoName: "test repository with spaces",
			orgName:  "test_org",
			wantErr:  true,
		},
		{
			name:     "invalid org name with spaces",
			repoName: "test",
			orgName:  "test org",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := NewRepository("", tt.orgName, tt.repoName)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NewRepository() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NewRepository() succeeded unexpectedly")
			}
		})
	}
}

func TestGitRepository_CreateBareRepo(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		baseURL  string
		orgName  string
		repoName string
		wantErr  bool
	}{
		{
			name:     "valid bare repo creation exists",
			baseURL:  "",
			orgName:  "test_org",
			repoName: "test_repo",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewRepository(tt.baseURL, tt.orgName, tt.repoName)
			defer g.DeleteRepo()

			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			gotErr := g.CreateBareRepo()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CreateBareRepo() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CreateBareRepo() succeeded unexpectedly")
			}

			// Ensure the repository exists
			_, err = os.Stat(g.FullPath)
			if errors.Is(err, fs.ErrNotExist) {
				t.Fatal("repository does not exist after creation")
			}
		})
	}
}
