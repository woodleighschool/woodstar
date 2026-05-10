SELECT c.*
FROM apps a
JOIN codesign c ON a.path = c.path;
