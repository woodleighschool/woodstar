import { Link, useParams } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import type { ReactNode } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { QueryGate } from "@/components/query-gate";
import { SoftwareIcon, softwareIconProps } from "@/components/software/software-icon";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/hooks/use-auth";
import { useSoftwareSantaReference, useSoftwareTitle } from "@/hooks/use-software";
import type { SantaRule, SantaSoftwareReference, SoftwareTitle, SoftwareVersion } from "@/lib/api";
import { ruleTypeLabel } from "@/lib/santa-rules";
import { softwareSourceLabel } from "@/pages/software/software-source-labels";
type BundleReference = NonNullable<SantaSoftwareReference["bundles"]>[number];
type CertificateReference = NonNullable<SantaSoftwareReference["certificates"]>[number];
type ExecutableReference = NonNullable<SantaSoftwareReference["executables"]>[number];
type SigningIdentityReference = NonNullable<SantaSoftwareReference["signing_identities"]>[number];
type SantaRuleType = SantaRule["rule_type"];
export function SoftwareDetailPage() {
  const { id: softwareId } = useParams({
    from: "/_authenticated/software/titles/$id",
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const query = useSoftwareTitle(Number(softwareId), {
    refetchInterval: 30000,
  });
  const title = query.data;
  if (query.error || !title) {
    return (
      <QueryGate
        title="Failed to load software title"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  return (
    <PageShell className="gap-6">
      <SoftwareHeader title={title} />
      <SoftwareInfoCard title={title} />
      <SoftwareSantaCard titleID={title.id} isAdmin={isAdmin} />
      <SoftwareVersionsCard title={title} />
    </PageShell>
  );
}
function SoftwareHeader({ title }: { title: SoftwareTitle }) {
  const typeLabel = softwareSourceLabel(title.source, title.extension_for);
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex min-w-0 items-center gap-4">
        <SoftwareIcon {...softwareIconProps(title.source)} size="lg" />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="truncate text-xl font-semibold text-foreground" title={title.name}>
              {title.name}
            </h1>
            <Badge variant="secondary" className="font-normal">
              {typeLabel}
            </Badge>
          </div>
        </div>
      </div>
      <Button
        variant="outline"
        size="sm"
        render={<Link to="/hosts" search={{ software_title_id: title.id }} />}
        nativeButton={false}
      >
        View hosts
      </Button>
    </div>
  );
}
function SoftwareInfoCard({ title }: { title: SoftwareTitle }) {
  return (
    <Card>
      <CardContent>
        <KeyValueGrid>
          {title.browser ? <KeyValueItem label="Browser" value={title.browser} /> : null}
          {title.bundle_identifier ? (
            <KeyValueItem label="Bundle Identifier" value={title.bundle_identifier} />
          ) : null}
          {title.extension_for ? (
            <KeyValueItem label="Extension for" value={title.extension_for} />
          ) : null}
          <KeyValueItem
            label="Hosts"
            value={<span className="tabular-nums">{title.hosts_count}</span>}
          />
          <KeyValueItem
            label="Versions"
            value={<span className="tabular-nums">{title.versions_count}</span>}
          />
        </KeyValueGrid>
      </CardContent>
    </Card>
  );
}
function SoftwareSantaCard({ titleID, isAdmin }: { titleID: number; isAdmin: boolean }) {
  const query = useSoftwareSantaReference(titleID);
  if (query.error) {
    return (
      <QueryError
        title="Failed to load Santa data"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (!query.data) {
    return null;
  }
  const ref = query.data;
  const bundles = ref.bundles ?? [];
  const executables = ref.executables ?? [];
  const identities = ref.signing_identities ?? [];
  const certificates = ref.certificates ?? [];
  const hasSantaData =
    ref.execution_count > 0 ||
    ref.block_count > 0 ||
    bundles.length > 0 ||
    executables.length > 0 ||
    identities.length > 0 ||
    certificates.length > 0;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Santa</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-3">
          <SantaMetric label="Executions" value={ref.execution_count} />
          <SantaMetric label="Blocks" value={ref.block_count} />
          <SantaMetric label="Bundles" value={bundles.length} />
        </div>

        {!hasSantaData ? (
          <EmptyPanel>No Santa data yet</EmptyPanel>
        ) : (
          <div className="grid gap-5 xl:grid-cols-2">
            <SantaBundlesTable bundles={bundles} isAdmin={isAdmin} />
            <SantaExecutablesTable executables={executables} isAdmin={isAdmin} />
            <SantaSigningTable identities={identities} isAdmin={isAdmin} />
            <SantaCertificatesTable certificates={certificates} isAdmin={isAdmin} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
function SantaMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs font-semibold text-muted-foreground">{label}</div>
      <div className="text-lg font-semibold text-foreground tabular-nums">{value}</div>
    </div>
  );
}
function SantaBundlesTable({ bundles, isAdmin }: { bundles: BundleReference[]; isAdmin: boolean }) {
  return (
    <SantaReferenceTable title="Bundles" empty="No related bundles." count={bundles.length}>
      {bundles.map((bundle) => (
        <TableRow key={bundle.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">{bundle.name || "-"}</div>
            <div className="truncate text-xs text-muted-foreground">{bundle.bundle_id || "-"}</div>
          </TableCell>
          <TableCell className="text-right text-xs text-muted-foreground tabular-nums">
            {bundle.collected_binary_count}/{bundle.binary_count}
          </TableCell>
          <TableCell className="w-10 text-right">
            {isAdmin && bundle.complete ? (
              <QuickAddRuleButton
                ruleType="bundle"
                identifier={bundle.sha256}
                name={bundle.name || undefined}
              />
            ) : null}
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}
function SantaExecutablesTable({
  executables,
  isAdmin,
}: {
  executables: ExecutableReference[];
  isAdmin: boolean;
}) {
  return (
    <SantaReferenceTable
      title="Executables"
      empty="No related executables."
      count={executables.length}
    >
      {executables.map((executable) => (
        <TableRow key={executable.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">{executable.file_name || "-"}</div>
            <div className="truncate font-mono text-xs text-muted-foreground">
              {executable.sha256}
            </div>
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">
            <span>{executable.execution_count}</span>
            <span className="text-muted-foreground"> / </span>
            <span>{executable.block_count}</span>
          </TableCell>
          <TableCell className="w-10 text-right">
            {isAdmin ? (
              <QuickAddRuleButton
                ruleType="binary"
                identifier={executable.sha256}
                name={executable.file_name || undefined}
              />
            ) : null}
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}
function SantaSigningTable({
  identities,
  isAdmin,
}: {
  identities: SigningIdentityReference[];
  isAdmin: boolean;
}) {
  return (
    <SantaReferenceTable title="Signing" empty="No signing identities." count={identities.length}>
      {identities.map((identity) => (
        <TableRow key={`${identity.rule_type}:${identity.identifier}`}>
          <TableCell className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <Badge variant="secondary" className="font-normal">
                {ruleTypeLabel(identity.rule_type)}
              </Badge>
              <span className="truncate font-medium">{identity.identifier}</span>
            </div>
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">{identity.rule_count}</TableCell>
          <TableCell className="w-10 text-right">
            {isAdmin ? (
              <QuickAddRuleButton ruleType={identity.rule_type} identifier={identity.identifier} />
            ) : null}
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}
function SantaCertificatesTable({
  certificates,
  isAdmin,
}: {
  certificates: CertificateReference[];
  isAdmin: boolean;
}) {
  return (
    <SantaReferenceTable
      title="Certificates"
      empty="No signing certificates."
      count={certificates.length}
    >
      {certificates.map((certificate) => (
        <TableRow key={certificate.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">{certificate.common_name || "-"}</div>
            {certificate.organization ? (
              <div className="truncate text-xs text-muted-foreground">
                Organization: {certificate.organization}
              </div>
            ) : null}
            {certificate.organizational_unit ? (
              <div className="truncate text-xs text-muted-foreground">
                Organizational Unit: {certificate.organizational_unit}
              </div>
            ) : null}
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">
            {certificate.rule_count}
          </TableCell>
          <TableCell className="w-10 text-right">
            {isAdmin ? (
              <QuickAddRuleButton
                ruleType="certificate"
                identifier={certificate.sha256}
                name={certificate.common_name}
              />
            ) : null}
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}
function SantaReferenceTable({
  title,
  empty,
  count,
  children,
}: {
  title: string;
  empty: string;
  count: number;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <h2 className="text-sm font-medium">{title}</h2>
      {count > 0 ? (
        <div className="rounded-md border">
          <Table>
            <TableBody>{children}</TableBody>
          </Table>
        </div>
      ) : (
        <EmptyPanel>{empty}</EmptyPanel>
      )}
    </div>
  );
}
function QuickAddRuleButton({
  ruleType,
  identifier,
  name,
}: {
  ruleType: SantaRuleType;
  identifier: string;
  name?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            render={
              <Link to="/santa/rules/new" search={{ rule_type: ruleType, identifier, name }} />
            }
            nativeButton={false}
            type="button"
            variant="ghost"
            size="icon-sm"
          />
        }
      >
        <Plus />
      </TooltipTrigger>
      <TooltipContent>New {ruleTypeLabel(ruleType)} Rule</TooltipContent>
    </Tooltip>
  );
}
function SoftwareVersionsCard({ title }: { title: SoftwareTitle }) {
  const versions = title.versions ?? [];
  return (
    <Card>
      <CardHeader>
        <CardTitle>Versions</CardTitle>
      </CardHeader>
      <CardContent>
        {versions.length === 0 ? (
          <EmptyPanel>No versions yet</EmptyPanel>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Version</TableHead>
                  <TableHead className="text-right">Hosts</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {versions.map((v) => (
                  <VersionRow key={v.id} title={title} version={v} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
function VersionRow({ title, version }: { title: SoftwareTitle; version: SoftwareVersion }) {
  return (
    <TableRow>
      <TableCell className="font-medium">{version.version || "-"}</TableCell>
      <TableCell className="text-right tabular-nums">
        <Link
          to="/hosts"
          search={{ software_title_id: title.id, software_id: version.id }}
          className="underline decoration-dotted underline-offset-4 hover:decoration-solid focus-visible:decoration-solid"
        >
          {version.hosts_count}
        </Link>
      </TableCell>
    </TableRow>
  );
}
