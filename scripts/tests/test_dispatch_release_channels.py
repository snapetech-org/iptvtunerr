import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
DISPATCHER = ROOT / "scripts" / "dispatch-release-channels.sh"
CHANNEL_TITLES = {
    "docker.yml": "Publish Docker",
    "release-aur.yml": "Publish AUR",
    "release-ppa.yml": "Publish PPA",
    "release-copr.yml": "Publish COPR",
    "publish-chocolatey.yml": "Publish Chocolatey",
    "publish-winget.yml": "Publish Winget",
    "publish-snap.yml": "Publish Snap",
}


class ReleaseChannelDispatcherTests(unittest.TestCase):
    def run_dispatcher(self, skip_channels=None):
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
                            "docker.yml": "Publish Docker",
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
                        print("completed\\tsuccess", end="")
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


if __name__ == "__main__":
    unittest.main()
