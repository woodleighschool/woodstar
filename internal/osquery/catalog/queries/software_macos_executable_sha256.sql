SELECT eh.*
FROM apps a
JOIN executable_hashes eh ON a.path = eh.path;
