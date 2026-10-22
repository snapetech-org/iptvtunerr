"""HTTP(S)-only URL validation and redirects for operator support scripts."""

from __future__ import annotations

import urllib.error
import urllib.parse
import urllib.request


def validate_http_url(raw_url: str) -> urllib.parse.SplitResult:
    parsed = urllib.parse.urlsplit(raw_url)
    if parsed.scheme.lower() not in {"http", "https"} or not parsed.hostname:
        raise ValueError("URL must use HTTP or HTTPS and include a hostname")
    if parsed.username is not None or parsed.password is not None:
        raise ValueError("URL user information is not allowed")
    try:
        parsed.port
    except ValueError as exc:
        raise ValueError("URL port is invalid") from exc
    return parsed


class HTTPOnlyRedirectHandler(urllib.request.HTTPRedirectHandler):
    """Allow only HTTP(S) redirects and refuse HTTPS downgrade redirects."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        target_url = urllib.parse.urljoin(req.full_url, newurl)
        try:
            current = validate_http_url(req.full_url)
            target = validate_http_url(target_url)
        except ValueError as exc:
            raise urllib.error.URLError(str(exc)) from exc
        if current.scheme.lower() == "https" and target.scheme.lower() != "https":
            raise urllib.error.URLError("refusing HTTPS downgrade redirect")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def open_http_url(url: str, timeout: int, headers: dict[str, str] | None = None):
    validate_http_url(url)
    request = urllib.request.Request(url, headers=headers or {})
    opener = urllib.request.build_opener(HTTPOnlyRedirectHandler())
    return opener.open(request, timeout=timeout)
