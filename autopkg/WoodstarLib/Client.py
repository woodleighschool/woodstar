#!/usr/local/autopkg/python

import hashlib
import json
import mimetypes
import os
import posixpath
from urllib.parse import urlsplit

import requests
from autopkglib import ProcessorError


class WoodstarClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        self.session.headers.update(
            {
                "Authorization": f"Bearer {api_key}",
                "Accept": "application/json",
            }
        )

    def get(self, path, query=None):
        return self.request("GET", path, query=query)

    def post(self, path, body=None):
        return self.request("POST", path, body)

    def put(self, path, body=None):
        return self.request("PUT", path, body)

    def patch(self, path, body=None):
        return self.request("PATCH", path, body)

    def delete(self, path):
        return self.request("DELETE", path)

    def request(self, method, path, body=None, query=None):
        request_kwargs = {}
        if query:
            request_kwargs["params"] = query
        if body is not None:
            request_kwargs["json"] = json_safe(body)
        try:
            response = self.session.request(method, f"{self.base_url}{path}", **request_kwargs)
            response.raise_for_status()
        except requests.HTTPError as err:
            raise ProcessorError(http_error_message(method, path, err.response)) from err
        except requests.RequestException as err:
            raise ProcessorError(f"{method} {path} failed: {err}") from err
        if not response.content:
            return None
        return response.json()

    def upload_artifact(self, kind, file_path, display_name=None):
        artifact, _uploaded = self.upload_artifact_status(kind, file_path, display_name, force=True)
        return artifact

    def upload_artifact_status(self, kind, file_path, display_name=None, force=False, artifact_index=None):
        target = artifact_target(kind, file_path, display_name)
        if not force:
            existing = find_artifact(self, target, artifact_index)
            if existing:
                return existing, False

        upload = self.post(
            "/api/munki/artifact-uploads",
            {
                "kind": target["kind"],
                "filename": target["filename"],
                "display_name": target["display_name"],
                "content_type": target["content_type"],
                "size_bytes": target["size_bytes"],
                "sha256": target["sha256"],
            },
        )
        put_file(upload["upload_url"], target["file_path"], upload.get("headers") or {})
        artifact = self.post("/api/munki/artifacts", upload["artifact"])
        if artifact_index is not None:
            artifact_index[artifact_index_key(artifact)] = artifact
        return artifact, True


def artifact_target(kind, file_path, display_name=None):
    file_path = os.path.abspath(file_path)
    if not os.path.isfile(file_path):
        raise ProcessorError(f"Artifact file does not exist: {file_path}")
    filename = clean_upload_filename(os.path.basename(file_path))
    size_bytes = os.path.getsize(file_path)
    sha256 = sha256sum(file_path)
    content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    location = artifact_location(sha256, filename)
    return {
        "kind": kind,
        "file_path": file_path,
        "filename": filename,
        "display_name": display_name or filename,
        "location": location,
        "content_type": content_type,
        "size_bytes": size_bytes,
        "sha256": sha256,
        "storage_key": artifact_storage_key(kind, location),
    }


def client_from_env(env):
    base_url = env.get("WOODSTAR_URL")
    api_key = env.get("WOODSTAR_API_KEY")
    if not base_url:
        raise ProcessorError("WOODSTAR_URL is required")
    if not api_key:
        raise ProcessorError("WOODSTAR_API_KEY is required")
    return WoodstarClient(str(base_url), str(api_key))


def find_exact(client, path, field, value, extra_query=None):
    query = {"q": value, "per_page": 1000}
    if extra_query:
        query.update(extra_query)
    for item in list_items(client, path, query):
        if item.get(field) == value:
            return item
    return None


def list_items(client, path, query=None, per_page=1000):
    query = dict(query or {})
    query.setdefault("per_page", per_page)
    page_number = int(query.get("page") or 1)
    items = []
    while True:
        query["page"] = page_number
        page = client.get(path, query)
        page_items = page.get("items") or []
        items.extend(page_items)
        count = page.get("count")
        if not page_items:
            break
        if count is not None and len(items) >= int(count):
            break
        if len(page_items) < int(query["per_page"]):
            break
        page_number += 1
    return items


def find_artifact(client, target, artifact_index=None):
    if artifact_index is not None:
        return artifact_index.get(artifact_index_key(target))
    for artifact in list_items(client, "/api/munki/artifacts"):
        if (
            artifact.get("kind") == target["kind"]
            and artifact.get("location") == target["location"]
            and artifact.get("sha256") == target["sha256"]
            and int(artifact.get("size_bytes") or 0) == int(target["size_bytes"])
        ):
            return artifact
    return None


def artifact_index(client):
    return {artifact_index_key(artifact): artifact for artifact in list_items(client, "/api/munki/artifacts")}


def artifact_index_key(artifact):
    return (
        artifact.get("kind"),
        artifact.get("location"),
        artifact.get("sha256"),
        int(artifact.get("size_bytes") or 0),
    )


def clean_upload_filename(filename):
    filename = posixpath.basename(str(filename).strip().replace("\\", "/"))
    if filename in {"", ".", "/"}:
        raise ProcessorError("artifact filename is required")
    return filename


def artifact_location(sha256, filename):
    if len(sha256) >= 12:
        return f"{sha256[:12]}/{filename}"
    return filename


def artifact_storage_key(kind, location):
    if kind == "package":
        return f"pkgs/{location}"
    if kind == "icon":
        return f"icons/{location}"
    return f"{kind}/{location}"


def sha256sum(file_path):
    digest = hashlib.sha256()
    with open(file_path, "rb", buffering=0) as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def put_file(url, file_path, headers):
    parts = urlsplit(url)
    if parts.scheme not in {"http", "https"}:
        raise ProcessorError(f"unsupported artifact upload URL scheme: {parts.scheme}")
    if not parts.hostname:
        raise ProcessorError("artifact upload URL is missing a host")

    headers = dict(headers)
    headers.setdefault("Content-Length", str(os.path.getsize(file_path)))
    with open(file_path, "rb") as handle:
        try:
            response = requests.put(url, data=handle, headers=headers)
        except requests.RequestException as err:
            raise ProcessorError(f"PUT artifact failed: {err}") from err
        if response.status_code < 200 or response.status_code >= 300:
            raise ProcessorError(f"PUT artifact failed: HTTP {response.status_code} {response.text}")


def http_error_message(method, path, response):
    body = response.text
    try:
        parsed = json.loads(body)
    except ValueError:
        parsed = {}
    message = parsed.get("message") or parsed.get("detail") or body or response.reason
    if parsed.get("errors"):
        message = f"{message}: {json.dumps(parsed['errors'], separators=(',', ':'))}"
    return f"{method} {path} failed: HTTP {response.status_code}: {message}"


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
