package reviewtransaction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func TestPrepareReviewRepositoryRootInitializesOnlyGenuinelyUnversionedWorkspace(t *testing.T) {
	t.Run("initializes an unversioned workspace", func(t *testing.T) {
		repo := t.TempDir()
		candidate := filepath.Join(repo, "candidate.txt")
		if err := os.WriteFile(candidate, []byte("candidate\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		root, err := PrepareReviewRepositoryRoot(context.Background(), repo)
		if err != nil {
			t.Fatalf("prepare local Git repository: %v", err)
		}
		// Identity, not string equality: on Windows the runner's TEMP is an
		// 8.3 short name that Git reports back in its long spelling.
		if !pathidentity.SameDirectory(root, repo) {
			t.Fatalf("prepared root = %q, want %q", root, repo)
		}
		if info, statErr := os.Stat(filepath.Join(repo, ".git")); statErr != nil || !info.IsDir() {
			t.Fatalf("local Git metadata = %v (%v), want directory", info, statErr)
		}
		if content, readErr := os.ReadFile(candidate); readErr != nil || string(content) != "candidate\n" {
			t.Fatalf("candidate after bootstrap = %q, %v", content, readErr)
		}
	})

	t.Run("reuses a containing worktree", func(t *testing.T) {
		repo := initUnbornSnapshotRepo(t)
		nested := filepath.Join(repo, "nested", "workspace")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}

		root, err := PrepareReviewRepositoryRoot(context.Background(), nested)
		if err != nil {
			t.Fatalf("prepare containing repository: %v", err)
		}
		if !pathidentity.SameDirectory(root, repo) {
			t.Fatalf("prepared root = %q, want containing root %q", root, repo)
		}
		if _, statErr := os.Lstat(filepath.Join(nested, ".git")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("nested metadata = %v, want no nested .git", statErr)
		}
	})

	t.Run("refuses unusable metadata without modifying it", func(t *testing.T) {
		repo := t.TempDir()
		metadata := filepath.Join(repo, ".git")
		before := []byte("not a git directory\n")
		if err := os.WriteFile(metadata, before, 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := PrepareReviewRepositoryRoot(context.Background(), repo); err == nil {
			t.Fatal("prepare accepted unusable Git metadata")
		}
		after, err := os.ReadFile(metadata)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(after, before) {
			t.Fatalf("Git metadata changed:\nbefore: %q\nafter:  %q", before, after)
		}
	})
}

func TestSnapshotBuilderResolveRepositoryRootDoesNotBootstrap(t *testing.T) {
	repo := t.TempDir()
	if _, err := (SnapshotBuilder{Repo: repo}).ResolveRepositoryRoot(context.Background()); err == nil {
		t.Fatal("ResolveRepositoryRoot accepted an unversioned workspace")
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ResolveRepositoryRoot created metadata: %v", err)
	}
}

func TestPrepareReviewRepositoryRootRefusesMultirepoWorkspaceWithoutWriting(t *testing.T) {
	mkdir := func(t *testing.T, path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(t *testing.T, path, content string) {
		t.Helper()
		mkdir(t, filepath.Dir(path))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name             string
		build            func(t *testing.T) (dir string)
		wantPointer      bool
		wantRepositories []string
	}{
		{
			name: "pointer in the workspace itself",
			build: func(t *testing.T) string {
				dir := t.TempDir()
				write(t, filepath.Join(dir, ".axiom-workspace"), "planning_home: .\n")
				return dir
			},
			wantPointer: true,
		},
		{
			name: "pointer only in an ancestor",
			build: func(t *testing.T) string {
				master := t.TempDir()
				write(t, filepath.Join(master, ".axiom-workspace"), "planning_home: .\n")
				dir := filepath.Join(master, "docs")
				mkdir(t, dir)
				return dir
			},
			wantPointer: true,
		},
		{
			name: "child repository whose .git is a directory",
			build: func(t *testing.T) string {
				dir := t.TempDir()
				mkdir(t, filepath.Join(dir, "api", ".git"))
				return dir
			},
			wantRepositories: []string{"api"},
		},
		{
			name: "child repository whose .git is a file",
			build: func(t *testing.T) string {
				dir := t.TempDir()
				write(t, filepath.Join(dir, "web", ".git"), "gitdir: ../.bare/web\n")
				return dir
			},
			wantRepositories: []string{"web"},
		},
		{
			name: "pointer and several child repositories among plain folders",
			build: func(t *testing.T) string {
				dir := t.TempDir()
				write(t, filepath.Join(dir, ".axiom-workspace"), "planning_home: .\n")
				mkdir(t, filepath.Join(dir, "api", ".git"))
				write(t, filepath.Join(dir, "web", ".git"), "gitdir: ../.bare/web\n")
				mkdir(t, filepath.Join(dir, "plain-folder"))
				return dir
			},
			wantPointer:      true,
			wantRepositories: []string{"api", "web"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := tt.build(t)

			root, err := PrepareReviewRepositoryRoot(context.Background(), workspacePath)
			if err == nil {
				t.Fatalf("prepare accepted a multirepo workspace and returned root %q", root)
			}
			var refusal *ReviewMultirepoWorkspaceError
			if !errors.As(err, &refusal) {
				t.Fatalf("prepare error = %v (%T), want *ReviewMultirepoWorkspaceError", err, err)
			}
			if root != "" {
				t.Fatalf("refused prepare returned root %q, want empty", root)
			}
			if _, statErr := os.Lstat(filepath.Join(workspacePath, ".git")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("refusal left workspace metadata behind: %v", statErr)
			}

			if !pathidentity.SameDirectory(refusal.Workspace, workspacePath) {
				t.Fatalf("refusal workspace = %q, want %q", refusal.Workspace, workspacePath)
			}
			if tt.wantPointer != (refusal.Pointer != "") {
				t.Fatalf("refusal pointer = %q, want a marker: %v", refusal.Pointer, tt.wantPointer)
			}
			if refusal.Pointer != "" && filepath.Base(refusal.Pointer) != workspace.WorkspacePointerFilename {
				t.Fatalf("refusal pointer = %q, want a %s entry", refusal.Pointer, workspace.WorkspacePointerFilename)
			}
			if len(refusal.Repositories) != len(tt.wantRepositories) {
				t.Fatalf("refusal repositories = %q, want %q", refusal.Repositories, tt.wantRepositories)
			}
			for index, name := range tt.wantRepositories {
				if filepath.Base(refusal.Repositories[index]) != name {
					t.Fatalf("refusal repositories = %q, want %q", refusal.Repositories, tt.wantRepositories)
				}
			}

			message := err.Error()
			for _, want := range []string{pathquote.Quote(refusal.Workspace), "multirepo workspace", "axiom review status --cwd <repo>"} {
				if !strings.Contains(message, want) {
					t.Fatalf("refusal = %q, want it to contain %q", message, want)
				}
			}
			if refusal.Pointer != "" && !strings.Contains(message, pathquote.Quote(refusal.Pointer)) {
				t.Fatalf("refusal = %q, want it to name the marker %q", message, refusal.Pointer)
			}
			for _, repository := range refusal.Repositories {
				if !strings.Contains(message, pathquote.Quote(repository)) {
					t.Fatalf("refusal = %q, want it to name repository %q", message, repository)
				}
			}
		})
	}
}

func TestPrepareReviewRepositoryRootSummarizesLargeMultirepoWorkspace(t *testing.T) {
	workspacePath := t.TempDir()
	const repositories = reviewMultirepoListedRepositories + 3
	for index := 0; index < repositories; index++ {
		if err := os.MkdirAll(filepath.Join(workspacePath, fmt.Sprintf("repo-%02d", index), ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_, err := PrepareReviewRepositoryRoot(context.Background(), workspacePath)
	var refusal *ReviewMultirepoWorkspaceError
	if !errors.As(err, &refusal) {
		t.Fatalf("prepare error = %v, want *ReviewMultirepoWorkspaceError", err)
	}
	if len(refusal.Repositories) != repositories {
		t.Fatalf("refusal repositories = %d, want every one of the %d recorded", len(refusal.Repositories), repositories)
	}
	message := err.Error()
	named := 0
	for _, repository := range refusal.Repositories {
		if strings.Contains(message, pathquote.Quote(repository)) {
			named++
		}
	}
	if named != reviewMultirepoListedRepositories {
		t.Fatalf("refusal names %d repositories, want the first %d only: %q", named, reviewMultirepoListedRepositories, message)
	}
	if !strings.Contains(message, "and 3 more") {
		t.Fatalf("refusal = %q, want the omitted repositories counted", message)
	}
}

func TestPrepareReviewRepositoryRootStillInitializesOrdinaryDirectoryWithSubfolders(t *testing.T) {
	workspacePath := t.TempDir()
	for _, name := range []string{"docs", "scripts"} {
		if err := os.MkdirAll(filepath.Join(workspacePath, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"README.md", filepath.Join("docs", "notes.md")} {
		if err := os.WriteFile(filepath.Join(workspacePath, name), []byte("text\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	root, err := PrepareReviewRepositoryRoot(context.Background(), workspacePath)
	if err != nil {
		t.Fatalf("prepare ordinary unversioned directory: %v", err)
	}
	if !pathidentity.SameDirectory(root, workspacePath) {
		t.Fatalf("prepared root = %q, want %q", root, workspacePath)
	}
	if info, statErr := os.Stat(filepath.Join(workspacePath, ".git")); statErr != nil || !info.IsDir() {
		t.Fatalf("local Git metadata = %v (%v), want directory", info, statErr)
	}
}

// The marker name is spelled twice because this package does not import
// internal/workspace in production code; the test keeps the two spellings one
// marker.
func TestReviewWorkspacePointerFilenameMatchesWorkspacePackage(t *testing.T) {
	if reviewWorkspacePointerFilename != workspace.WorkspacePointerFilename {
		t.Fatalf("reviewWorkspacePointerFilename = %q, workspace.WorkspacePointerFilename = %q", reviewWorkspacePointerFilename, workspace.WorkspacePointerFilename)
	}
}
