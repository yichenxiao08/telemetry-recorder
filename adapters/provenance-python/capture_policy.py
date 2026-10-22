"""Opt-in synthetic P0 policy. No arbitrary text capture and no privacy guarantee for legacy mode."""
import hashlib
import hmac
import json
import math
import re

SENSITIVE = re.compile(r"password|credential|authorization|api.?key|access.?token|secret|cookie|bearer|private.?key", re.I)
CREDENTIAL = re.compile(r"bearer\s|sk[-_]|token[=:]|password[=:]|\b(select|insert|update|delete)\b|\d{11,19}", re.I)
EMAIL = re.compile(r"[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}", re.I)
VERSION = re.compile(r"[a-zA-Z0-9._-]{1,32}")
FIELD = re.compile(r"[A-Za-z_][A-Za-z0-9_.-]{0,255}")


def unique(pairs):
    result, seen = {}, set()
    for key, value in pairs:
        if key.lower() in seen:
            raise ValueError("duplicate JSON key")
        seen.add(key.lower())
        result[key] = value
    return result


class Policy:
    def __init__(self, data, key=b""):
        if type(data) is not dict or set(data) - {"version", "key_version", "profiles"} or type(data.get("version")) is not int or data.get("version") != 1:
            raise ValueError("invalid policy version/keys")
        self.profiles, self.key, self.version = data.get("profiles", {}), bytes(key), data.get("key_version", "")
        if not 1 <= len(self.profiles) <= 32:
            raise ValueError("bounded profiles required")
        for name, profile in self.profiles.items():
            if not VERSION.fullmatch(name) or set(profile) != {"fields"} or len(profile["fields"]) > 128:
                raise ValueError("invalid profile")
            for path, rule in profile["fields"].items():
                if not FIELD.fullmatch(path) or SENSITIVE.search(path) or set(rule) - {"kind", "values", "equality", "value_class", "nonpersonal", "hash_basis"}:
                    raise ValueError("invalid/credential field")
                kind = rule.get("kind")
                if kind in ("number", "boolean"):
                    if rule.get("values") or rule.get("equality"):
                        raise ValueError("invalid scalar rule")
                elif kind in ("enum", "artifact-sha256"):
                    if kind == "artifact-sha256" and (rule.get("nonpersonal") is not True or rule.get("hash_basis") != "redacted"):
                        raise ValueError("artifact requires nonpersonal approval/redacted basis")
                    values = rule.get("values", [])
                    if type(values) is not list or not 1 <= len(values) <= 32 or rule.get("equality"):
                        raise ValueError("finite enum required")
                    if any(type(v) is not str or len(v) > 128 or CREDENTIAL.search(v) or EMAIL.search(v) or any(c in v for c in "/?@'\"\\") for v in values):
                        raise ValueError("unsafe enum")
                elif kind == "hmac":
                    if rule.get("equality") is not True or rule.get("value_class") != "email" or rule.get("values") or len(key) < 32 or not VERSION.fullmatch(self.version):
                        raise ValueError("HMAC equality/runtime key/version required")
                else:
                    raise ValueError("unknown rule")

    @classmethod
    def load(cls, path, key=b""):
        with open(path, "rb") as stream:
            data = stream.read(65537)
        if len(data) > 65536:
            raise ValueError("policy too large")
        return cls(json.loads(data, object_pairs_hook=unique), key)

    def value(self, profile, path, value):
        if SENSITIVE.search(path):
            return None, False
        rule = self.profiles.get(profile, {}).get("fields", {}).get(path)
        if rule is None:
            if type(value) is dict:
                result = {}
                for key, item in value.items():
                    clean, ok = self.value(profile, path + "." + key, item)
                    if ok:
                        result[key] = clean
                return result, bool(result)
            return None, False
        kind = rule["kind"]
        if kind == "number":
            return (value, True) if type(value) in (int, float) and math.isfinite(value) and abs(value) < 1e10 else (None, False)
        if kind == "boolean":
            return (value, True) if type(value) is bool else (None, False)
        if kind == "enum":
            return (value, True) if type(value) is str and value in rule["values"] else (None, False)
        if kind == "artifact-sha256":
            for content in rule["values"]:
                expected = {"sha256": hashlib.sha256(content.encode()).hexdigest(), "hash_basis": "redacted", "nonpersonal": True}
                if value == content or value == expected:
                    return expected, True
        if kind == "hmac" and type(value) is str and len(value) <= 256:
            prefix = "hmac-sha256:" + self.version + ":"
            if re.fullmatch(re.escape(prefix) + r"[a-f0-9]{64}", value):
                return value, True
            if CREDENTIAL.search(value) or not EMAIL.fullmatch(value):
                return None, False
            message = "\0".join(("signalscout-p0", self.version, profile, path, value)).encode()
            return prefix + hmac.new(self.key, message, hashlib.sha256).hexdigest(), True
        return None, False

    def envelope(self, profile, body):
        unknown = '{"version":0,"capture":"off","detail":"unknown"}'
        try:
            if len(body.encode()) > 8192:
                return unknown
            env = json.loads(body, object_pairs_hook=unique)
            if type(env) is not dict or type(env.get("version")) is not int or env.get("version") != 1 or len(env.get("inputs", [])) > 32:
                return unknown
            inputs, seen = [], set()
            for item in env.get("inputs", []):
                name = item["name"]
                if name in seen:
                    return unknown
                seen.add(name)
                clean, ok = self.value(profile, "inputs." + name, item.get("inline"))
                if ok and item.get("capture") == "allowed":
                    inputs.append({"name": name, "capture": "allowed", "inline": clean})
            output = {"capture": "off"}
            clean, ok = self.value(profile, "output", env.get("output", {}).get("inline"))
            if ok and env.get("output", {}).get("capture") == "allowed":
                output = {"capture": "allowed", "inline": clean}
            return json.dumps({"version": 1, "inputs": inputs, "output": output, "omitted_inputs": len(env.get("inputs", [])) - len(inputs),
                               "detail": "redacted; identity and semantic metadata unknown"}, separators=(",", ":"))
        except Exception:
            return unknown

    def attributes(self, profile, values, prefix=""):
        result = {}
        for key, value in values.items():
            if key == "signalscout.record.json":
                result[key] = self.envelope(profile, value) if type(value) is str else self.envelope(profile, "")
            else:
                clean, ok = self.value(profile, prefix + "attributes." + key, value)
                if ok:
                    result[key] = clean
        return result

    def queued(self, profile, item):
        result = {k: item[k] for k in ("trace", "span", "parent", "start", "end", "error")}
        name, ok = self.value(profile, "otlp.name", item["name"])
        result["name"] = name if ok else "redacted"
        result["attrs"] = self.attributes(profile, item["attrs"])
        result["events"] = []
        for name, stamp, fields in item["events"]:
            clean, ok = self.value(profile, "otlp.name", name)
            result["events"].append((clean if ok else "redacted", stamp, self.attributes(profile, fields)))
        return result
