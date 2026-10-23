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
            "github.event_name == 'push'",
            "github.ref == 'refs/heads/main'",
            "|| 'ubuntu-latest'",
        )
        for name in ("verify", "smoke"):
            with self.subTest(job=name):
                block = job_block(name)
                for token in expected:
                    self.assertIn(token, block)

    def test_package_asset_contract_runs_on_hosted_linux(self):
        block = job_block("release-assets")
        self.assertIn("runs-on: ubuntu-latest", block)
        self.assertIn(
            "bash scripts/install-ci-tools.sh rpm zip debian-packaging", block
        )
        self.assertNotIn("ng-shared-main-jit", block)


if __name__ == "__main__":
    unittest.main()
