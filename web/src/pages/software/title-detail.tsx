import { Link, useParams } from "@tanstack/react-router";
import { Loader2, Package, Plus } from "lucide-react";
import type { ReactNode } from "react";

import { PageShell } from "@/components/layout/page-layout";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useSoftwareSantaReference,
  useSoftwareTitle,
  type SoftwareSantaReference,
  type SoftwareTitle,
  type SoftwareVersion,
} from "@/hooks/use-software";
import { softwareSourceLabel } from "@/lib/software-source-labels";
import { formatRelative } from "@/lib/utils";
import { ruleTypeLabel, type RuleType } from "@/pages/santa/rules/shared";

type BundleReference = NonNullable<SoftwareSantaReference["bundles"]>[number];
type CertificateReference = NonNullable<SoftwareSantaReference["certificates"]>[number];
type ExecutableReference = NonNullable<SoftwareSantaReference["executables"]>[number];
type RuleReference = NonNullable<SoftwareSantaReference["rules"]>[number];
type SigningIdentityReference = NonNullable<SoftwareSantaReference["signing_identities"]>[number];

export function SoftwareTitleDetailPage() {
  const { softwareId } = useParams({ from: "/_authenticated/software/titles/$softwareId" });
  const query = useSoftwareTitle(Number(softwareId));
  const title = query.data;

  if (query.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Software Title</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </PageShell>
    );
  }

  if (query.isLoading || !title) {
    if (query.isLoading) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </PageShell>
      );
    }
    return (
      <PageShell>
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Package />
            </EmptyMedia>
            <EmptyTitle>Software Title Not Found</EmptyTitle>
            <EmptyDescription>This title is no longer available.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </PageShell>
    );
  }

  return (
    <PageShell className="gap-6">
      <SoftwareHeader title={title} />
      <SoftwareInfoCard title={title} />
      <SoftwareSantaCard titleID={title.id} />
      <SoftwareVersionsCard title={title} />
    </PageShell>
  );
}

function SoftwareHeader({ title }: { title: SoftwareTitle }) {
  const displayName = title.display_name || title.name;
  const typeLabel = softwareSourceLabel(title.source, title.extension_for);

  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex min-w-0 items-center gap-4">
        <SoftwareIcon source={title.source} size="lg" />
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-foreground truncate text-xl font-semibold" title={displayName}>
              {displayName}
            </h1>
            <Badge variant="secondary" className="font-normal">
              {typeLabel}
            </Badge>
          </div>
          <p className="text-muted-foreground text-xs">
            {title.counts_updated_at ? (
              <span title={new Date(title.counts_updated_at).toLocaleString()}>
                Counts updated {formatRelative(title.counts_updated_at)}
              </span>
            ) : (
              "Counts not yet computed"
            )}
          </p>
        </div>
      </div>
      <Button asChild variant="outline" size="sm">
        <Link to="/hosts" search={{ software_title_id: title.id.toString() }}>
          View hosts
        </Link>
      </Button>
    </div>
  );
}

interface Tile {
  label: string;
  value: ReactNode;
}

function SoftwareInfoCard({ title }: { title: SoftwareTitle }) {
  const tiles: Tile[] = [];

  if (title.bundle_identifier) {
    tiles.push({
      label: "Bundle Identifier",
      value: title.bundle_identifier,
    });
  }
  if (title.browser) {
    tiles.push({ label: "Browser", value: title.browser });
  }
  if (title.extension_for) {
    tiles.push({ label: "Extension for", value: title.extension_for });
  }
  tiles.push({ label: "Hosts", value: <span className="tabular-nums">{title.hosts_count}</span> });
  tiles.push({ label: "Versions", value: <span className="tabular-nums">{title.versions_count}</span> });

  tiles.sort((a, b) => a.label.localeCompare(b.label));

  return (
    <Card>
      <CardContent className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
        {tiles.map((t) => (
          <div key={t.label} className="flex min-w-0 flex-col gap-1">
            <dt className="text-muted-foreground text-xs font-semibold">{t.label}</dt>
            <dd className="text-foreground truncate text-sm">{t.value}</dd>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function SoftwareSantaCard({ titleID }: { titleID: number }) {
  const query = useSoftwareSantaReference(titleID);

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to Load Santa Data</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    );
  }

  if (query.isLoading || !query.data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Santa</CardTitle>
        </CardHeader>
        <CardContent className="text-muted-foreground flex items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </CardContent>
      </Card>
    );
  }

  const ref = query.data;
  const bundles = ref.bundles ?? [];
  const executables = ref.executables ?? [];
  const identities = ref.signing_identities ?? [];
  const certificates = ref.certificates ?? [];
  const rules = ref.rules ?? [];
  const hasSantaData =
    ref.execution_count > 0 ||
    ref.block_count > 0 ||
    bundles.length > 0 ||
    executables.length > 0 ||
    identities.length > 0 ||
    certificates.length > 0 ||
    rules.length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Santa</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-3">
          <SantaMetric label="Executions" value={ref.execution_count} />
          <SantaMetric label="Blocks" value={ref.block_count} />
          <SantaMetric label="Rules" value={rules.length} />
          <SantaMetric label="Bundles" value={bundles.length} />
        </div>

        {!hasSantaData ? (
          <div className="bg-muted/30 rounded-md border border-dashed px-4 py-6 text-sm">
            <p className="font-medium">No Santa Data</p>
            <p className="text-muted-foreground">No related executions, bundles, signing identities, or rules.</p>
          </div>
        ) : (
          <div className="grid gap-5 xl:grid-cols-2">
            <SantaBundlesTable bundles={bundles} />
            <SantaExecutablesTable executables={executables} />
            <SantaSigningTable identities={identities} />
            <SantaCertificatesTable certificates={certificates} />
            <SantaRulesTable rules={rules} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function SantaMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-muted-foreground text-xs font-semibold">{label}</div>
      <div className="text-foreground text-lg font-semibold tabular-nums">{value}</div>
    </div>
  );
}

function SantaBundlesTable({ bundles }: { bundles: BundleReference[] }) {
  return (
    <SantaReferenceTable title="Bundles" empty="No related bundles.">
      {bundles.map((bundle) => (
        <TableRow key={bundle.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">
              {bundle.name || bundle.bundle_id || shortIdentifier(bundle.sha256)}
            </div>
            <div className="text-muted-foreground truncate text-xs">{bundle.bundle_id || bundle.path}</div>
          </TableCell>
          <TableCell className="text-muted-foreground text-right text-xs tabular-nums">
            {bundle.collected_binary_count}/{bundle.binary_count}
          </TableCell>
          <TableCell className="w-10 text-right">
            {bundle.complete ? (
              <QuickAddRuleButton
                targetType="bundle"
                identifier={bundle.sha256}
                name={bundle.name || bundle.bundle_id}
              />
            ) : null}
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}

function SantaExecutablesTable({ executables }: { executables: ExecutableReference[] }) {
  return (
    <SantaReferenceTable title="Executables" empty="No related executables.">
      {executables.map((executable) => (
        <TableRow key={executable.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">{executableDisplayName(executable)}</div>
            <div className="text-muted-foreground truncate font-mono text-xs">{shortIdentifier(executable.sha256)}</div>
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">
            <span>{executable.execution_count}</span>
            <span className="text-muted-foreground"> / </span>
            <span>{executable.block_count}</span>
          </TableCell>
          <TableCell className="w-10 text-right">
            <QuickAddRuleButton targetType="binary" identifier={executable.sha256} name={executable.file_bundle_name} />
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}

function SantaSigningTable({ identities }: { identities: SigningIdentityReference[] }) {
  return (
    <SantaReferenceTable title="Signing" empty="No signing identities.">
      {identities.map((identity) => (
        <TableRow key={`${identity.target_type}:${identity.identifier}`}>
          <TableCell className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <Badge variant="secondary" className="font-normal">
                {ruleTypeLabel(identity.target_type)}
              </Badge>
              <span className="truncate font-medium">{identity.identifier}</span>
            </div>
            <div className="text-muted-foreground truncate text-xs">{identity.name}</div>
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">{identity.rule_count}</TableCell>
          <TableCell className="w-10 text-right">
            <QuickAddRuleButton
              targetType={identity.target_type}
              identifier={identity.identifier}
              name={identity.name || identity.identifier}
            />
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}

function executableDisplayName(executable: ExecutableReference) {
  const bundleName = executable.file_bundle_name ?? "";
  if (bundleName.trim() !== "") return bundleName;
  if (executable.file_name.trim() !== "") return executable.file_name;
  return "Executable";
}

function SantaCertificatesTable({ certificates }: { certificates: CertificateReference[] }) {
  return (
    <SantaReferenceTable title="Certificates" empty="No signing certificates.">
      {certificates.map((certificate) => (
        <TableRow key={certificate.sha256}>
          <TableCell className="min-w-0">
            <div className="truncate font-medium">{certificate.common_name || shortIdentifier(certificate.sha256)}</div>
            <div className="text-muted-foreground truncate text-xs">
              {certificate.organizational_unit || certificate.organization || shortIdentifier(certificate.sha256)}
            </div>
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">{certificate.rule_count}</TableCell>
          <TableCell className="w-10 text-right">
            <QuickAddRuleButton
              targetType="certificate"
              identifier={certificate.sha256}
              name={certificate.common_name}
            />
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}

function SantaRulesTable({ rules }: { rules: RuleReference[] }) {
  return (
    <SantaReferenceTable title="Rules" empty="No matching rules.">
      {rules.map((rule) => (
        <TableRow key={rule.id}>
          <TableCell className="min-w-0">
            <Link
              to="/santa/rules/$ruleId/edit"
              params={{ ruleId: String(rule.id) }}
              className="hover:text-primary block truncate font-medium hover:underline"
            >
              {rule.name || rule.identifier}
            </Link>
            <div className="text-muted-foreground truncate font-mono text-xs">{shortIdentifier(rule.identifier)}</div>
          </TableCell>
          <TableCell className="text-right">
            <Badge variant="secondary" className="font-normal">
              {ruleTypeLabel(rule.rule_type)}
            </Badge>
          </TableCell>
        </TableRow>
      ))}
    </SantaReferenceTable>
  );
}

function SantaReferenceTable({ title, empty, children }: { title: string; empty: string; children: ReactNode }) {
  const hasRows = Array.isArray(children) ? children.length > 0 : Boolean(children);
  return (
    <div className="min-w-0 space-y-2">
      <h2 className="text-sm font-medium">{title}</h2>
      {hasRows ? (
        <div className="rounded-md border">
          <Table>
            <TableBody>{children}</TableBody>
          </Table>
        </div>
      ) : (
        <div className="bg-muted/30 rounded-md border border-dashed px-3 py-4 text-sm">{empty}</div>
      )}
    </div>
  );
}

function QuickAddRuleButton({
  targetType,
  identifier,
  name,
}: {
  targetType: RuleType;
  identifier: string;
  name?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button asChild type="button" variant="ghost" size="icon-sm">
          <Link to="/santa/rules/new" search={{ rule_type: targetType, identifier, name }}>
            <Plus />
          </Link>
        </Button>
      </TooltipTrigger>
      <TooltipContent>New {ruleTypeLabel(targetType)} Rule</TooltipContent>
    </Tooltip>
  );
}

function shortIdentifier(identifier: string) {
  if (identifier.length <= 28) return identifier;
  return `${identifier.slice(0, 12)}...${identifier.slice(-8)}`;
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
          <div className="bg-muted/30 rounded-md border border-dashed px-4 py-6 text-sm">
            <p className="font-medium">No Versions Discovered</p>
            <p className="text-muted-foreground">Hosts have not reported a concrete version.</p>
          </div>
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
                  <VersionRow key={v.id} version={v} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function VersionRow({ version }: { version: SoftwareVersion }) {
  return (
    <TableRow>
      <TableCell className="font-medium">{version.version || "-"}</TableCell>
      <TableCell className="text-right tabular-nums">
        <Link
          to="/hosts"
          search={{ software_id: version.id.toString() }}
          className="hover:text-primary hover:underline"
        >
          {version.hosts_count}
        </Link>
      </TableCell>
    </TableRow>
  );
}
