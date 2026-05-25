SELECT
  eh.path,
  eh.executable_path,
  eh.executable_sha256
FROM apps a
JOIN executable_hashes eh ON a.path = eh.path;
