# Security scanning

Run `./scripts/security-scan-local.sh` to scan this checkout locally. Semgrep Community Edition uses the free `p/default` community rules with metrics disabled and one worker by default to fit constrained hosts; set `SEMGREP_JOBS` to change the worker count. The source stays on the machine. The rule pack is fetched from the public Semgrep Registry when it is missing or refreshed, so the first scan needs network access. For a disconnected run, set `SEMGREP_CONFIG` to a local rules directory and `SEMGREP_OFFLINE=1` after caching the Semgrep package.

Semgrep runs on supported source and security configuration. It defaults to two worker jobs for memory-limited environments; set `SEMGREP_JOBS` to change that. Trivy selects `secret` for source/config, `vuln` when dependency manifests or lockfiles are present, and `misconfig` for container, workflow, Kubernetes, Helm, or Terraform files.

The installed pre-commit hook runs Semgrep on staged source/config files; Trivy runs during full local scans and any configured CI workflow.

Install the repository's local hooks with `./scripts/install-git-hooks.sh` to enable the staged Semgrep check.

A weekly public GitHub Actions job runs the current official Semgrep CE image and refreshes `p/default` rules; scan output is uploaded to GitHub code scanning.

The weekly Trivy filesystem job uses the current Trivy release and refreshes its vulnerability database.

Scans report findings without failing by default so existing findings can be triaged. Set `SEMGREP_FAIL_ON_FINDINGS=1` or `TRIVY_EXIT_CODE=1` to make a local scan fail on findings. Trivy downloads its vulnerability and check databases automatically; use `TRIVY_OFFLINE=1` only after those databases are cached.

Run `./scripts/update-security-tools.sh` to refresh the local Semgrep package cache, pull the current Semgrep fallback image, and refresh Trivy's database. Public scan workflows use weekly schedules to exercise current scanner releases and rules. Existing private runner workflows retain their current version and runner policy.
