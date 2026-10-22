from __future__ import annotations

import http.client
import importlib.util
import ssl
import sys
import unittest
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))

from safe_http import HTTPOnlyRedirectHandler, validate_http_url  # noqa: E402


def load_script(name: str, filename: str):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / filename)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load support script {filename}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


backfill = load_script("vod_backfill_series_catalog", "vod-backfill-series-catalog.py")
stream_overrides = load_script("plex_generate_stream_overrides", "plex-generate-stream-overrides.py")
label_proxy = load_script("plex_media_providers_label_proxy", "plex-media-providers-label-proxy.py")


class SafeHTTPURLTests(unittest.TestCase):
    def test_accepts_http_and_https_urls(self) -> None:
        self.assertEqual(validate_http_url("http://provider.example:8080/api").hostname, "provider.example")
        self.assertEqual(validate_http_url("https://provider.example/api").scheme, "https")

    def test_rejects_file_other_schemes_missing_hosts_and_credentials(self) -> None:
        for value in (
            "file:///etc/passwd",
            "ftp://provider.example/catalog.json",
            "https:///missing-host",
            "https://user:pass@provider.example/api",
            "https://provider.example:invalid/api",
        ):
            with self.subTest(value=value), self.assertRaises(ValueError):
                validate_http_url(value)

    def test_redirects_cannot_switch_to_file_or_downgrade_tls(self) -> None:
        handler = HTTPOnlyRedirectHandler()
        for current, target in (
            ("http://provider.example/api", "file:///etc/passwd"),
            ("https://provider.example/api", "http://provider.example/api"),
        ):
            request = urllib.request.Request(current)
            with self.subTest(current=current, target=target), self.assertRaises(urllib.error.URLError):
                handler.redirect_request(request, None, 302, "Found", {}, target)

    def test_support_scripts_reject_file_urls(self) -> None:
        for load in (
            lambda: backfill._load_json("file:///etc/passwd", timeout=1),
            lambda: stream_overrides.load_json("file:///etc/passwd"),
        ):
            with self.assertRaises(ValueError):
                load()

    def test_https_upstream_uses_default_certificate_verification(self) -> None:
        connection, _ = label_proxy.build_http_conn("https://provider.example")
        self.assertIsInstance(connection, http.client.HTTPSConnection)
        self.assertEqual(connection._context.verify_mode, ssl.CERT_REQUIRED)
        self.assertTrue(connection._context.check_hostname)
        connection.close()

    def test_plain_http_upstream_remains_supported(self) -> None:
        connection, parsed = label_proxy.build_http_conn("http://provider.example")
        self.assertIsInstance(connection, http.client.HTTPConnection)
        self.assertEqual(parsed.hostname, "provider.example")
        connection.close()

    def test_label_proxy_rejects_unsupported_schemes_and_embedded_credentials(self) -> None:
        for value in ("file:///etc/passwd", "https://user:pass@provider.example"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                label_proxy.build_http_conn(value)


if __name__ == "__main__":
    unittest.main()
