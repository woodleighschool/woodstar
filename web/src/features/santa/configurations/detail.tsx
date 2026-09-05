import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { EnumBadge } from "@components/enum-badge";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import type { SantaRemovableMediaPolicy } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatInterval } from "@lib/utils";

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
  const canEdit = useCan({ resource: "santa.configurations", access: "edit" });
  const id = parseRouteID(configurationID);
  const query = useSantaConfiguration(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return (
      <QueryGate
        title="Failed to Load Configuration"
        error={{ message: "Configuration route is invalid." }}
      />
    );
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Configuration"
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
        actions={
          canEdit ? (
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

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={configuration.name} />
        <KeyValueRow label="Description" value={configuration.description} />
        <KeyValueRow label="Client Mode" value={CLIENT_MODES[configuration.client_mode].name} />
        <KeyValueRow label="Order" value={configuration.position + 1} />
        <KeyValueRow
          label="Full Sync Interval"
          value={formatInterval(configuration.full_sync_interval_seconds)}
        />
        <KeyValueRow label="Batch Size" value={configuration.batch_size} />
        <KeyValueRow
          label="Bundles"
          value={<BooleanIndicator value={configuration.enable_bundles} />}
        />
        <KeyValueRow
          label="Transitive Rules"
          value={<BooleanIndicator value={configuration.enable_transitive_rules} />}
        />
        <KeyValueRow
          label="All Event Upload"
          value={<BooleanIndicator value={configuration.enable_all_event_upload} />}
        />
        <KeyValueRow
          label="Unknown Event Upload"
          value={<BooleanIndicator value={!configuration.disable_unknown_event_upload} />}
        />
        <KeyValueRow
          label="File Access"
          value={
            <EnumBadge
              value={configuration.override_file_access_action}
              metadata={FILE_ACCESS_ACTIONS}
            />
          }
        />
        <KeyValueRow label="Allowed Path Regex" value={configuration.allowed_path_regex} />
        <KeyValueRow label="Blocked Path Regex" value={configuration.blocked_path_regex} />
        <KeyValueRow label="Event Detail URL" value={configuration.event_detail_url} />
        <KeyValueRow label="Event Detail Text" value={configuration.event_detail_text} />
        <KeyValueRow
          label="Removable Media"
          value={<MediaPolicyValue policy={configuration.removable_media_policy} />}
        />
        <KeyValueRow
          label="Encrypted Removable Media"
          value={<MediaPolicyValue policy={configuration.encrypted_removable_media_policy} />}
        />
      </KeyValueSection>

      <LabelTargetDetails targets={configuration.targets} />

      <ConfigurationDeleteDialog
        configuration={configuration}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/santa/configurations" })}
      />
    </PageShell>
  );
}

function MediaPolicyValue({ policy }: { policy: SantaRemovableMediaPolicy | undefined }) {
  const action: SantaMediaAction = policy?.action ?? "none";
  return (
    <div className="flex flex-wrap items-center gap-2">
      <EnumBadge value={action} metadata={MEDIA_ACTIONS} />
      {policy?.remount_flags?.length ? <TokenList values={policy.remount_flags} /> : null}
    </div>
  );
}
