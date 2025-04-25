package git

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
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
			_, gotErr := NewRemoteRepository("", tt.orgName, tt.repoName)
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
			g, err := NewRemoteRepository(tt.baseURL, tt.orgName, tt.repoName)
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

func TestGitRepository_Command(t *testing.T) {
	tests := []struct {
		testName string // description of this test case
		// Named input parameters for target function.
		name       string
		arg        []string
		want       *exec.Cmd
		wantStdOut string
		wantStdErr string
	}{
		{
			testName:   "generic command",
			name:       "ls",
			arg:        []string{"-1"},
			want:       &exec.Cmd{},
			wantStdOut: "git.go\ngit_test.go\ntest_repositories\n",
			wantStdErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// Don't need to worry about creating an actual git repository while
			// just testing a generic command
			var g GitRepository
			gotCommand, gotStdOut, gotStdErr := g.Command(tt.name, tt.arg...)
			gotCommand.Run()

			if gotStdOut.String() != tt.wantStdOut {
				t.Errorf("Command.StdOut = %v, want %v", gotStdOut, tt.wantStdOut)
			}
			if gotStdErr.String() != tt.wantStdErr {
				t.Errorf("Command.StdErr = %v, want %v", gotStdErr, tt.wantStdErr)
			}
		})
	}
}

func TestGitRepository_GetFiles_Empty(t *testing.T) {
	g, err := NewRemoteRepository("", "test_org", "test_repo_for_getfiles")

	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	err = g.CreateBareRepo()
	if err != nil {
		t.Fatal(err)
	}
	defer g.DeleteRepo()

	gotFiles, gotErr := g.GetFiles("main")

	if gotErr == nil {
		t.Fatal("GetFiles() did not return error and should have")
	}

	if !errors.As(gotErr, &EmptyRepositoryError{}) {
		t.Errorf("GetFiles() did not return expected error: %v", gotErr)
	}

	if len(gotFiles) > 0 {
		t.Errorf("GetFiles() = %v, want empty slice", gotFiles)
	}

}
