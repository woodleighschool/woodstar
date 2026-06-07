# Rewrite Inventory - Phase 0

Phase 0 purpose: inventory the current Woodstar topology before rewrite work starts.
This document is descriptive. No source rewrite, package move, schema change, generated-code refresh, staging of `GOAL.md`, or compatibility shim has started.

## 1. Resource Map Table

| Resource / surface | Model files | Store / service files | API handlers | DB tables | sqlc / query files | Frontend routes / components |
|---|---|---|---|---|---|---|
| Auth, account, setup, sessions | `internal/auth/account.go`, `internal/auth/api_key.go`, `internal/auth/oidc.go`, `internal/directory/user.go` | `internal/auth/service.go`, `internal/directory/user_service.go`, `internal/directory/user_store.go` | `internal/api/handlers/auth.go`, `account.go`, `sso.go`, `users.go` | `users`, `sessions` | `internal/database/queries/users.sql` | `web/src/routes/login.tsx`, `setup.tsx`, `_authenticated/account.tsx`, `_authenticated/directory.users*`, `web/src/pages/login.tsx`, `setup.tsx`, `account.tsx`, `users.tsx`, `users/edit.tsx` |
| Directory groups | `internal/directory/group.go`, `source.go`, `provider_snapshot.go` | `internal/directory/group_store.go`, `provider_store.go`, `entra/service.go` | `internal/api/handlers/groups.go` | `directory_groups`, `directory_group_memberships` | `internal/database/queries/directory.sql` | `web/src/routes/_authenticated/directory.groups*`, `web/src/pages/groups/list.tsx`, `web/src/hooks/use-groups.ts` |
| Agent secrets | `internal/agentauth/model.go` | `internal/agentauth/store.go`, `header.go` | `internal/api/handlers/agent_secrets.go` | `agent_secrets` | `internal/database/queries/agent_secrets.sql` | `web/src/hooks/use-agent-secrets.ts`, `web/src/components/enrollments/secrets-dialog.tsx`, enrollments pages |
| Hosts and host detail | `internal/hosts/model.go`, `detail.go`, `enricher.go`, `user_affinity.go`, `targeting.go` | `internal/hosts/store.go`, `user_affinity.go`; enriched by Munki/Santa host-state services at wiring time | `internal/api/handlers/hosts.go`, `hosts_osquery_checks.go`, `hosts_osquery_reports.go`, `hosts_santa.go`, `hosts_munki.go` | `hosts`, `host_users`, `host_batteries`, `host_certificates`, `host_user_affinity_mappings`, `host_user_links`, plus joins to label/software/check/report/Santa/Munki tables | `internal/database/queries/hosts.sql`, `host_user_affinity.sql`, `host_user_links.sql`, plus `checks.sql`, `reports.sql`, `software.sql`, `santa.sql`, `munki.sql` for detail tabs | `web/src/routes/_authenticated/hosts*`, `web/src/pages/hosts/*`, `web/src/hooks/use-hosts.ts`, `web/src/components/hosts/*` |
| Labels and label membership | `internal/labels/model.go` | `internal/labels/store.go`; osquery inventory label evaluator in `internal/osquery/ingest/labels.go` | `internal/api/handlers/labels.go` | `labels`, `label_membership` | `internal/database/queries/labels.sql`, plus `hosts.sql` for selected-label host resolution | `web/src/routes/_authenticated/labels*`, `web/src/pages/labels/*`, `web/src/hooks/use-labels.ts`, `web/src/components/labels/*` |
| Observed software inventory | `internal/software/model.go` | `internal/software/store.go`, `projection.go`, `titles.go`, `host.go`; observed host filtering in `internal/hosts/store.go` | `internal/api/handlers/software.go`; host software list/filtering in `hosts.go`; Santa references through `software.go` | `software_titles`, `software`, `host_software`, `host_software_installed_paths` | `internal/database/queries/software.sql`, `santa_references.sql` | `web/src/routes/_authenticated/software*`, `web/src/pages/software/*`, `web/src/hooks/use-software.ts`, host filters in `web/src/hooks/use-hosts.ts` and `web/src/pages/hosts/list.tsx`, `web/src/components/software/software-icon.tsx` |
| osquery reports | `internal/osquery/reports/model.go` | `internal/osquery/reports/store.go`, `targets.go`, `results.go` | `internal/api/handlers/osquery_reports.go`, `hosts_osquery_reports.go` | `reports`, `report_targets`, `report_results` | `internal/database/queries/reports.sql` | `web/src/routes/_authenticated/osquery.reports*`, `web/src/pages/osquery/reports/*`, `web/src/hooks/use-reports.ts`, `web/src/components/reports/report-result-card.tsx` |
| osquery checks | `internal/osquery/checks/model.go` | `internal/osquery/checks/store.go`, `targets.go` | `internal/api/handlers/osquery_checks.go`, `hosts_osquery_checks.go` | `checks`, `check_targets`, `check_membership` | `internal/database/queries/checks.sql` | `web/src/routes/_authenticated/osquery.checks*`, `web/src/pages/osquery/checks/*`, `web/src/hooks/use-checks.ts`, `web/src/components/osquery/checks/*` |
| osquery live queries | `internal/osquery/livequery/livequery.go` | `internal/osquery/livequery/livequery.go`; host target resolution in `internal/hosts/targeting.go` | `internal/api/handlers/osquery_livequery.go` | In-memory live-query manager only; resolves against `hosts` and `label_membership` | `internal/database/queries/hosts.sql` named queries `ListSelectedHostIDs`, `ListSelectedLabels`, `ListAllHostIDs`, `ListHostIDsByAnyLabel`, `ListHostIDsByBuiltinAndRegularLabels`, `ListOnlineSelectedHostIDs`, `CountSelectedHostStatus` | `web/src/components/osquery/live-runner.tsx`, `web/src/hooks/use-live-queries.ts`, live routes under osquery reports/checks |
| Santa configurations | `internal/santa/configurations/model.go` | `internal/santa/configurations/store.go` | `internal/api/handlers/santa_configurations.go` | `santa_configurations`, `santa_configuration_targets` | `internal/database/queries/santa.sql` named queries `CreateSantaConfiguration`, `UpdateSantaConfiguration`, `ResolveSantaConfigurationForHost`, `InsertSantaConfigurationTargets`, `ListSantaConfigurationTargets` | `web/src/routes/_authenticated/santa.configurations*`, `web/src/pages/santa/configurations/*`, `web/src/hooks/use-santa.ts`, `web/src/components/targeting/label-scope-editor.tsx` |
| Santa rules and rule target catalog | `internal/santa/rules/model.go`, `cel.go` | `internal/santa/rules/store.go` | `internal/api/handlers/santa_rules.go`, `hosts_santa.go` | `santa_rules`, `santa_rule_includes`, `santa_rule_exclude_labels`, `santa_sync_targets`, `santa_sync_pending_rules`; target catalog reads Santa/software reference tables | `internal/database/queries/santa.sql`, `santa_references.sql`, `santa_syncstate.sql` | `web/src/routes/_authenticated/santa.rules*`, `web/src/pages/santa/rules/*`, `web/src/components/santa/rules/rule-form-fields.tsx`, `web/src/components/targeting/label-target-rows-table.tsx`, `target-labels-cell.tsx` |
| Santa events, file access, and references | `internal/santa/events/model.go`, `internal/santa/references/model.go`, `internal/santa/syncstate/model.go`, `internal/santa/model.go` | `internal/santa/events/store.go`, `cleanup.go`; `internal/santa/references/store.go`; `internal/santa/syncstate/*`; `internal/santa/service.go`, `hoststate.go`, `store.go` | `internal/api/handlers/santa_events.go`, `software.go`, `hosts_santa.go`; agent protocol in `internal/santa/protocol/santa.go` | `santa_hosts`, `santa_sync_state`, `santa_sync_targets`, `santa_execution_events`, `santa_file_access_events`, `santa_executables`, `santa_signing_chains`, `santa_executable_signing_chains`, `santa_certificates`, `santa_signing_chain_entries`, `santa_bundles`, `santa_bundle_executables` | `internal/database/queries/santa.sql`, `santa_events.sql`, `santa_references.sql`, `santa_syncstate.sql` | `web/src/routes/_authenticated/santa.events*`, `web/src/pages/santa/events/*`, `web/src/pages/santa/file-access-events/*`, `web/src/lib/santa-events.ts` |
| Munki software titles and desired state | `internal/munki/softwaretitles/model.go` | `internal/munki/softwaretitles/store.go`; assignment rows delegated to `internal/munki/assignments` | `internal/api/handlers/munki_software_titles.go` via `internal/api/handlers/munki.go` | `munki_software_titles`, `munki_assignments`, `munki_assignment_exclude_labels`, related `munki_packages`, `munki_artifacts` | `internal/database/queries/munki.sql` named queries from `CreateMunkiSoftwareTitle` through `ListEffectiveMunkiPackagesForHost` | `web/src/routes/_authenticated/munki.software-titles*`, `web/src/pages/munki/software-title/*`, `web/src/hooks/munki/software-titles.ts`, `web/src/lib/munki-software-title*`, `web/src/lib/munki-assignment-form.ts` |
| Munki assignments | `internal/munki/assignments/model.go` | `internal/munki/assignments/store.go`, `resolver.go` | No standalone route. Exposed as nested `includes` and `exclude_label_ids` in `internal/api/handlers/munki_software_titles.go` | `munki_assignments`, `munki_assignment_exclude_labels` | `internal/database/queries/munki.sql` named queries `CreateMunkiAssignment`, `DeleteMunkiAssignmentsBySoftware*`, `ListMunkiAssignmentExcludeLabels`, `ListEffectiveMunkiPackagesForHost` | `web/src/components/munki/software-title/assignment-form.tsx`, `web/src/lib/munki-assignment-form.ts`, software-title detail/edit pages |
| Munki packages | `internal/munki/packages/model.go` | `internal/munki/packages/store.go`, `pkginfo.go` | `internal/api/handlers/munki_packages.go` | `munki_packages`, `munki_package_relations`, joins to `munki_artifacts`, `munki_software_titles` | `internal/database/queries/munki.sql` named queries `CreateMunkiPackage`, `UpdateMunkiPackage`, `UpsertMunkiPackage`, `CreateMunkiPackageRelation`, `GetMunkiPackageByID` | `web/src/routes/_authenticated/munki.software-titles.$softwareId_.packages*`, `web/src/pages/munki/software-title/package-edit.tsx`, `web/src/hooks/munki/packages.ts`, `web/src/components/munki/software-title/package-editor-fields.tsx` |
| Munki artifacts, storage, host state, repo protocol | `internal/munki/artifacts/model.go`, `internal/munki/hoststate/model.go` | `internal/munki/artifacts/store.go`, `internal/munki/storage/*`, `internal/munki/hoststate/store.go`, `internal/munki/service.go`; protocol in `internal/munki/protocol/munki.go` | `internal/api/handlers/munki_artifacts.go`, `hosts_munki.go` | `munki_artifacts`, `munki_host_status`, `munki_host_items` | `internal/database/queries/munki.sql` | Artifact upload hooks/components in Munki package/software-title pages; host Munki tab in `web/src/components/hosts/host-munki-tab.tsx` |
| Agent-facing protocols | `internal/orbit/protocol.go`, `internal/osquery/protocol.go`, `internal/santa/protocol.go`, Munki protocol service models | `internal/orbit/service.go`, `internal/osquery/service.go`, `internal/munki/service.go`, `internal/santa/service.go` | Mounted by `internal/api/protocols.go`; concrete protocol handlers live under each capability's `protocol/` package | Protocol-specific writes into host, inventory, label, Santa, Munki state tables | Capability query files listed above | Not OpenAPI admin UI; frontend only links enrollment instructions |

## 2. Terminology / Call-Site Classification

File-level audit command:

```bash
rg -n -c "scope|Scope|target|Target|targeting|Targeting|assignment|Assignment|assignments|Assignments|desired state|desired_state|DesiredState|software title|softwaretitles|SoftwareTitle" internal web/src cmd docs --glob '!**/routeTree.gen.ts' --glob '!**/api-client/**' --glob '!**/*lock*'
```

Generated/noise mirrors are also classified: `web/openapi.yaml` has 87 matching lines, `web/src/lib/api-client/types.gen.ts` has 107, `web/src/routeTree.gen.ts` has 84, and lockfiles contain dependency-name or compiler-target matches only.

| Class | Meaning today | Files / call sites |
|---|---|---|
| Shared label target value type | `include` / `exclude` label rows stored in multiple resource-target tables. This is the current `internal/scope` package and is the main thing that should not remain as `scope`. | `internal/scope/model.go`, `scope_test.go`; imported by `internal/osquery/checks/*`, `internal/osquery/reports/*`, `internal/santa/configurations/*`, frontend `TargetLabel` mirrors |
| Saved osquery query-definition target shape | `internal/osquery/queries/query_definition.go` imports `internal/scope` and has `Targets []scope.TargetLabel`. Grep found no production caller of `QueryDefinition`, so classify it as a real shared label-target call site that is currently dead or unreferenced topology code. | `internal/osquery/queries/query_definition.go` |
| osquery report/check targets | Saved report/check resources use `targets []scope.TargetLabel`, persisted to `report_targets` / `check_targets`, and resolved through `label_membership` when scheduled/check state is evaluated. | `internal/osquery/reports/model.go`, `targets.go`, `store.go`; `internal/osquery/checks/model.go`, `targets.go`, `store.go`; handlers `osquery_reports.go`, `osquery_checks.go`; frontend report/check forms and `LabelScopeEditor` |
| Santa configuration label targets | Admin configurations use `targets []scope.TargetLabel`; the chosen configuration for a host is resolved from include/exclude label matches and position order. | `internal/santa/configurations/model.go`, `store.go`; handler `santa_configurations.go`; queries in `internal/database/queries/santa.sql`; frontend `santa/configurations/*`, `TargetLabelsCell`, `LabelScopeEditor` |
| Santa rule label assignments | Santa rules do not use `scope.TargetLabel`. They use ordered `includes` rows plus `exclude_label_ids`; includes carry policy/CEL per label. | `internal/santa/rules/model.go`, `store.go`; tables `santa_rule_includes`, `santa_rule_exclude_labels`; handler `santa_rules.go`; frontend `santa/rules/*`, `label-target-rows-table.tsx`, `label-target-rows.ts` |
| Santa rule target catalog | "Target" means a binary/certificate/teamid/signingid/cdhash/bundle candidate derived from observed software/security facts, not host targeting. | `internal/santa/rules/model.go` `RuleTarget`, `RuleTargetListParams`; `internal/santa/rules/store.go` `ListRuleTargets`; `/api/santa/rule-targets`; `web/src/hooks/use-santa.ts`, `web/src/components/santa/rules/rule-form-fields.tsx` |
| Santa sync targets | "Target" means desired/applied Santa rule payloads for a host sync. This is protocol state, not label scope. | `santa_sync_targets`, `santa_sync_pending_rules`; `internal/santa/syncstate/*`; `internal/santa/service.go`; queries `santa_syncstate.sql` and `santa.sql` summary fields |
| Santa event target / scope words | `target` is a file-access target path. `allow_scope` / `block_scope` are Santa execution-decision enum values from the client/protocol. | `internal/santa/events/model.go`, `store.go`; `santa_file_access_events.target`; `web/src/pages/santa/events/list.tsx`, `file-access-events/detail.tsx`; `web/src/lib/santa-events.ts`, `santa-cel.ts` |
| Host live-query targeting | `TargetSelection` is a runtime union of selected host IDs and label IDs for live queries. It is not persisted as resource scope. | `internal/hosts/targeting.go`; `internal/api/handlers/osquery_livequery.go`; `internal/database/queries/hosts.sql`; `web/src/components/osquery/live-runner.tsx`; `web/src/hooks/use-live-queries.ts` |
| Munki assignments / desired state | Assignments are desired-state rows owned by Munki software titles. Include labels decide which assignment wins for a host; exclude labels suppress selected hosts. | `cmd/woodstar/main.go` wiring; `internal/munki/assignments/*`; `internal/munki/softwaretitles/*`; `internal/munki/service.go`; `internal/munki/protocol/*_test.go`; frontend Munki software-title routes, forms, hooks |
| Munki package relation target | `target_package_id` is a package-to-package relation target for requires/update_for, not host targeting. | `internal/munki/packages/store.go`, `model.go`; `munki_package_relations`; `internal/database/queries/munki.sql` |
| Software title, observed inventory | `internal/software` software titles are observed inventory aggregates from osquery/software projection. Host filtering by `software_title_id` / `software_id` is observed software inventory filtering, not Munki desired-state software. | `internal/software/model.go`, `titles.go`, `projection.go`, `host.go`; `internal/api/handlers/software.go`; host filter call sites in `internal/api/handlers/hosts.go`, `internal/hosts/model.go`, `internal/hosts/store.go`, `web/src/hooks/use-hosts.ts`, `web/src/pages/hosts/list.tsx`; `web/src/pages/software/*`, `web/src/hooks/use-software.ts` |
| Munki software title, desired state | `internal/munki/softwaretitles` software titles are Woodstar-authored Munki desired-state roots with packages and assignment rows. | `internal/munki/softwaretitles/*`; `internal/api/handlers/munki_software_titles.go`; `web/src/pages/munki/software-title/*`, `web/src/hooks/munki/software-titles.ts` |
| API/generated mirrors | OpenAPI and generated client files mirror Huma/backend shapes and should not drive source ownership decisions. | `web/openapi.yaml`, `web/src/lib/api-client/*`, `web/src/routeTree.gen.ts` |
| Non-topology false positives | Compiler/build target, DOM `event.target`, config OIDC scopes, schema docs paths, markdown/editor path variables, lockfile dependency names. These should not be rewritten as targeting concepts. | `web/vite.config.ts`, `web/tsconfig*.json`, `web/src/**/*.tsx` form handlers, `internal/auth/oidc.go`, `internal/config/config.go`, lockfiles, generic path helpers |

## 3. Current Admin API Route Map And Future Disposition

All paths below are current OpenAPI admin paths from `web/openapi.yaml`. Agent-facing Orbit/osquery/Munki/Santa protocol routes are mounted separately by `internal/api/protocols.go` and are not OpenAPI admin routes.

| Current route group | Paths | Current handler home | Future disposition |
|---|---|---|---|
| Account/auth/setup | `GET,PUT /api/account`; `POST,DELETE /api/account/api-key`; `POST /api/auth/login`; `POST /api/auth/logout`; `GET /api/auth/session`; `POST /api/setup`; SSO routes live under `/api/auth/sso/*` but are chi routes, not OpenAPI | `internal/api/handlers/account.go`, `auth.go`, `sso.go` | Admin API composition moves to `internal/adminapi`; resource-owned auth/account registration should replace central handler bucket |
| Users/groups | `GET,POST /api/users`; `GET /api/users/departments`; `GET,PUT,DELETE /api/users/{id}`; `GET /api/groups`; `GET /api/groups/{id}` | `internal/api/handlers/users.go`, `groups.go` | Directory/user resource-owned API registrations |
| Hosts | `GET /api/hosts`; `POST /api/hosts/bulk-delete`; `GET,DELETE /api/hosts/{id}`; `GET,PUT,DELETE /api/hosts/{id}/user-affinity`; `GET /api/hosts/{id}/software`; `GET /api/hosts/{id}/osquery/checks`; `GET /api/hosts/{id}/osquery/reports`; `GET /api/hosts/{id}/osquery/reports/{report_id}`; `GET /api/hosts/{id}/santa/rules` | `internal/api/handlers/hosts*.go` | `inventory`/host resource API with capability enrichers preserved at wiring time |
| Labels | `GET,POST /api/labels`; `GET,PUT,DELETE /api/labels/{id}` | `internal/api/handlers/labels.go` | Label admin API near the future targeting/label owner; label membership remains concrete |
| Observed software | `GET /api/software`; `GET /api/software/{id}`; `GET /api/software/{id}/santa` | `internal/api/handlers/software.go` | Move observed software into `internal/inventory`; do not confuse with Munki desired-state software titles |
| osquery reports/checks/live queries | `GET,POST /api/osquery/reports`; `POST /api/osquery/reports/bulk-delete`; `GET,PUT,DELETE /api/osquery/reports/{id}`; `GET /api/osquery/reports/{id}/results`; `GET,POST /api/osquery/checks`; `POST /api/osquery/checks/bulk-delete`; `GET,PUT,DELETE /api/osquery/checks/{id}`; `GET /api/osquery/checks/{id}/hosts`; `POST /api/live-queries`; `POST /api/live-queries/targets/count`; `POST /api/live-queries/{id}/stop`; `GET /api/live-queries/{id}/stream` | `internal/api/handlers/osquery_*.go` | osquery-owned API registration; shared label-target DTO should move out of `internal/scope` |
| Santa admin | `GET,POST /api/santa/configurations`; `POST /api/santa/configurations/bulk-delete`; `PUT /api/santa/configurations/order`; `GET,PATCH,DELETE /api/santa/configurations/{id}`; `GET /api/santa/rule-targets`; `GET,POST /api/santa/rules`; `POST /api/santa/rules/bulk-delete`; `GET,PATCH,DELETE /api/santa/rules/{id}`; `GET /api/santa/events`; `GET /api/santa/events/{id}`; `GET /api/santa/file-access-events`; `GET /api/santa/file-access-events/{id}` | `internal/api/handlers/santa_*.go` | Santa resource-owned API registrations; keep rule-target catalog separate from label targeting |
| Munki admin | `POST /api/munki/artifact-uploads`; `POST /api/munki/artifacts`; `GET /api/munki/artifacts/{id}/content`; `GET,POST /api/munki/packages`; `POST /api/munki/packages/import`; `GET,PATCH /api/munki/packages/{id}`; `GET,POST /api/munki/software-titles`; `POST /api/munki/software-titles/bulk-delete`; `GET,PATCH,DELETE /api/munki/software-titles/{id}` | `internal/api/handlers/munki*.go` | Munki resource-owned API registrations; assignments remain nested desired-state behavior, not a standalone public surface |
| Agent secrets | `GET,POST /api/agent-secrets`; `PATCH,DELETE /api/agent-secrets/{id}` | `internal/api/handlers/agent_secrets.go` | `agentauth` resource-owned API registration |

### Route-by-route target / scope / assignment disposition

| Current route | Methods | Current target/scope/assignment behavior | Future disposition |
|---|---|---|---|
| `/api/hosts` | `GET` | Host list filters include `label_id`, `software_title_id`, `software_id`, and check filters. `software_title_id` is observed software inventory filtering through `software_titles`, not Munki desired state. | Retain as inventory host filtering; do not fold observed software title filters into Munki targeting. |
| `/api/hosts/{id}` | `GET` | `GET /api/hosts/{id}` returns host detail with labels plus Munki/Santa/osquery enrichment. | Retain as host detail projection, with label/Munki/Santa/osquery enrichers at wiring/resource boundaries; not a target mutation route. |
| `/api/hosts/{id}/software` | `GET` | Observed host software inventory for one host. | Retain as inventory projection. |
| `/api/hosts/{id}/osquery/checks` | `GET` | Host-facing status for checks after target resolution. | Retain as resolved status projection, not target configuration. |
| `/api/hosts/{id}/osquery/reports` | `GET` | Host-facing scheduled report membership/results after target resolution. | Retain as resolved report projection. |
| `/api/hosts/{id}/osquery/reports/{report_id}` | `GET` | Single host/report result projection. | Retain as projection. |
| `/api/hosts/{id}/santa/rules` | `GET` | Host-facing Santa rule sync/status projection. | Retain as effective-state projection, not rule-target editing. |
| `/api/labels` | `GET,POST` | Label list/create for the concrete host targeting primitive. | Retain labels as concrete targeting data; do not replace with a generic expression engine. |
| `/api/labels/{id}` | `GET,PUT,DELETE` | Label read/update/delete, including membership settings. | Retain as label resource API near the future targeting owner. |
| `/api/software/{id}/santa` | `GET` | Santa references derived from observed software/security facts. | Retain as observed software/Santa catalog projection, not host targeting. |
| `/api/live-queries` | `POST` | Runtime `selected.hosts` / `selected.labels` selection for one live query. | Retain outside persisted resource targeting; consider naming it runtime selection rather than `targets`. |
| `/api/live-queries/targets/count` | `POST` | Runtime host/label selection count. | Retain for now as a runtime selection-count command route because it is not persisted resource targeting. Exclude it from persisted `targets.include` / `targets.exclude` convergence; revisit naming only if Phase 6 changes live-query command routes. |
| `/api/live-queries/{id}/stop` | `POST` | Command on a running query; no label target model. | Retain as command-shaped runtime path. |
| `/api/live-queries/{id}/stream` | `GET` | Stream for a running query; no label target model. | Retain as runtime stream. |
| `/api/osquery/reports` | `GET,POST` | List/create target-bearing reports with `targets: TargetLabel[]`. | Future resource body should use `targets.include` / `targets.exclude` while preserving report ownership. |
| `/api/osquery/reports/{id}` | `GET,PUT,DELETE` | Get/update/delete one target-bearing report. | `GET`/update should expose full-resource targets include/exclude; delete remains resource deletion. |
| `/api/osquery/reports/bulk-delete` | `POST` | Delete command; target word not involved. | Retain only if bulk command remains useful, otherwise delete with the handler migration. |
| `/api/osquery/reports/{id}/results` | `GET` | Report result snapshots after scheduling/target resolution. | Retain as results projection. |
| `/api/osquery/checks` | `GET,POST` | List/create target-bearing checks with `targets: TargetLabel[]`. | Future resource body should use `targets.include` / `targets.exclude` while preserving check ownership. |
| `/api/osquery/checks/{id}` | `GET,PUT,DELETE` | Get/update/delete one target-bearing check. | `GET`/update should expose full-resource targets include/exclude; delete remains resource deletion. |
| `/api/osquery/checks/bulk-delete` | `POST` | Delete command; target word not involved. | Retain only if bulk command remains useful, otherwise delete with the handler migration. |
| `/api/osquery/checks/{id}/hosts` | `GET` | Per-host check status after target resolution. | Retain as status projection. |
| `/api/santa/configurations` | `GET,POST` | List/create ordered Santa configurations with `targets: TargetLabel[]`. | Future full-resource body should use `targets.include` / `targets.exclude`; ordered matching remains Santa-specific. |
| `/api/santa/configurations/{id}` | `GET,PATCH,DELETE` | Get/patch/delete one target-bearing Santa configuration. | `GET`/patch should expose full-resource targets include/exclude; delete remains resource deletion. |
| `/api/santa/configurations/order` | `PUT` | Reorders configuration priority; no target model. | Retain as Santa-specific ordering operation if ordered configurations survive. |
| `/api/santa/configurations/bulk-delete` | `POST` | Delete command; target word not involved. | Retain only if bulk command remains useful. |
| `/api/santa/rule-targets` | `GET` | Santa rule target catalog for binary/certificate/teamid/signingid/cdhash/bundle candidates. | Retain as catalog, not host targeting. Rename only if needed to make that distinction clearer. |
| `/api/santa/rules` | `GET,POST` | List/create rules with label includes and exclude labels; includes also carry policy/CEL. | Future full-resource body should use `targets.include` / `targets.exclude` for label selection while preserving Santa rule identity and include policy/CEL semantics. |
| `/api/santa/rules/{id}` | `GET,PATCH,DELETE` | Get/patch/delete one rule with label includes/excludes. | `GET`/patch should expose full-resource targets include/exclude; delete remains resource deletion. |
| `/api/santa/rules/bulk-delete` | `POST` | Delete command; target word not involved. | Retain only if bulk command remains useful. |
| `/api/santa/events` | `GET` | Execution-event rows include Santa client `allow_scope` / `block_scope` decision vocabulary. | Retain as Santa event vocabulary, not `internal/scope`. |
| `/api/santa/events/{id}` | `GET` | Single execution event with Santa decision scope fields. | Retain as event detail. |
| `/api/santa/file-access-events` | `GET` | File-access event rows include a file `target` path. | Retain as event vocabulary, not host/resource targeting. |
| `/api/santa/file-access-events/{id}` | `GET` | Single file-access event with file `target` path. | Retain as event detail. |
| `/api/munki/software-titles` | `GET,POST` | Current desired-state root exposes nested assignment includes and exclude labels. | Future path should be `/api/munki/software` with `targets.include` / `targets.exclude`; no separate assignments endpoint. |
| `/api/munki/software-titles/{id}` | `GET,PATCH,DELETE` | Current detail/update/delete for nested assignment desired state. | Future `/api/munki/software/{id}` should expose packages plus targets include/exclude; no standalone assignments endpoint. |
| `/api/munki/software-titles/bulk-delete` | `POST` | Delete command for desired-state software-title roots. | Rename/delete with the future `/api/munki/software` resource. |
| `/api/munki/packages` | `GET,POST` | Package list/create; list may be scoped by `software_id`, not host targeting. | Retain as Munki package resource or nest deliberately under desired-state software; do not classify as target behavior. |
| `/api/munki/packages/{id}` | `GET,PATCH` | Package read/update. Relations use `target_package_id` internally for requires/update_for. | Retain as package resource; package relation targets stay package-to-package references. |
| `/api/munki/packages/import` | `POST` | Package import command; no host targeting. | Retain as command if importer remains. |

## 4. OpenAPI / Schema References Involving Scope, Target, Assignment

OpenAPI source inspected: `web/openapi.yaml`.

| Schema / path | Current behavior | Notes for convergence |
|---|---|---|
| `TargetLabel` | `{ label_id, effect }` where effect is `include` or `exclude` | Generated from `internal/scope.TargetLabel`; primary candidate for future `internal/targeting` value type |
| `Label`, `LabelMutation`, `PageLabel` | Label entity, manual/dynamic/derived membership mutation, and label list page. | Labels remain the concrete targeting primitive; membership mutation belongs with labels, not generic targeting. |
| `HostDetail`, `PageHost` | Host detail embeds `labels`; host list supports filters including `label_id`, `software_title_id`, and `software_id`. | Host label and software-title filtering is inventory filtering/projection, not desired-state assignment. |
| `OsqueryReport`, `OsqueryReportMutation` | `targets: TargetLabel[]` | Persists to `report_targets` |
| `PageReport` | Report list page whose items are `OsqueryReport` with target rows. | Must reflect future `targets.include` / `targets.exclude` if the report body changes. |
| `HostReport`, `HostReportResultsBody`, `ReportResult` | Host report membership/results from `/api/hosts/{id}/osquery/reports`, `/api/hosts/{id}/osquery/reports/{report_id}`, and report result rows from `/api/osquery/reports/{id}/results`. | Target-resolution projections only; keep distinct from persisted report target mutation schemas. |
| `OsqueryCheck`, `OsqueryCheckMutation` | `targets: TargetLabel[]` | Persists to `check_targets` |
| `PageCheck` | Check list page whose items are `OsqueryCheck` with target rows. | Must reflect future `targets.include` / `targets.exclude` if the check body changes. |
| `CheckHostStatus` | Host/check status rows returned by `/api/hosts/{id}/osquery/checks` and `/api/osquery/checks/{id}/hosts`. | Target-resolution projection only; not a target mutation surface. |
| `SantaConfiguration`, `ConfigurationMutation`, `ConfigurationMatch` | `targets: TargetLabel[]`; host match reports `matched_via_label` | Persists to `santa_configuration_targets`; ordered configuration matching is Santa-specific |
| `PageConfiguration` | Configuration list page whose items are `SantaConfiguration` with target rows. | Must reflect future full-resource targets include/exclude. |
| `SantaHostState`, `RuleSyncSummary`, `PageRuleStatus`, `RuleStatus` | Host Santa state and rule status projections after configuration/rule resolution and sync state. | Projection/read-model schemas; keep distinct from target editing schemas. |
| `LiveQueryCreateBody`, `LiveQuerySelectedBody`, `LiveQueryTargetCountBody`, `LiveQueryTargetCountOutputBody` | `selected.hosts`, `selected.labels` runtime target selection; output counts target totals/online/offline | No persisted target resource; resolves through `internal/hosts/targeting.go` |
| `AssignmentIncludeMutation` | Munki desired-state include row: label, action, package selection, pinned package, optional/featured flags | Nested under `MunkiSoftwareTitleMutation.includes` |
| `MunkiAssignment` | Stored Munki assignment row with priority and resolved pinned-package display fields | Returned inside `MunkiSoftwareTitleDetail.includes` |
| `MunkiSoftwareTitle`, `MunkiSoftwareTitleDetail`, `MunkiSoftwareTitleMutation`, `PageMunkiSoftwareTitle` | Munki desired-state software title root; detail includes packages, `includes`, and `exclude_label_ids` | Separate from observed `SoftwareTitle` |
| `PackageReference`, `MunkiPackage`, `MunkiPackageMutation`, `PageMunkiPackage` | Munki package schemas expose package relation references in `requires` and `update_for`; DB stores these via `munki_package_relations.target_package_id`. | `target_package_id` is package-to-package relation vocabulary, not host/label targeting. |
| `RuleInclude`, `RuleIncludeWrite` | Santa rule include rows with `label_id`, policy, position, and optional CEL expression. | These are label-target-ish but not mechanical `TargetLabel` rows because include policy/CEL is Santa rule behavior. |
| `RuleMutation`, `SantaRule` | Santa rule includes/exclude-label shape; no `TargetLabel` field | Includes carry policy/CEL and label ID; excludes are IDs only |
| `PageRule` | Rule list page whose items are `SantaRule` with includes/exclude-labels. | Must reflect future full-resource targets include/exclude if rule schema changes. |
| `RuleTarget` and `/api/santa/rule-targets` | Santa target catalog for binaries/certs/signing IDs/bundles | Not host targeting; `target_type` is Santa rule identity type |
| `ExecutionEvent`, `PageExecutionEvent` | Execution event item/page schemas; items contain enum values `allow_scope` and `block_scope`. | External Santa decision vocabulary, not `internal/scope`. |
| `FileAccessEvent`, `PageFileAccessEvent` | File-access event item/page schemas; items contain `target` path/string for file access. | Not host/resource targeting. |
| `SigningIdentityReference` | `target_type` for Santa signing identity rule targets | Santa evidence/catalog vocabulary |
| `SoftwareReference`, `BundleReference`, `CertificateReference`, `ExecutableReference` | `/api/software/{id}/santa` response and nested Santa reference catalog facts for observed software. | Target-adjacent Santa evidence/catalog schemas; not host targeting. |
| `SoftwareTitle`, `PageSoftwareTitle` | Observed inventory aggregate under `/api/software` | Future `internal/inventory`, not Munki desired state |
| Generated client mirrors | `web/src/lib/api-client/types.gen.ts`, `client.gen.ts`, `routeTree.gen.ts` | Regenerate after contract changes; do not edit by hand |

## 5. DB Label / Resource Relationship Tables And Query Files

| Table | Migration | Relationship stored | Query files / key names | Current owners |
|---|---|---|---|---|
| `labels` | `001_initial_schema.sql` | Label entity. `label_type` = builtin/regular; `label_membership_type` = dynamic/manual/derived. | `labels.sql` CRUD/list/builtin lookup | `internal/labels` |
| `label_membership` | `001_initial_schema.sql` | Label to host membership. This is the concrete targeting primitive every label-target resource resolves through. | `labels.sql` membership writes; `hosts.sql` selected-label host resolution; read by `reports.sql`, `checks.sql`, `santa.sql`, `munki.sql` | `internal/labels`, `internal/hosts`, osquery ingest |
| `report_targets` | `001_initial_schema.sql` | osquery report to label with include/exclude effect. | `reports.sql`: `ListReportTargets`, `DeleteReportTargets`, `InsertReportTargets`, `ListScheduledReportsForHost` | `internal/osquery/reports` |
| `check_targets` | `001_initial_schema.sql` | osquery check to label with include/exclude effect. | `checks.sql`: `ListCheckTargets`, `DeleteCheckTargets`, `InsertCheckTargets`, applicable/host status queries | `internal/osquery/checks` |
| `santa_configuration_targets` | `002_santa.sql` | Santa configuration to label with include/exclude effect. | `santa.sql`: `InsertSantaConfigurationTargets`, `ListSantaConfigurationTargets`, `ResolveSantaConfigurationForHost` | `internal/santa/configurations` |
| `santa_rule_includes` | `002_santa.sql` | Santa rule include labels with position, policy, and optional CEL expression. | `santa.sql`: `InsertSantaRuleIncludes`, `ListSantaRuleIncludes`, `ListSantaRulesForHost*`, `CountSantaRulesForHost` | `internal/santa/rules` |
| `santa_rule_exclude_labels` | `002_santa.sql` | Santa rule exclude labels. | `santa.sql`: `InsertSantaRuleExcludeLabels`, `ListSantaRuleExcludeLabels`, `ListSantaRulesForHost*` | `internal/santa/rules` |
| `munki_assignments` | `006_munki_desired_state.sql` | Munki software title include label to action/package-selection row. | `munki.sql`: `CreateMunkiAssignment`, `DeleteMunkiAssignmentsBySoftware*`, `ListEffectiveMunkiPackagesForHost` | `internal/munki/softwaretitles`, `internal/munki/assignments` |
| `munki_assignment_exclude_labels` | `006_munki_desired_state.sql` | Munki software title exclude labels. | `munki.sql`: `InsertMunkiAssignmentExcludeLabels`, `ListMunkiAssignmentExcludeLabels`, `ListEffectiveMunkiPackagesForHost` | `internal/munki/softwaretitles`, `internal/munki/assignments` |

Related target-named tables/columns that are not label-resource relationships:

- `santa_sync_targets`: host sync desired/applied payload rows, owned by Santa sync state.
- `munki_package_relations.target_package_id`: package-to-package relation target.
- `santa_file_access_events.target`: observed file-access target path.
- Santa rule target catalog/reference tables: `santa_executables`, `santa_signing_chains`, `santa_executable_signing_chains`, `santa_certificates`, `santa_signing_chain_entries`, `santa_bundles`, `santa_bundle_executables`.
- `santa_references.sql` / `santa.sql` CTEs named `targets`: Santa rule target catalog facts built from the exact reference tables above.

### Duplicated target resolution inventory

| Query / file | Current resolution behavior | Convergence note |
|---|---|---|
| `reports.sql` `ListScheduledReportsForHost` | Resolves report include/exclude label matches against `label_membership` for one host. | Converge on `internal/targeting` label semantics, while preserving SQL as the scheduler/report persistence read path. |
| `checks.sql` `ListApplicableChecksForHost` | Resolves check include/exclude label matches against `label_membership` for one host. | Converge on the same include/exclude semantics as reports. |
| `santa.sql` `ResolveSantaConfigurationForHost` | Selects the first ordered Santa configuration whose include/exclude labels match one host. | Share label-target semantics, but preserve Santa configuration ordering. |
| `santa.sql` `ListSantaRulesForHost` and `ListSantaRulesForHostPage` | Resolves Santa rule include labels, exclude labels, policy/CEL payload, and applied sync state for one host. | Use shared label-match semantics where applicable; do not flatten Santa policy/CEL or sync-state payload into generic targeting. |
| `munki.sql` `ListEffectiveMunkiPackagesForHost` | Resolves Munki assignment include/exclude labels, pinned package choice, optional/featured flags, and action per host. | Share label-match semantics where applicable, but preserve Munki desired-state assignment precedence and package selection in SQL. |
| `internal/munki/assignments/resolver.go` `ResolveEffectivePackages` | Applies assignment precedence to candidate Munki packages after rows are resolved. | This is downstream Munki precedence logic, not a generic `internal/targeting` replacement. It should survive only if Munki still needs this policy after convergence. |

## 6. Validation / Business-Logic Placement Notes

### Validation currently in model files

- `internal/scope/model.go`: `TargetLabelEffect` enum, Huma schema, pgx scan/value, and `ValidTargetLabelEffect`.
- `internal/osquery/checks/model.go`: `CheckMutation.Validate` only checks required name/query.
- `internal/osquery/reports/model.go`: `ReportMutation.Validate` checks required name/query and non-negative schedule interval.
- `internal/munki/assignments/model.go`: assignment action/package-selection enums, Huma schemas, `AssignmentMutation.Validate`, package-payload semantics, and `AssignmentIncludeMutation.Mutation`.
- `internal/munki/artifacts/model.go`: artifact kind/location/SHA-256/storage-key validation and public `ValidArtifactKind` / `ValidArtifactLocation`.
- `internal/munki/packages/model.go`: package mutation validation for installer/uninstaller semantics, package references, installs/receipts/items-to-copy, import JSON validity, enum validators, and `EffectiveIconArtifactID`.
- `internal/labels/model.go`: enum Huma schemas and `Criteria.json`; mutation validation is in `store.go`.
- `internal/santa/configurations/model.go`: enum schemas and `RemovableMediaPolicy.IsZero`; mutation validation is in `store.go`.
- `internal/santa/rules/model.go`: enum schemas and DTOs only; rule mutation validation is in `store.go`.
- `internal/santa/events/model.go`: Santa decision/filter/signing enums and Huma schemas, including `allow_scope` / `block_scope` decision values.

### Validation currently in stores or store-adjacent files

- `internal/labels/store.go`: `LabelMutation.Validate`, membership-type pairing, derived criteria validation, manual/derived membership replacement, and derived refresh behavior.
- `internal/osquery/checks/targets.go` and `internal/osquery/reports/targets.go`: duplicate target-row validation for effect validity and duplicate `(label_id,effect)` rows; store create/update replaces child target rows transactionally.
- `internal/santa/configurations/store.go`: `ConfigurationMutation.Validate`, target validation, removable-media policy validation, exact-set reorder validation, and host configuration resolution.
- `internal/santa/rules/store.go`: `RuleMutation.Validate`, CEL syntax validation, identifier format validation by rule type, duplicate include/exclude label checks, built-in exclude-label rejection, and bundle-target collected/completeness validation.
- `internal/munki/softwaretitles/store.go`: trims title fields, normalizes icons from artifacts, builds assignment mutations from ordered includes, validates duplicate/conflicting include/exclude labels, rejects built-in exclude labels, validates pinned package ownership, and replaces all desired-state child rows transactionally.
- `internal/munki/packages/store.go`: trims and normalizes package mutation fields, validates artifact IDs/kinds, marshals package JSON fields, and writes package relations.
- `internal/munki/artifacts/store.go`: trims and defaults artifact display fields before model validation.
- `internal/hosts/store.go`: validates host list filters (`status`, `check_id` / `check_response` pairing) while building dynamic list SQL.
- `internal/hosts/targeting.go`: resolves live-query host/label target selections, special-cases the built-in "All Hosts" label, and combines built-in plus regular labels.
- `internal/software/projection.go` and `titles.go`: observed software title grouping/upsert policy and browser extension display logic.

### Business logic currently in `model.go`

The behavior-bearing model files are `internal/scope/model.go`, `internal/osquery/checks/model.go`, `internal/osquery/reports/model.go`, `internal/munki/assignments/model.go`, `internal/munki/artifacts/model.go`, `internal/munki/packages/model.go`, and `internal/santa/configurations/model.go` (`IsZero`). These should be reviewed deliberately during package moves so validators do not get stranded in DTO-only model files by accident.

## 7. Compatibility Shims / Dead Code That Must Not Survive

- `internal/scope` is a shared label-target value package, not a real bounded-context owner. It should not survive under that name.
- `internal/api/handlers` is the current central admin handler bucket. The target shape calls for `internal/adminapi` composition plus resource-owned `api.go` files, so this package should disappear rather than be kept as a forwarding shim.
- `internal/web` currently serves the embedded SPA (`internal/web/handler.go`) and is imported by `cmd/woodstar/main.go`, `internal/api/dependencies.go`, `internal/api/router.go`, and `internal/api/server_test.go`. GOAL maps this to `internal/webui`; `internal/web` should not survive as a compatibility package.
- `internal/software` owns observed inventory, while `internal/munki/softwaretitles` owns Munki desired state. The name collision is real and should be resolved by the future `internal/inventory` move.
- `internal/munki/assignments` is not a standalone admin resource today. It is nested desired-state behavior for Munki software titles and should not survive as a generic assignment package.
- Generated OpenAPI/client files and route tree files are mirrors. They must be regenerated after API changes, not used as source ownership.
- Duplicate `validateTargets` functions in reports/checks/configurations are current evidence of a shared target-row concept. Do not replace them with temporary adapters just to keep intermediate commits green.
- Duplicate target resolution exists in report/check/Santa/Munki SQL and Munki assignment precedence code. It must converge on shared `internal/targeting` label semantics only where that does not erase resource-specific persistence, ordering, policy, sync, or package-selection behavior.
- The API path `/api/live-queries/targets/count` currently uses "targets" for runtime host selection; it is contract surface, not proof that every "target" belongs to a generic target package.
- Santa `RuleTarget`, `target_type`, file-access `target`, and execution decisions `allow_scope` / `block_scope` are Santa vocabulary. They should not be folded blindly into host/label targeting.
- False-positive `target`/`scope` matches from DOM events, TypeScript/compiler targets, Vite proxy target, OIDC scopes, markdown/editor paths, and lockfile package names are not topology concepts.

## 8. Phase 0 Coverage Checklist

- [x] Built resource map across model/store/service/API/DB/sqlc/frontend surfaces.
- [x] Classified every matching `scope`, `target`, `targeting`, `assignment`, `desired state`, and `software title` file into topology, Santa/Munki domain vocabulary, generated mirror, or false-positive categories.
- [x] Listed every current OpenAPI admin route group and route-by-route target/scope/assignment future disposition.
- [x] Listed exact OpenAPI schemas and paths involving target/scope/assignment behavior.
- [x] Identified DB tables storing label/resource relationships.
- [x] Identified duplicated target resolution queries and Munki assignment precedence code.
- [x] Identified validation currently located in models, stores, and store-adjacent files.
- [x] Identified business logic currently located in `model.go`.
- [x] Documented shims/dead code that must not survive the topology convergence.
- [x] No code rewrite has started.
