import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SCANNER = ROOT / "scripts" / "check-local-identity-leaks.sh"


@unittest.skipUnless(
    shutil.which("bash") and shutil.which("git") and shutil.which("rg") and shutil.which("jq"),
    "scanner dependencies unavailable",
)
class LocalIdentityScanRangeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="iptvtunerr-identity-scan-")
        self.repo = Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "Fixture User")
        self.git("config", "user.email", "fixture@example.invalid")
        (self.repo / "README.md").write_text("fixture\n", encoding="utf-8")
        self.git("add", "README.md")
        self.git("commit", "-q", "-m", "baseline")
        self.baseline = self.git("rev-parse", "HEAD")
        self.git("commit", "-q", "--allow-empty", "-m", "ScanMarkerFixture historical commit")
        self.marker_commit = self.git("rev-parse", "HEAD")
        self.git("commit", "-q", "--allow-empty", "-m", "clean follow-up")

    def tearDown(self):
        self.temp.cleanup()

    def git(self, *args):
        result = subprocess.run(
            ["git", *args],
            cwd=self.repo,
            check=True,
            capture_output=True,
            text=True,
        )
        return result.stdout.strip()

    def scan(self, **overrides):
        env = os.environ.copy()
        for name in (
            "LOCAL_IDENTITY_DENYLIST_FILE",
            "SLSKDN_LOCAL_IDENTITY_DENYLIST",
            "SLSKDN_FORBIDDEN_LOCAL_HOSTNAME",
            "SLSKDN_LOCAL_IDENTITY_DENYLIST_FILE",
        ):
            env.pop(name, None)
        env.update(
            {
                "LOCAL_IDENTITY_DENYLIST": "ScanMarkerFixture",
                "LOCAL_IDENTITY_SCAN_ROOT": str(self.repo),
                "LOCAL_IDENTITY_SCAN_COMMITS": "1",
                **overrides,
            }
        )
        return subprocess.run(
            ["bash", str(SCANNER)],
            cwd=self.repo,
            env=env,
            capture_output=True,
            text=True,
        )

    def test_base_scans_only_commits_after_the_push_base(self):
        result = self.scan(LOCAL_IDENTITY_SCAN_BASE=self.marker_commit)
        self.assertEqual(result.returncode, 0, result.stderr)

        result = self.scan(LOCAL_IDENTITY_SCAN_BASE=self.baseline)
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertIn("recent commit messages", result.stderr)

    def test_commit_count_scans_the_requested_newest_commits(self):
        result = self.scan(LOCAL_IDENTITY_SCAN_COMMIT_COUNT="1")
        self.assertEqual(result.returncode, 0, result.stderr)

        result = self.scan(LOCAL_IDENTITY_SCAN_COMMIT_COUNT="2")
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)

    def test_invalid_range_fails_closed(self):
        result = self.scan(LOCAL_IDENTITY_SCAN_BASE="deadbeef")
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)

        result = self.scan(LOCAL_IDENTITY_SCAN_COMMIT_COUNT="0")
        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
