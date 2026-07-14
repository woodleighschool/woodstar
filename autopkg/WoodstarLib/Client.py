#!/usr/local/autopkg/python

import hashlib
import json
import mimetypes
import os
from urllib.parse import urlsplit, urlunsplit

import requests
from autopkglib import ProcessorError

API_TIMEOUT = (10, 60)
UPLOAD_TIMEOUT = (10, 3600)


class WoodstarClient:
    def __init__(self, base_url, api_key, ca_file=None):
        self.base_url = validate_base_url(base_url)
        verify = validate_ca_file(ca_file)
        self.session = requests.Session()
        self.session.verify = verify
        self.session.headers.update(
            {
                "Authorization": f"Bearer {api_key}",
                "Accept": "application/json",
            }
        )
        self.upload_session = requests.Session()
        self.upload_session.verify = verify

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

    def request(self, method, path, body=None, query=None, timeout=API_TIMEOUT):
        request_kwargs = {}
        if query:
            request_kwargs["params"] = query
        if body is not None:
            request_kwargs["json"] = body
        try:
            response = self.session.request(
                method,
                f"{self.base_url}{path}",
                timeout=timeout,
                **request_kwargs,
            )
            response.raise_for_status()
        except requests.HTTPError as err:
            raise ProcessorError(http_error_message(method, path, err.response)) from err
        except requests.RequestException as err:
            raise ProcessorError(f"{method} {path} failed: {err}") from err
        if not response.content:
            return None
        return response.json()

    def attach_object(self, attach_path, file_path, display_name=None):
        """Upload a file to a resource via the create-first storage lifecycle:
        create a pending object on the resource, PUT the bytes, then confirm.
        attach_path is the resource upload endpoint, for example
        /api/munki/packages/12/installer. Returns the confirmed object."""
        file_path = os.path.abspath(file_path)
        if not os.path.isfile(file_path):
            raise ProcessorError(f"upload file does not exist: {file_path}")
        filename = display_name or os.path.basename(file_path)
        content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
        target = self.post(attach_path, {"filename": filename, "content_type": content_type})
        self.upload_bytes(target, file_path)
        return self.request(
            "POST",
            f"{attach_path}/{target['object_id']}/confirm",
            timeout=UPLOAD_TIMEOUT,
        )

    def upload_bytes(self, target, file_path):
        url = target["upload_url"]
        method = target["method"].upper()
        transport = target["upload_transport"]
        if transport not in ("woodstar", "s3"):
            raise ProcessorError(f"unsupported upload transport: {transport}")
        parsed_url = urlsplit(url)
        if parsed_url.scheme != "https" or not parsed_url.netloc:
            raise ProcessorError(f"upload URL must use HTTPS: {safe_url(url)}")
        headers = dict(target.get("headers") or {})
        headers.setdefault("Content-Length", str(os.path.getsize(file_path)))
        with open(file_path, "rb") as handle:
            try:
                response = self.upload_session.request(
                    method,
                    url,
                    data=handle,
                    headers=headers,
                    timeout=UPLOAD_TIMEOUT,
                )
            except requests.RequestException as err:
                raise ProcessorError(
                    f"upload to {safe_url(url)} failed: {type(err).__name__}"
                ) from err
        if response.status_code < 200 or response.status_code >= 300:
            raise ProcessorError(
                f"upload to {safe_url(url)} failed: HTTP {response.status_code}"
            )


def safe_url(value):
    """Return a URL safe for errors by dropping user information and query credentials."""
    try:
        parsed = urlsplit(value)
        hostname = parsed.hostname or ""
        if ":" in hostname and not hostname.startswith("["):
            hostname = f"[{hostname}]"
        if parsed.port is not None:
            hostname = f"{hostname}:{parsed.port}"
        return urlunsplit((parsed.scheme, hostname, parsed.path, "", ""))
    except ValueError:
        return "<invalid URL>"


def needs_object(resource, kind, file_path, force=False):
    """Return whether a local artifact differs from the attached object."""
    if force or not resource.get(f"{kind}_object_id"):
        return True
    attached = resource.get(f"{kind}_file")
    if not attached:
        raise ProcessorError(f"Woodstar response is missing {kind}_file metadata")
    local = local_file_metadata(file_path)
    return any(attached.get(key) != value for key, value in local.items())


def local_file_metadata(file_path):
    path = os.path.abspath(file_path)
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return {
        "filename": os.path.basename(path),
        "size_bytes": os.path.getsize(path),
        "sha256": digest.hexdigest(),
    }


def client_from_env(env):
    base_url = env.get("WOODSTAR_URL")
    api_key = env.get("WOODSTAR_API_KEY")
    if not base_url:
        raise ProcessorError("WOODSTAR_URL is required")
    if not api_key:
        raise ProcessorError("WOODSTAR_API_KEY is required")
    return WoodstarClient(str(base_url), str(api_key), env.get("WOODSTAR_CA_FILE"))


def validate_base_url(base_url):
    value = str(base_url).strip().rstrip("/")
    parsed = urlsplit(value)
    if parsed.scheme != "https" or not parsed.netloc:
        raise ProcessorError("WOODSTAR_URL must be an HTTPS origin")
    if parsed.username or parsed.password or parsed.path or parsed.query or parsed.fragment:
        raise ProcessorError("WOODSTAR_URL must be an HTTPS origin")
    return value


def validate_ca_file(ca_file):
    if not ca_file:
        return True
    path = os.path.abspath(str(ca_file).strip())
    if not os.path.isfile(path):
        raise ProcessorError(f"WOODSTAR_CA_FILE does not exist: {path}")
    return path


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


def truthy(value):
    if isinstance(value, bool):
        return value
    text = str(value).strip().lower()
    if text in {"true", "1", "yes"}:
        return True
    if text in {"false", "0", "no", ""}:
        return False
    raise ProcessorError(f"expected a boolean value, got {value!r}")
