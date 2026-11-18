-- +goose Up

CREATE OR REPLACE VIEW osquery_policy_assignments AS
SELECT target.policy_id, membership.host_id
FROM osquery_policy_targets target
JOIN label_membership membership ON membership.label_id = target.label_id
GROUP BY target.policy_id, membership.host_id
HAVING bool_and(target.direction = 'include');

CREATE OR REPLACE VIEW osquery_report_assignments AS
SELECT target.report_id, membership.host_id
FROM osquery_report_targets target
JOIN label_membership membership ON membership.label_id = target.label_id
GROUP BY target.report_id, membership.host_id
HAVING bool_and(target.direction = 'include');
