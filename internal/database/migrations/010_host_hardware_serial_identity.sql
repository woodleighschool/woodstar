-- +goose Up

CREATE UNIQUE INDEX hosts_hardware_serial_idx
    ON hosts (hardware_serial)
    WHERE hardware_serial <> '';

-- +goose Down

DROP INDEX IF EXISTS hosts_hardware_serial_idx;
