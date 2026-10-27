---
id: security-scanning
type: how-to
status: active
tags: [security, tooling]
---

# Security scanning

Run `./scripts/security-scan-local.sh` to scan this checkout locally with Semgrep and Trivy. Semgrep Community Edition uses the free `p/default` community rules with metrics disabled and one worker by default to fit constrained hosts; set `SEMGREP_JOBS` to change the worker count. The wrapper selects Semgrep's POSIX backend by default for environments where `io_uring` is unavailable; set `SEMGREP_EIO_BACKEND` to override it. The source stays on the machine. The rule pack is fetched from the public Semgrep Registry when it is missing or refreshed, so the first scan needs network access. For a disconnected run, set `SEMGREP_CONFIG` to a local rules directory and `SEMGREP_OFFLINE=1` after caching the Semgrep package. Semgrep documents this backend workaround in [issue 11836](https://github.com/semgrep/semgrep/issues/11836).

Semgrep runs on supported source and security configuration. It defaults to one worker job for memory-limited environments; set `SEMGREP_JOBS` to change that. Trivy selects `secret` for source/config, `vuln` when dependency manifests or lockfiles are present, and `misconfig` for container, workflow, Kubernetes, Helm, or Terraform files.

The installed pre-commit hook runs Semgrep on staged source/config files; Trivy runs during full local scans and any configured CI workflow.

Install the repository's local hooks with `./scripts/install-git-hooks.sh` to enable the staged Semgrep check.

Dispatch the Semgrep workflow with `action_tag=semgrep` and `runner_pool_tag=private` to run the current Semgrep CE image and refresh `p/default` rules; scan output is uploaded to GitHub code scanning.

Dispatch the Trivy workflow with `action_tag=trivy` and `runner_pool_tag=private` to use the current Trivy release and refresh its vulnerability database.

The local identity leak check uses Python's standard library for case-insensitive fixed-string matching. It needs no jq, ripgrep, or privileged package installation, so the trusted scanner runs on shared JIT workers. Files over 16 MiB fail closed. Dispatch it from `main` with `action_tag=local-identity-scan` and `runner_pool_tag=private` to scan the selected commit.

Scans report findings without failing by default so existing findings can be triaged. Set `SEMGREP_FAIL_ON_FINDINGS=1` or `TRIVY_EXIT_CODE=1` to make a local scan fail on findings. Trivy downloads its vulnerability and check databases automatically; use `TRIVY_OFFLINE=1` only after those databases are cached. The default timeout is 20 minutes for slow first downloads; set `TRIVY_TIMEOUT` to change it.

Run `./scripts/update-security-tools.sh` to refresh the local Semgrep package cache, pull the current Semgrep image, and refresh Trivy's database. Scan workflows run only after an explicit action and private runner-pool selection.

See also
--------
- [Release channels](how-to/release-channels.md)
