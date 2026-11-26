# Windows test policy

This page covers how to run the Go tests on Windows and when CI runs the Windows tests.

## Running tests locally on Windows

- **Run focused packages while iterating.** Use `go test ./internal/<pkg> -run '^TestName$' -count=1`.
- **The whole suite except `internal/cli` finishes locally.** Measured on 2026-11-25: 118 packages in about 9 minutes of wall time, with `internal/branch/engine` the slowest at about 8.5 minutes, so pass an explicit `-timeout`:

  ```powershell
  go test (go list ./... | Where-Object { $_ -notmatch '/internal/cli$' }) -count=1 -timeout 20m
  ```

  `internal/cli` exceeds the default 10-minute timeout: run it on its own with an explicit `-timeout`, or split it with `-run` (the Windows full-suite workflow uses four selectors: `^Test[A-M]`, `^Test[N-O]`, `^Test[P-Q]|^Test[S-Z]` and `^TestR`).
- **Symlink tests skip without privilege.** Creating a symlink on Windows needs a privilege that a normal user lacks (error 1314, `ERROR_PRIVILEGE_NOT_HELD`).
  - Tests that need symlinks call `testenv.RequireSymlink(t)` (`internal/testenv`), which probes once per process.
  - The test is skipped only when the probe fails with that error, and the skip message says so.
  - Any other probe failure fails the test.
  - To run these tests, enable Windows Developer Mode or use an elevated shell.
- **Some tests skip under a protected `%TEMP%`.** On machines whose endpoint security protects `...\AppData\Local\Temp` (the default `TEMP`), two operations fail there and pass in any other folder, including the profile root:
  - creating a file with `O_EXCL` through a directory junction (`internal/components/filemerge`): the junction tests probe it first and skip with the reason;
  - `os.Lstat` or removal of an AF_UNIX socket file (`internal/backup`): it cannot be probed without leaving an undeletable file, so on Windows `TestSnapshotterSkipsUnixSockets` runs only when `CI` is set.

  To run them locally, point the temporary directory elsewhere:

  ```powershell
  $env:TMP = "C:\tmp\go-test"; $env:TEMP = $env:TMP
  ```

- **Tests must not read the checkout's project state.** A developer checkout can hold an ignored `.axiom/state.json`, and `cli.EffectiveStateStore` resolves from the working directory, so a test that resolves the effective store from a temporary home first calls `t.Chdir(t.TempDir())`.

- **Time budgets scale.** `AXIOM_TEST_TIMEOUT_SCALE` multiplies the test-side time budgets of `internal/sddstatus`, `e2e/organicruntime` and the catalog cancellation test in `internal/opencode`.
  - It must be a number of at least 1. Any other value aborts the package instead of shrinking a budget.
  - Unset, it is 2 on Windows and 1 elsewhere.
  - Production budgets never change.
  - Raise it on a loaded machine, for example `$env:AXIOM_TEST_TIMEOUT_SCALE = "3"`. Tests are never retried automatically.
- **Tests that build binaries skip unless you opt in.** Corporate EDR tools (Cortex XDR) block and delete the unsigned Go binaries that some tests compile into `%TEMP%` and run, and every local run raises a security alert.
  - Those tests call `testenv.RequireBinaryBuilds(t)` before any `go build` and skip with a message naming `AXIOM_TEST_BUILD_BINARIES`.
  - The variable must be `1` or `true`. It is not implied by `CI`: the workflows set `AXIOM_TEST_BUILD_BINARIES: "1"` explicitly, so CI runs them all.
  - On a machine without such an EDR, opt in with `$env:AXIOM_TEST_BUILD_BINARIES = "1"`.
  - Affected: `internal/app`, `internal/update`, `internal/opencode` and `e2e/organicruntime` (its `TestMain` builds nothing without the variable).
- **Tests that launch the Go toolchain skip behind the same variable.** They call `testenv.RequireGoToolchain(t)` before the first `go build`, `go run`, `go test -list` or `go version` subprocess, because each costs minutes on Windows. CI runs them because the workflows set the variable.
  - `internal/update`: `TestCanonicalReleasePublicKeysControlRealLinkerBuild` and the release distribution policy tests (`newReleasePolicyFixture`, which runs `go run ./internal/releasepolicycmd`).
  - `internal/app`: `TestDocumentedInvocationsRunAsDocumented`.
  - `internal/assets`: `TestWindowsFullSuiteShardsCoverEveryTestName` (nested `go test -list`).
- **In `ci.yml` the toolchain meta-tests run in their own job.** The workflow-level `TOOLCHAIN_META_TESTS` selector names them, together with `TestBuiltBinaryClosedStdinRefusesBeforeBubbleTea` (`internal/app`, `go build`). The `Go Tests` jobs skip it (`-skip`) and the `Toolchain Meta Tests` job runs it (`-run`).
  - That job runs on the nightly run, on `workflow_dispatch`, and on PRs that touch its inputs: the workflows, `.goreleaser.yaml`, `internal/update/`, `scripts/`, `cmd/axiom/`, the command documentation (`docs/**/*.md` except `docs/audits/`, `internal/assets/**/*.md`), `go.mod`, `go.sum`, and the meta-test files themselves.
  - `Unit Tests` accepts the job as skipped only when the change detection succeeded and found none of those inputs.
  - `internal/assets` fails if a selected name is declared outside the job's packages, or if a test in those packages launches the toolchain without being selected.

## When CI runs the Windows tests

Windows tests do not run on every PR or merge. Linux runs the full suite on every PR, except the toolchain meta-tests above when the PR does not touch their inputs. Windows runs when a change needs it, on request, or once a week.

A merge to `main` does not start `ci.yml`: the PR already ran it. `ci.yml` runs on `main` once a night at 02:40 UTC, and only when `main` has commits that no successful run has covered; otherwise every job is skipped. It also runs on `workflow_dispatch`.

| Workflow | Runs on |
|---|---|
| `ci.yml`, job `windows-runtime` | `workflow_dispatch`, plus any PR whose diff touches a Windows-relevant path, plus the nightly run on `main` when its new commits touch one, plus any PR carrying the `ci:windows` label. |
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
