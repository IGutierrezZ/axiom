# Windows test policy

This page covers how to run the Go tests on Windows and when CI runs the Windows tests.

## Running tests locally on Windows

- **Run focused packages, not the whole suite.** Use `go test ./internal/<pkg> -run '^TestName$' -count=1`. The full suite needs about ten CI shards of up to 30 minutes each, so a local run on Windows costs too much to be useful.
- **Symlink tests skip without privilege.** Creating a symlink on Windows needs a privilege that a normal user lacks (error 1314, `ERROR_PRIVILEGE_NOT_HELD`).
  - Tests that need symlinks call `testenv.RequireSymlink(t)` (`internal/testenv`), which probes once per process.
  - The test is skipped only when the probe fails with that error, and the skip message says so.
  - Any other probe failure fails the test.
  - To run these tests, enable Windows Developer Mode or use an elevated shell.
- **Keep `TEMP` outside `%LOCALAPPDATA%`.** Some tests fail when their temporary directory is under `...\AppData\Local\...`, which is the default:
  - creating a file with `O_EXCL` through a directory junction (`internal/components/filemerge`);
  - `os.Lstat` on an AF_UNIX socket (`internal/backup`).

  The same tests pass with a temporary directory elsewhere:

  ```powershell
  $env:TMP = "C:\tmp\go-test"; $env:TEMP = $env:TMP
  ```

- **Time budgets scale.** `AXIOM_TEST_TIMEOUT_SCALE` multiplies the test-side time budgets of `internal/sddstatus`, `e2e/organicruntime` and the catalog cancellation test in `internal/opencode`.
  - It must be a number of at least 1. Any other value aborts the package instead of shrinking a budget.
  - Unset, it is 2 on Windows and 1 elsewhere.
  - Production budgets never change.
  - Raise it on a loaded machine, for example `$env:AXIOM_TEST_TIMEOUT_SCALE = "3"`. Tests are never retried automatically.
- **`bench/` is Linux-only.** Its tests execute `#!/bin/sh` fakes and drive a TTY, so CI runs them only on Ubuntu.

## When CI runs the Windows tests

Windows tests do not run on every PR or merge. Linux runs the full suite on every PR. Windows runs when a change needs it, on request, or once a week.

| Workflow | Runs on |
|---|---|
| `ci.yml`, job `windows-runtime` | `workflow_dispatch`, plus any PR or push to `main` whose diff touches a Windows-relevant path, plus any PR carrying the `ci:windows` label. |
| `windows-full-suite.yml` | `workflow_dispatch`, plus the `ci:windows` label, plus a weekly schedule on Mondays at 03:20 UTC. |

The Windows-relevant paths are:

- `**/*_windows.go` and `**/*_windows_test.go`;
- `**/*_other.go` and `**/*_unix.go`;
- `scripts/*.ps1`;
- the two workflow files;
- `go.mod` and `go.sum`.

The `windows-runtime` job does not filter by package, because so many packages contain Windows-specific code that filtering by package would run almost every time.

- **Forcing Windows on a PR.** Add the `ci:windows` label.
  - A label added after the PR was opened starts only the full suite: `ci.yml` does not listen to `labeled` events, so labelling does not rerun all of CI.
  - `windows-runtime` picks the label up on the next push to the PR.
- **Weekly run.** The scheduled run first compares `main` with the commit of the last successful full-suite run. If nothing changed, it stops without starting a Windows runner. Until a full-suite run succeeds, it runs every week.
- **Failing open.** If the change detection cannot compute a diff (a new branch, a force push, a broken range), `windows-runtime` runs instead of being skipped silently.

The cost is that a change that breaks Windows without touching those paths is caught by the next weekly run, by a manual dispatch, or by the label.
