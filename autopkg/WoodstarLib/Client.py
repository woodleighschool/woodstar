#!/usr/local/autopkg/python

import json
import mimetypes
import os

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
            request_kwargs["json"] = body
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
        return self.post(f"{attach_path}/{target['object_id']}/confirm")

    def upload_bytes(self, target, file_path):
        url = target["upload_url"]
        method = target["method"].upper()
        transport = target["upload_transport"]
        if transport not in ("woodstar", "s3"):
            raise ProcessorError(f"unsupported upload transport: {transport}")
        if not (url.startswith("http://") or url.startswith("https://")):
            raise ProcessorError(f"upload URL must be absolute: {url}")
        headers = dict(target.get("headers") or {})
        headers.setdefault("Content-Length", str(os.path.getsize(file_path)))
        with open(file_path, "rb") as handle:
            try:
                response = requests.request(method, url, data=handle, headers=headers)
            except requests.RequestException as err:
                raise ProcessorError(f"upload to {url} failed: {err}") from err
        if response.status_code < 200 or response.status_code >= 300:
            raise ProcessorError(f"upload to {url} failed: HTTP {response.status_code}: {response.text}")


def needs_object(resource, kind, force=False):
    """True when a resource still needs its installer/icon uploaded
    (or force re-uploads it). resource is a package or software response."""
    return bool(force) or not resource.get(f"{kind}_object_id")


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
