import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import { ExternalLink } from "lucide-react";
import type { PlistValue } from "plist";
import * as plist from "plist";
import type { ReactNode } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { Integration } from "@/lib/enrollments";
import { cn } from "@/lib/utils";

const FLEETCTL_INSTALL_URL = "https://fleetdm.com/guides/fleetctl#installing-fleetctl";
const PUBLIC_URL_PLACEHOLDER = "REPLACE_WITH_PUBLIC_URL";
const SECRET_PLACEHOLDER = "REPLACE_WITH_SECRET";
const EMAIL_PLACEHOLDER = "$EMAIL";
const SERIAL_PLACEHOLDER = "$SERIALNUMBER";
const ORG = "Woodleigh School";
const PREFIX = "au.vic.edu.woodleigh.woodstar";
const xmlExtensions: Extension[] = [xml()];

type PlistDict = { [key: string]: PlistValue };

export function DeploymentInstructions({
  integration,
  publicURL,
}: {
  integration: Integration;
  publicURL?: string;
}) {
  if (integration === "orbit") {
    return <OrbitDeploymentInstructions publicURL={publicURL} />;
  }
  if (integration === "munki") {
    return <MunkiDeploymentInstructions publicURL={publicURL} />;
  }

  return <SantaDeploymentInstructions publicURL={publicURL} />;
}

function OrbitDeploymentInstructions({ publicURL }: { publicURL?: string }) {
  return (
    <section className="grid gap-6">
      <div className="flex max-w-3xl flex-col gap-2 text-sm leading-relaxed text-muted-foreground">
        <p>
          Build the Orbit package once. Keep the Woodstar URL, enroll secret, and user email in the
          profile.
        </p>
      </div>

      <DeploymentArtifact
        title="Orbit Package"
        description="fleetctl produces an Orbit package for software distribution. It does not embed the Woodstar URL or enroll secret."
        value={orbitPackageCommand()}
        action={
          <Button type="button" variant="outline" size="sm" asChild>
            <a href={FLEETCTL_INSTALL_URL} target="_blank" rel="noreferrer">
              <ExternalLink data-icon="inline-start" />
              Install Fleetctl
            </a>
          </Button>
        }
      />

      <DeploymentArtifact
        title="Orbit Profile"
        description={`Sets the Woodstar URL, enroll secret, and assigned user. ${EMAIL_PLACEHOLDER} should be your MDM user-email variable.`}
        value={orbitProfileTemplate(publicURL)}
        extensions={xmlExtensions}
        multiline
      />
    </section>
  );
}

function SantaDeploymentInstructions({ publicURL }: { publicURL?: string }) {
  return (
    <section className="grid gap-6">
      <div className="flex max-w-3xl flex-col gap-2 text-sm leading-relaxed text-muted-foreground">
        <p>Deploy Santa with a profile for Woodstar and a bearer secret.</p>
        <p>MachineID identifies Santa clients. Leave it unset so Santa uses the hardware UUID.</p>
      </div>

      <DeploymentArtifact
        title="Santa Profile"
        description="Replace the bearer secret before deployment."
        value={santaProfileTemplate(publicURL)}
        extensions={xmlExtensions}
        multiline
      />
    </section>
  );
}

function MunkiDeploymentInstructions({ publicURL }: { publicURL?: string }) {
  return (
    <section className="grid gap-6">
      <div className="flex max-w-3xl flex-col gap-2 text-sm leading-relaxed text-muted-foreground">
        <p>Deploy Munki with a profile for Woodstar.</p>
      </div>

      <DeploymentArtifact
        title="Munki Profile"
        description="Replace the bearer secret before deployment. The serial header should use the MDM-expanded device serial."
        value={munkiProfileTemplate(publicURL)}
        extensions={xmlExtensions}
        multiline
      />
    </section>
  );
}

function DeploymentArtifact({
  title,
  description,
  value,
  extensions,
  multiline = false,
  action,
}: {
  title: string;
  description: string;
  value: string;
  extensions?: Extension[];
  multiline?: boolean;
  action?: ReactNode;
}) {
  const titleID = deploymentArtifactTitleID(title);

  return (
    <Card className="grid min-w-0 gap-3 rounded-lg p-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 id={titleID} className="text-base font-medium">
            {title}
          </h2>
          <p className="max-w-3xl text-sm leading-relaxed text-muted-foreground">{description}</p>
        </div>
        {action}
      </div>
      <DeploymentExample value={value} extensions={extensions} multiline={multiline} />
    </Card>
  );
}

function DeploymentExample({
  value,
  extensions,
  multiline = false,
}: {
  value: string;
  extensions?: Extension[];
  multiline?: boolean;
}) {
  return (
    <CodeEditor
      value={value}
      onChange={() => null}
      extensions={extensions}
      readOnly
      lineNumbers={false}
      lineWrapping={false}
      highlightActiveLine={false}
      className={cn(
        multiline
          ? "max-h-96 min-h-56 overflow-auto [&_.cm-content]:py-1.5"
          : "min-h-9 [&_.cm-content]:py-2 [&_.cm-line]:whitespace-pre [&_.cm-scroller]:overflow-x-auto",
      )}
    />
  );
}

function orbitPackageCommand() {
  return "fleetctl package --type=pkg --use-system-configuration";
}

function deploymentArtifactTitleID(title: string) {
  return `deployment-${title.toLowerCase().replaceAll(" ", "-")}`;
}

function profile(opts: {
  id: string;
  uuid: string;
  displayName: string;
  description?: string;
  payloads: PlistDict[];
}) {
  return plist.build({
    PayloadContent: opts.payloads,
    PayloadDisplayName: opts.displayName,
    PayloadIdentifier: opts.id,
    PayloadOrganization: ORG,
    PayloadScope: "System",
    PayloadType: "Configuration",
    PayloadUUID: opts.uuid,
    PayloadVersion: 1,
    ...(opts.description ? { PayloadDescription: opts.description } : {}),
  });
}

function payloadMeta(opts: {
  id: string;
  uuid: string;
  type: string;
  displayName: string;
}): PlistDict {
  return {
    PayloadDisplayName: opts.displayName,
    PayloadIdentifier: opts.id,
    PayloadType: opts.type,
    PayloadUUID: opts.uuid,
    PayloadVersion: 1,
  };
}

function publicURL(publicURL: string | undefined) {
  return (publicURL ?? PUBLIC_URL_PLACEHOLDER).replace(/\/+$/, "");
}

function woodstarURL(publicURLValue: string | undefined, path: string) {
  return `${publicURL(publicURLValue)}${path}`;
}

function orbitProfileTemplate(publicURLValue: string | undefined) {
  return profile({
    id: `${PREFIX}.orbit`,
    uuid: "0C6AFB45-01B6-4E19-944A-123CD16381C7",
    displayName: "Woodstar - Orbit",
    description: "Configures Orbit for Woodstar.",
    payloads: [
      {
        EnrollSecret: SECRET_PLACEHOLDER,
        FleetURL: publicURL(publicURLValue),
        ...payloadMeta({
          id: "com.fleetdm.fleetd.config",
          type: "com.fleetdm.fleetd.config",
          uuid: "476F5334-D501-4768-9A31-1A18A4E1E807",
          displayName: "Orbit",
        }),
      },
      {
        EndUserEmail: EMAIL_PLACEHOLDER,
        ...payloadMeta({
          id: "com.fleetdm.fleet.mdm.apple.mdm",
          type: `${PREFIX}.orbit.user`,
          uuid: "29713130-1602-4D27-90C9-B822A295E44E",
          displayName: "Orbit User Mapping",
        }),
      },
    ],
  });
}

function santaProfileTemplate(publicURLValue: string | undefined) {
  return profile({
    id: `${PREFIX}.santa`,
    uuid: "7CE340DE-AAB6-448B-A558-EB3C49A3A687",
    displayName: "Woodstar - Santa",
    description: "Configures Santa for Woodstar.",
    payloads: [
      {
        ClientMode: 1,
        SyncBaseURL: woodstarURL(publicURLValue, "/santa/sync"),
        SyncClientContentEncoding: "gzip",
        SyncEnableProtoTransfer: true,
        SyncExtraHeaders: {
          Authorization: `Bearer ${SECRET_PLACEHOLDER}`,
        },
        ...payloadMeta({
          id: "com.northpolesec.santa.4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA",
          type: "com.northpolesec.santa",
          uuid: "4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA",
          displayName: "Santa",
        }),
      },
    ],
  });
}

function munkiProfileTemplate(publicURLValue: string | undefined) {
  return profile({
    id: `${PREFIX}.munki`,
    uuid: "56E74DA2-6F02-4E85-8C95-BA51C34F88F0",
    displayName: "Woodstar - Munki",
    description: "Configures Munki for Woodstar.",
    payloads: [
      {
        SoftwareRepoURL: woodstarURL(publicURLValue, "/munki"),
        ClientIdentifier: SERIAL_PLACEHOLDER,
        FollowHTTPRedirects: "all",
        AdditionalHttpHeaders: [
          `Authorization: Bearer ${SECRET_PLACEHOLDER}`,
          `Serial: ${SERIAL_PLACEHOLDER}`,
        ],
        ...payloadMeta({
          id: `${PREFIX}.munki.managedinstalls`,
          type: "ManagedInstalls",
          uuid: "EF6B0B39-B2BE-44F7-A2B5-5F49282B221D",
          displayName: "Munki",
        }),
      },
    ],
  });
}
