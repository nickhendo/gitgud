package git

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

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
