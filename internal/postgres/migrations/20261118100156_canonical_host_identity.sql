-- +goose Up

-- osquery can supply an opaque host_identifier when system_info has no hardware UUID.
CREATE FUNCTION canonical_host_identity(identity text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT
RETURN CASE WHEN identity ~ '^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$'
    THEN lower(identity) ELSE identity END;

UPDATE hosts
SET hardware_uuid = canonical_host_identity(hardware_uuid)
WHERE hardware_uuid <> canonical_host_identity(hardware_uuid);
