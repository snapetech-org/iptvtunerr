import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
CI_WORKFLOW = (ROOT / ".github/workflows/ci.yml").read_text(encoding="utf-8")


def job_block(name: str) -> str:
    match = re.search(
        rf"(?ms)^  {re.escape(name)}:\n(.*?)(?=^  [A-Za-z0-9_-]+:\n|\Z)",
        CI_WORKFLOW,
    )
    if match is None:
        raise AssertionError(f"CI workflow is missing the {name!r} job")
    return match.group(1)


class CIRunnerRoutingTests(unittest.TestCase):
    def test_trusted_verification_jobs_use_run_scoped_jit_pool(self):
        expected = (
            "ng-shared-main-jit",
            "ng-shared-main-iptvtunerr",
            "ng-main-run-{0}",
            "contains(fromJSON('[\"push\",\"workflow_dispatch\",\"schedule\",\"release\",\"repository_dispatch\"]'), github.event_name)",
            "github.ref == 'refs/heads/main'",
            "github.ref_type == 'tag'",
            "startsWith(github.ref_name, 'v')",
            "|| 'ubuntu-latest'",
        )
        for name in ("verify", "smoke"):
            with self.subTest(job=name):
                block = job_block(name)
                for token in expected:
                    self.assertIn(token, block)

    def test_package_asset_contract_uses_trusted_runner_route_with_hosted_fallback(self):
        block = job_block("release-assets")
        for token in (
            "ng-shared-main-jit",
            "ng-shared-main-iptvtunerr",
            "ng-main-run-{0}",
            "github.ref == 'refs/heads/main'",
            "|| 'ubuntu-latest'",
        ):
            self.assertIn(token, block)
        self.assertIn(
            "bash scripts/install-ci-tools.sh rpm zip debian-packaging", block
        )


if __name__ == "__main__":
    unittest.main()
