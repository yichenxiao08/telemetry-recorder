"""Explicit, restricted public-data experiment. Not a privacy guarantee."""
import hashlib
import ipaddress
import json
import urllib.parse
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def prepare(endpoint, policy, profile, raw_policy, *, check_server=True):
    parsed = urllib.parse.urlsplit(endpoint)
    try:
        ip = ipaddress.ip_address(parsed.hostname or "")
        port = parsed.port
    except ValueError as exc:
        raise ValueError("public capture requires literal loopback HTTP endpoint") from exc
    if (parsed.scheme != "http" or not ip.is_loopback or port is None or
            parsed.username is not None or parsed.password is not None or
            parsed.path not in ("", "/") or parsed.query or parsed.fragment):
        raise ValueError("public capture requires literal loopback HTTP endpoint")
    fields = policy.profiles.get(profile, {}).get("fields", {})
    if not fields or any(rule.get("kind") != "number" or not
            (path.startswith("inputs.") or path == "output" or path.startswith("output."))
            for path, rule in fields.items()):
        raise ValueError("public capture requires numeric input/output allowlist")
    digest = hashlib.sha256(raw_policy).hexdigest()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    if check_server:
        try:
            with opener.open(endpoint.rstrip("/") + "/public-capture-policy", timeout=2) as response:
                content = response.read(4097)
                if response.status != 200 or len(content) > 4096:
                    raise ValueError("invalid public capture policy response")
                actual = json.loads(content)
        except Exception as exc:
            raise ValueError("public capture server policy unavailable") from exc
        expected = {"mode": "public-data-experiment", "profile": profile, "sha256": digest}
        if actual != expected:
            raise ValueError("public capture server policy mismatch")
    return digest, opener
