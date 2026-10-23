---
id: ref-ci-runner-routing
type: reference
status: stable
tags: [reference, ci, security]
---

# CI runner routing

IPTV Tunerr uses the organization shared JIT runner group for trusted
main-branch verification and security scans: CI, CodeQL, Gitleaks, the trusted
Local Identity scan, Semgrep, and Trivy. Each job carries a repository label
and a workflow-run label so a runner can accept only work from its assigned
run.

Pull-request jobs continue to use GitHub-hosted runners. The shared group is
selected only for pushes and manual dispatches on `main`; credentialed release
and package-publishing jobs also stay on GitHub-hosted runners. The broker and
runner group separately allowlist this repository and the exact workflow
files, and the broker accepts only the current `main` commit.

Runner-group selection and GitHub App installation are separate controls.
Keep this repository in both: the broker can discover queued workflows only
in repositories its App installation can access. A selected workflow without
App repository access is silently absent from queue discovery, so the broker
now requires every trusted-main route to be installed before it starts.

When adding a trusted-main workflow, keep the branch/event condition and both
labels in its `runs-on` expression, then update the broker policy and runner
group workflow selection together. `./scripts/verify` remains the local
verification entrypoint; it does not depend on a shared runner.

See also
--------
- [Verification commands](../../memory-bank/commands.yml)
- [Runner pool design](https://github.com/snapetech/packer/blob/main/docs/linux-runner-pool.md)
- [Testing and supervisor config](testing-and-supervisor-config.md)
