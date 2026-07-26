import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { EnumBadge } from "@components/enum-badge";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { useAuth } from "@features/auth/queries";
import { LabelRefList } from "@features/labels/components/label-ref-list";
import type { SantaRemovableMediaPolicy } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatInterval, formatRelative } from "@lib/utils";

import { ConfigurationDeleteDialog } from "./delete-dialog";
import {
  CLIENT_MODES,
  FILE_ACCESS_ACTIONS,
  MEDIA_ACTIONS,
  type SantaMediaAction,
} from "./metadata";
import { useSantaConfiguration } from "./queries";

export function ConfigurationDetailPage() {
  const { id: configurationID } = useParams({
    from: "/_authenticated/santa/configurations/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const id = parseRouteID(configurationID);
  const query = useSantaConfiguration(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return (
      <QueryGate
        title="Failed to load configuration"
        error={{ message: "Configuration route is invalid." }}
      />
    );
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load configuration"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const configuration = query.data;

  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Configuration Details"
        description={configuration.description || undefined}
        context={<EnumStatusIndicator value={configuration.client_mode} metadata={CLIENT_MODES} />}
        actions={
          isAdmin ? (
            <>
              <Button
                size="sm"
                render={
                  <Link
                    to="/santa/configurations/$id/edit"
                    params={{ id: String(configuration.id) }}
                  />
                }
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>Options</CardTitle>
        </CardHeader>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem label="Name" value={configuration.name} />
            <KeyValueItem label="Order" value={configuration.position + 1} />
            <KeyValueItem
              label="Full Sync Interval"
              value={formatInterval(configuration.full_sync_interval_seconds)}
            />
            <KeyValueItem label="Batch Size" value={configuration.batch_size} />
            <KeyValueItem label="Bundles" value={enabledLabel(configuration.enable_bundles)} />
            <KeyValueItem
              label="Transitive Rules"
              value={enabledLabel(configuration.enable_transitive_rules)}
            />
            <KeyValueItem
              label="All Event Upload"
              value={enabledLabel(configuration.enable_all_event_upload)}
            />
            <KeyValueItem
              label="Unknown Event Upload"
              value={enabledLabel(!configuration.disable_unknown_event_upload)}
            />
            <KeyValueItem
              label="File Access"
              value={
                <EnumBadge
                  value={configuration.override_file_access_action}
                  metadata={FILE_ACCESS_ACTIONS}
                />
              }
            />
            <KeyValueItem
              label="Allowed Path Regex"
              value={<CodeValue value={configuration.allowed_path_regex} />}
              className="sm:col-span-2"
            />
            <KeyValueItem
              label="Blocked Path Regex"
              value={<CodeValue value={configuration.blocked_path_regex} />}
              className="sm:col-span-2"
            />
            <KeyValueItem
              label="Event Detail URL"
              value={configuration.event_detail_url}
              className="sm:col-span-2"
            />
            <KeyValueItem
              label="Event Detail Text"
              value={configuration.event_detail_text}
              className="sm:col-span-2"
            />
            <KeyValueItem
              label="Removable Media"
              value={<MediaPolicyValue policy={configuration.removable_media_policy} />}
            />
            <KeyValueItem
              label="Encrypted Removable Media"
              value={<MediaPolicyValue policy={configuration.encrypted_removable_media_policy} />}
            />
            <KeyValueItem label="Created" value={formatRelative(configuration.created_at)} />
            <KeyValueItem label="Updated" value={formatRelative(configuration.updated_at)} />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Targets</CardTitle>
        </CardHeader>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem
              label="Include"
              value={
                <LabelRefList
                  labelIDs={configuration.targets.include.map((target) => target.label_id)}
                />
              }
            />
            <KeyValueItem
              label="Exclude"
              value={
                <LabelRefList
                  labelIDs={configuration.targets.exclude.map((target) => target.label_id)}
                />
              }
            />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <ConfigurationDeleteDialog
        configuration={configuration}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/santa/configurations" })}
      />
    </PageShell>
  );
}

function enabledLabel(value: boolean) {
  return value ? "Enabled" : "Disabled";
}

function CodeValue({ value }: { value: string | undefined }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return <code className="font-mono text-xs">{value}</code>;
}

function MediaPolicyValue({ policy }: { policy: SantaRemovableMediaPolicy | undefined }) {
  const action: SantaMediaAction = policy?.action ?? "none";
  return (
    <div className="flex flex-wrap items-center gap-2">
      <EnumBadge value={action} metadata={MEDIA_ACTIONS} />
      {policy?.remount_flags?.length ? (
        <code className="text-xs text-muted-foreground">{policy.remount_flags.join(", ")}</code>
      ) : null}
    </div>
  );
}
