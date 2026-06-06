#!/usr/local/autopkg/python

import hashlib
import http.client
import json
import mimetypes
import os
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlsplit
from urllib.request import Request, urlopen

from autopkglib import ProcessorError


class WoodstarClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key

    def get(self, path, query=None):
        if query:
            path = f"{path}?{urlencode(query)}"
        return self.request("GET", path)

    def post(self, path, body=None):
        return self.request("POST", path, body)

    def put(self, path, body=None):
        return self.request("PUT", path, body)

    def patch(self, path, body=None):
        return self.request("PATCH", path, body)

    def request(self, method, path, body=None):
        data = None
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Accept": "application/json",
        }
        if body is not None:
            data = json.dumps(json_safe(body), separators=(",", ":")).encode("utf-8")
            headers["Content-Type"] = "application/json"
        request = Request(
            f"{self.base_url}{path}",
            data=data,
            headers=headers,
            method=method,
        )
        try:
            with urlopen(request) as response:
                payload = response.read()
        except HTTPError as err:
            raise ProcessorError(http_error_message(method, path, err)) from err
        except URLError as err:
            raise ProcessorError(f"{method} {path} failed: {err.reason}") from err
        if not payload:
            return None
        return json.loads(payload.decode("utf-8"))

    def upload_artifact(self, kind, file_path, display_name=None):
        file_path = os.path.abspath(file_path)
        if not os.path.isfile(file_path):
            raise ProcessorError(f"Artifact file does not exist: {file_path}")
        filename = os.path.basename(file_path)
        size_bytes = os.path.getsize(file_path)
        sha256 = sha256sum(file_path)
        content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
        upload = self.post(
            "/api/munki/artifact-uploads",
            {
                "kind": kind,
                "filename": filename,
                "display_name": display_name or filename,
                "content_type": content_type,
                "size_bytes": size_bytes,
                "sha256": sha256,
            },
        )
        put_file(upload["upload_url"], file_path, upload.get("headers") or {})
        return self.post("/api/munki/artifacts", upload["artifact"])


def client_from_env(env):
    base_url = env.get("WOODSTAR_URL")
    api_key = env.get("WOODSTAR_API_KEY")
    if not base_url:
        raise ProcessorError("WOODSTAR_URL is required")
    if not api_key:
        raise ProcessorError("WOODSTAR_API_KEY is required")
    return WoodstarClient(str(base_url), str(api_key))


def find_exact(client, path, field, value, extra_query=None):
    query = {"q": value, "page_size": 1000}
    if extra_query:
        query.update(extra_query)
    page = client.get(path, query)
    for item in page.get("items", []):
        if item.get(field) == value:
            return item
    return None


def sha256sum(file_path):
    digest = hashlib.sha256()
    with open(file_path, "rb", buffering=0) as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def put_file(url, file_path, headers):
    parts = urlsplit(url)
    path = parts.path or "/"
    if parts.query:
        path = f"{path}?{parts.query}"
    conn_cls = http.client.HTTPSConnection if parts.scheme == "https" else http.client.HTTPConnection
    headers = dict(headers)
    headers.setdefault("Content-Length", str(os.path.getsize(file_path)))
    with open(file_path, "rb") as handle:
        conn = conn_cls(parts.netloc)
        try:
            conn.request("PUT", path, body=handle, headers=headers)
            response = conn.getresponse()
            body = response.read().decode("utf-8", "replace")
            if response.status < 200 or response.status >= 300:
                raise ProcessorError(f"PUT artifact failed: HTTP {response.status} {body}")
        finally:
            conn.close()


def http_error_message(method, path, err):
    body = err.read().decode("utf-8", "replace")
    try:
        parsed = json.loads(body)
    except ValueError:
        parsed = {}
    message = parsed.get("message") or parsed.get("detail") or body or err.reason
    return f"{method} {path} failed: HTTP {err.code}: {message}"


def json_safe(value):
    if isinstance(value, dict):
        return {str(key): json_safe(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [json_safe(item) for item in value]
    if hasattr(value, "isoformat"):
        return value.isoformat()
    return value


def truthy(value):
    if isinstance(value, bool):
        return value
    return str(value).strip().lower() not in {"", "0", "false", "no", "none"}
