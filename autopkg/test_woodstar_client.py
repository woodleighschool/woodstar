import os
import sys
import tempfile
import unittest
from unittest.mock import Mock

sys.path.insert(0, os.path.dirname(__file__))
sys.path.insert(0, "/Library/AutoPkg")

from autopkglib import ProcessorError  # noqa: E402
from WoodstarLib.Client import (  # noqa: E402
    API_TIMEOUT,
    UPLOAD_TIMEOUT,
    WoodstarClient,
    local_file_metadata,
    needs_object,
    safe_url,
)


class WoodstarClientTests(unittest.TestCase):
    def test_api_session_uses_ca_and_finite_timeout(self):
        with tempfile.NamedTemporaryFile() as ca:
            client = WoodstarClient("https://woodstar.example/", "secret", ca.name)
            response = Mock(content=b'{"ok":true}')
            response.json.return_value = {"ok": True}
            client.session.request = Mock(return_value=response)

            self.assertEqual(client.get("/api/test"), {"ok": True})
            self.assertEqual(client.session.verify, os.path.abspath(ca.name))
            self.assertEqual(client.upload_session.verify, os.path.abspath(ca.name))
            self.assertEqual(client.session.request.call_args.kwargs["timeout"], API_TIMEOUT)

    def test_direct_upload_uses_separate_session_ca_and_timeout(self):
        with tempfile.NamedTemporaryFile() as ca, tempfile.NamedTemporaryFile() as artifact:
            artifact.write(b"package")
            artifact.flush()
            client = WoodstarClient("https://woodstar.example", "secret", ca.name)
            response = Mock(status_code=200, text="")
            client.upload_session.request = Mock(return_value=response)

            client.upload_bytes(
                {
                    "upload_url": "https://uploads.example/package",
                    "method": "PUT",
                    "upload_transport": "s3",
                    "headers": {},
                },
                artifact.name,
            )

            call = client.upload_session.request.call_args
            self.assertEqual(call.kwargs["timeout"], UPLOAD_TIMEOUT)
            self.assertNotIn("Authorization", client.upload_session.headers)
            self.assertEqual(call.kwargs["headers"]["Content-Length"], str(len(b"package")))

    def test_direct_upload_failure_omits_signed_url_and_response_body(self):
        with tempfile.NamedTemporaryFile() as artifact:
            client = WoodstarClient("https://woodstar.example", "secret")
            client.upload_session.request = Mock(
                return_value=Mock(
                    status_code=403,
                    text="denied https://uploads.example/package?signature=secret",
                )
            )

            with self.assertRaises(ProcessorError) as caught:
                client.upload_bytes(
                    {
                        "upload_url": "https://uploads.example/package?signature=secret",
                        "method": "PUT",
                        "upload_transport": "s3",
                        "headers": {},
                    },
                    artifact.name,
                )

            message = str(caught.exception)
            self.assertEqual(message, "upload to https://uploads.example/package failed: HTTP 403")
            self.assertNotIn("signature", message)

    def test_resource_attachment_uses_transfer_timeout(self):
        with tempfile.NamedTemporaryFile() as artifact:
            artifact.write(b"package")
            artifact.flush()
            client = WoodstarClient("https://woodstar.example", "secret")
            create = Mock(
                content=(
                    b'{"object_id":42,"upload_url":"https://uploads.example/object",'
                    b'"method":"PUT","upload_transport":"s3"}'
                )
            )
            create.json.return_value = {
                "object_id": 42,
                "upload_url": "https://uploads.example/object",
                "method": "PUT",
                "upload_transport": "s3",
            }
            adopted = Mock(content=b'{"id":42}')
            adopted.json.return_value = {"id": 42}
            client.session.request = Mock(side_effect=[create, adopted])
            client.upload_bytes = Mock()

            self.assertEqual(
                client.attach_object("/api/munki/software/7/icon", artifact.name),
                {"id": 42},
            )

            adoption_call = client.session.request.call_args_list[1]
            self.assertEqual(
                adoption_call.args[:2],
                ("PUT", "https://woodstar.example/api/munki/software/7/icon"),
            )
            self.assertEqual(adoption_call.kwargs["json"], {"object_id": 42})
            self.assertEqual(adoption_call.kwargs["timeout"], UPLOAD_TIMEOUT)

    def test_package_installer_upload_reserves_then_finalizes(self):
        with tempfile.NamedTemporaryFile() as artifact:
            artifact.write(b"package")
            artifact.flush()
            client = WoodstarClient("https://woodstar.example", "secret")
            reserved = Mock(
                content=(
                    b'{"object_id":42,"upload_url":"https://uploads.example/object",'
                    b'"method":"PUT","upload_transport":"s3"}'
                )
            )
            reserved.json.return_value = {
                "object_id": 42,
                "upload_url": "https://uploads.example/object",
                "method": "PUT",
                "upload_transport": "s3",
            }
            finalized = Mock(content=b'{"id":42}')
            finalized.json.return_value = {"id": 42}
            client.session.request = Mock(side_effect=[reserved, finalized])
            client.upload_bytes = Mock()

            self.assertEqual(client.upload_package_installer(artifact.name), {"id": 42})

            reserve_call, finalize_call = client.session.request.call_args_list
            self.assertEqual(
                reserve_call.args[:2],
                ("POST", "https://woodstar.example/api/munki/package-installers"),
            )
            self.assertEqual(reserve_call.kwargs["json"], {"filename": os.path.basename(artifact.name)})
            self.assertEqual(
                finalize_call.args[:2],
                ("PUT", "https://woodstar.example/api/munki/package-installers/42"),
            )
            self.assertNotIn("json", finalize_call.kwargs)
            self.assertEqual(finalize_call.kwargs["timeout"], UPLOAD_TIMEOUT)

    def test_safe_url_removes_user_information_and_query(self):
        self.assertEqual(
            safe_url(
                "https://access:secret@storage.example:8443/"
                "object.pkg?signature=secret#fragment"
            ),
            "https://storage.example:8443/object.pkg",
        )

    def test_rejects_non_https_origins_and_uploads(self):
        with self.assertRaises(ProcessorError):
            WoodstarClient("http://woodstar.example", "secret")

        client = WoodstarClient("https://woodstar.example", "secret")
        with tempfile.NamedTemporaryFile() as artifact:
            with self.assertRaises(ProcessorError):
                client.upload_bytes(
                    {
                        "upload_url": "http://uploads.example/package",
                        "method": "PUT",
                        "upload_transport": "woodstar",
                    },
                    artifact.name,
                )

    def test_rejects_missing_ca_file(self):
        with self.assertRaises(ProcessorError):
            WoodstarClient("https://woodstar.example", "secret", "/missing/woodstar-ca.pem")

    def test_installer_reuse_compares_name_size_and_sha256(self):
        with tempfile.NamedTemporaryFile() as artifact:
            artifact.write(b"package-v1")
            artifact.flush()
            metadata = local_file_metadata(artifact.name)
            package = {
                "installer_object_id": 42,
                "installer_file": {
                    **metadata,
                    "installer_item_location": "packages/42/installer/package.pkg",
                },
            }

            self.assertFalse(needs_object(package, "installer", artifact.name))
            self.assertTrue(needs_object(package, "installer", artifact.name, force=True))

            artifact.seek(0)
            artifact.truncate()
            artifact.write(b"package-v2")
            artifact.flush()
            self.assertTrue(needs_object(package, "installer", artifact.name))

    def test_icon_reuse_compares_name_size_sha256_and_force(self):
        with tempfile.NamedTemporaryFile(suffix=".png") as icon:
            icon.write(b"icon-v1")
            icon.flush()
            metadata = local_file_metadata(icon.name)
            software = {"icon_object_id": 42, "icon_file": metadata}

            self.assertFalse(needs_object(software, "icon", icon.name))
            self.assertTrue(needs_object(software, "icon", icon.name, force=True))

            software["icon_file"] = {**metadata, "filename": "different.png"}
            self.assertTrue(needs_object(software, "icon", icon.name))

            software["icon_file"] = metadata
            icon.seek(0)
            icon.truncate()
            icon.write(b"icon-v2")
            icon.flush()
            self.assertTrue(needs_object(software, "icon", icon.name))

    def test_attached_object_requires_metadata(self):
        with tempfile.NamedTemporaryFile() as artifact:
            with self.assertRaises(ProcessorError):
                needs_object({"icon_object_id": 42}, "icon", artifact.name)


if __name__ == "__main__":
    unittest.main()
