import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
DISPATCHER = ROOT / "scripts" / "dispatch-release-channels.sh"
RELEASE_WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"
TESTER_BUNDLES_WORKFLOW = ROOT / ".github" / "workflows" / "tester-bundles.yml"
CHANNEL_TITLES = {
    "release-aur.yml": "Publish AUR",
    "release-ppa.yml": "Publish PPA",
    "release-copr.yml": "Publish COPR",
    "publish-chocolatey.yml": "Publish Chocolatey",
    "publish-winget.yml": "Publish Winget",
    "publish-snap.yml": "Publish Snap",
}


class ReleaseChannelDispatcherTests(unittest.TestCase):
    def run_dispatcher(
        self,
        skip_channels=None,
        run_status="completed",
        run_conclusion="success",
        job_status="completed",
        job_conclusion="success",
    ):
        with tempfile.TemporaryDirectory(prefix="iptvtunerr-release-dispatch-") as temp_dir:
            temp = Path(temp_dir)
            fake_bin = temp / "bin"
            fake_bin.mkdir()
            calls_file = temp / "workflow-calls.txt"
            fake_gh = fake_bin / "gh"
            fake_gh.write_text(
                textwrap.dedent(
                    """\
                    #!/usr/bin/env python3
                    import json
                    import os
                    import sys

                    args = sys.argv[1:]
                    if args[:2] == ["workflow", "run"]:
                        with open(os.environ["GH_TEST_CALLS"], "a", encoding="utf-8") as calls:
                            calls.write(args[2] + "\\n")
                    elif args[:2] == ["run", "list"]:
                        workflow = args[args.index("--workflow") + 1]
                        titles = {
                            "release-aur.yml": "Publish AUR",
                            "release-ppa.yml": "Publish PPA",
                            "release-copr.yml": "Publish COPR",
                            "publish-chocolatey.yml": "Publish Chocolatey",
                            "publish-winget.yml": "Publish Winget",
                            "publish-snap.yml": "Publish Snap",
                        }
                        print(json.dumps([{
                            "databaseId": 9876,
                            "displayTitle": titles[workflow] + " v0.1.95",
                            "createdAt": "9999-12-31T23:59:59Z",
                            "event": "workflow_dispatch",
                        }]))
                    elif args and args[0] == "api":
                        if args[1].endswith("/jobs?per_page=100"):
                            print(json.dumps({
                                "total_count": 1,
                                "jobs": [{
                                    "status": os.environ["GH_TEST_JOB_STATUS"],
                                    "conclusion": os.environ["GH_TEST_JOB_CONCLUSION"],
                                }],
                            }))
                        else:
                            print(
                                os.environ["GH_TEST_RUN_STATUS"] + "\\t" +
                                os.environ["GH_TEST_RUN_CONCLUSION"],
                                end="",
                            )
                    else:
                        print("unexpected gh invocation: " + repr(args), file=sys.stderr)
                        sys.exit(2)
                    """
                ),
                encoding="utf-8",
            )
            fake_gh.chmod(0o755)

            env = os.environ.copy()
            env["PATH"] = os.pathsep.join((str(fake_bin), env.get("PATH", "")))
            env["GH_TEST_CALLS"] = str(calls_file)
            env["GH_TEST_RUN_STATUS"] = run_status
            env["GH_TEST_RUN_CONCLUSION"] = run_conclusion
            env["GH_TEST_JOB_STATUS"] = job_status
            env["GH_TEST_JOB_CONCLUSION"] = job_conclusion
            env.pop("SKIP_RELEASE_CHANNELS", None)
            if skip_channels is not None:
                env["SKIP_RELEASE_CHANNELS"] = skip_channels

            result = subprocess.run(
                ["bash", str(DISPATCHER), "v0.1.95", "snapetech-org/iptvtunerr"],
                cwd=ROOT,
                env=env,
                capture_output=True,
                text=True,
                timeout=60,
                check=False,
            )
            calls = calls_file.read_text(encoding="utf-8").splitlines() if calls_file.exists() else []
            return result, calls

    def test_unset_skip_setting_dispatches_every_channel(self):
        result, calls = self.run_dispatcher()

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertCountEqual(calls, CHANNEL_TITLES)
        self.assertIn("publish-snap.yml run 9876 completed successfully", result.stdout)

    def test_explicit_deferred_channel_is_skipped(self):
        result, calls = self.run_dispatcher(" release-ppa.yml , publish-snap.yml ")

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertCountEqual(calls, set(CHANNEL_TITLES) - {"release-ppa.yml", "publish-snap.yml"})
        self.assertIn("Skipping deferred release channel release-ppa.yml", result.stdout)
        self.assertIn("Skipping deferred release channel publish-snap.yml", result.stdout)

    def test_stale_run_summary_uses_completed_job_results(self):
        result, calls = self.run_dispatcher(run_status="in_progress")

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertCountEqual(calls, CHANNEL_TITLES)
        self.assertIn("all job results are terminal", result.stderr)

    def test_stale_run_summary_does_not_hide_a_failed_job(self):
        result, calls = self.run_dispatcher(run_status="in_progress", job_conclusion="failure")

        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertCountEqual(calls, CHANNEL_TITLES)
        self.assertIn("completed with conclusion 'failure'", result.stderr)


class VersionTagReleaseWorkflowTests(unittest.TestCase):
    def test_tester_assets_are_built_only_by_the_pushed_tag_release(self):
        release = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        tester = TESTER_BUNDLES_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn("tags: ['v*']", release)
        self.assertIn("uses: ./.github/workflows/tester-bundles.yml", release)
        self.assertIn("needs: [release]", release)
        self.assertNotIn("secrets: inherit", release)
        self.assertIn("GHCR_TOKEN: ${{ secrets.GHCR_TOKEN }}", release)
        self.assertIn("workflow_call:", tester)
        self.assertNotIn("\n  workflow_dispatch:", tester)
        self.assertIn('[[ "$GITHUB_EVENT_NAME" == push', tester)
        self.assertIn('"$GITHUB_REF_NAME" == "$TAG"', tester)
        self.assertIn("git merge-base --is-ancestor", tester)
        self.assertIn("actions/download-artifact@", tester)
        self.assertIn("actions/upload-artifact@", tester)
        self.assertIn("gh release view", tester)
        self.assertIn("softprops/action-gh-release@", tester)


if __name__ == "__main__":
    unittest.main()
