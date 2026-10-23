---
id: ref-ci-runner-routing
type: reference
status: stable
tags: [reference, ci, security]
---

# CI runner routing

The coordinated runner rollout adds hosted/JIT selection for Linux x64 CI,
release assets, Docker, and GitHub-release orchestration. The workflow changes
must be deployed with the shared broker and exact workflow-access catalog;
source changes alone do not activate the runner service.

`NG_HOSTED_FIRST=true` selects hosted Linux on attempt one. `NG_FORCE_JIT=true`
or a retry selects the shared group; leaving hosted-first unset preserves the
existing JIT-first default. Each JIT runner has repository and run labels.

The broker can recover an entirely unstarted hosted run after its queue deadline
or a startup rejection. It persists cancellation/retry intent and rechecks that
no runner or step executed. It does not replay a partially executed publisher.
Native platform jobs retain their compatible runners, and public PRs stay hosted.
SSH-based package publishing and additional package channels keep their existing
routes. The CI packaging job gets RPM, ZIP, and Debian tools during guest bootstrap,
preserving its existing no-sudo policy.

For release tags, the broker verifies the exact tag commit against the run SHA
and main ancestry, then grants only the reviewed workflows at that exact tag ref.
The runner group remains restricted to selected repositories and workflows.
GitHub reruns use the original workflow source; old tags do not acquire a new
runner selector just because main changed.

Runner-group selection and GitHub App installation are separate controls.
Keep this repository in both: the broker can discover queued workflows only
in repositories its App installation can access. A selected workflow without
App repository access is silently absent from queue discovery, so the broker
now requires every trusted-main route to be installed before it starts.

When adding a build or release workflow, keep the trusted event/ref condition and both
labels in its `runs-on` expression, then update the broker policy and runner
group workflow selection together. `./scripts/verify` remains the local
verification entrypoint; it does not depend on a shared runner.

See also
--------
- [Verification commands](../../memory-bank/commands.yml)
- [Runner pool design](https://github.com/snapetech-org/packer/blob/main/docs/application-runner-rollout.md)
- [Testing and supervisor config](testing-and-supervisor-config.md)
