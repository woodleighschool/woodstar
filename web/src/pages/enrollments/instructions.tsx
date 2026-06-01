import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import { ExternalLink } from "lucide-react";
import type { ReactNode } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

import type { Integration } from "./types";

const FLEETCTL_INSTALL_URL = "https://fleetdm.com/guides/fleetctl#installing-fleetctl";
const USER_EMAIL_PLACEHOLDER = "REPLACE_WITH_ASSIGNED_USER_EMAIL";
const PUBLIC_URL_PLACEHOLDER = "REPLACE_WITH_WOODSTAR_PUBLIC_URL";
const xmlExtensions: Extension[] = [xml()];

export function DeploymentInstructions({ integration, publicURL }: { integration: Integration; publicURL?: string }) {
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
      <div className="text-muted-foreground max-w-3xl space-y-2 text-sm leading-relaxed">
        <p>
          Build one Orbit package and reuse it. Put the Woodstar URL, enroll secret, and assigned user email in the
          configuration profile so the package can move between environments.
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
        title="Configuration Profile"
        description={`The profile points Orbit at Woodstar, supplies the enroll secret, and maps the assigned user. Replace ${USER_EMAIL_PLACEHOLDER} with your MDM's user-email variable.`}
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
      <div className="text-muted-foreground max-w-3xl space-y-2 text-sm leading-relaxed">
        <p>Deploy Santa with a configuration profile that points sync at Woodstar and sends a bearer enroll secret.</p>
      </div>

      <DeploymentArtifact
        title="Santa Configuration Profile"
        description="Replace the bearer secret with an active Santa enrollment secret before deployment."
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
      <div className="text-muted-foreground max-w-3xl space-y-2 text-sm leading-relaxed">
        <p>Deploy Munki with a configuration profile that points ManagedInstalls at Woodstar.</p>
      </div>

      <DeploymentArtifact
        title="Munki Configuration Profile"
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
        <div className="min-w-0 space-y-1">
          <h2 id={titleID} className="text-base font-medium">
            {title}
          </h2>
          <p className="text-muted-foreground max-w-3xl text-sm leading-relaxed">{description}</p>
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
          : "min-h-9 [&_.cm-content]:py-2 [&_.cm-scroller]:overflow-x-auto [&_.cm-line]:whitespace-pre",
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

function orbitProfileTemplate(publicURL: string | undefined) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>EnrollSecret</key>
      <string>REPLACE_WITH_SECRET</string>
      <key>FleetURL</key>
      <string>${publicURL ?? PUBLIC_URL_PLACEHOLDER}</string>
      <key>PayloadDisplayName</key>
      <string>Fleetd Configuration</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.fleetd.config</string>
      <key>PayloadType</key>
      <string>com.fleetdm.fleetd.config</string>
      <key>PayloadUUID</key>
      <string>476F5334-D501-4768-9A31-1A18A4E1E807</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
    <dict>
      <key>EndUserEmail</key>
      <string>${USER_EMAIL_PLACEHOLDER}</string>
      <key>PayloadDisplayName</key>
      <string>Orbit User Mapping</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.fleet.mdm.apple.mdm</string>
      <key>PayloadType</key>
      <string>au.vic.edu.woodleigh.woodstar.orbit.user</string>
      <key>PayloadUUID</key>
      <string>29713130-1602-4D27-90C9-B822A295E44E</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Orbit and osquery</string>
  <key>PayloadIdentifier</key>
  <string>au.vic.edu.woodleigh.woodstar.orbit</string>
  <key>PayloadOrganization</key>
  <string>Woodleigh School</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>`;
}

function santaProfileTemplate(publicURL: string | undefined) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadDisplayName</key>
      <string>Santa (North Pole Security)</string>
      <key>PayloadIdentifier</key>
      <string>com.northpolesec.santa.4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA</string>
      <key>PayloadType</key>
      <string>com.northpolesec.santa</string>
      <key>PayloadUUID</key>
      <string>4BB570FE-55D7-46C1-BFE9-BAD4BC2763CA</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>ClientMode</key>
      <integer>1</integer>
      <key>SyncBaseURL</key>
      <string>${publicURL ? `${publicURL}/santa/sync` : `${PUBLIC_URL_PLACEHOLDER}/santa/sync`}</string>
      <key>SyncClientContentEncoding</key>
      <string>gzip</string>
      <key>SyncEnableProtoTransfer</key>
      <true/>
      <key>SyncExtraHeaders</key>
      <dict>
        <key>Authorization</key>
        <string>Bearer REPLACE_WITH_SECRET</string>
      </dict>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Santa</string>
  <key>PayloadIdentifier</key>
  <string>au.vic.edu.woodleigh.woodstar.santa</string>
  <key>PayloadOrganization</key>
  <string>Woodleigh School</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>7CE340DE-AAB6-448B-A558-EB3C49A3A687</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>`;
}

function munkiProfileTemplate(publicURL: string | undefined) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadContent</key>
      <dict>
        <key>ManagedInstalls</key>
        <dict>
          <key>Forced</key>
          <array>
            <dict>
              <key>mcx_preference_settings</key>
              <dict>
                <key>SoftwareRepoURL</key>
                <string>${publicURL ? `${publicURL}/munki` : `${PUBLIC_URL_PLACEHOLDER}/munki`}</string>
                <key>ClientIdentifier</key>
                <string>$SERIALNUMBER</string>
                <key>FollowHTTPRedirects</key>
                <string>all</string>
                <key>AdditionalHttpHeaders</key>
                <array>
                  <string>Authorization: Bearer REPLACE_WITH_SECRET</string>
                  <string>Serial: $SERIALNUMBER</string>
                </array>
              </dict>
            </dict>
          </array>
        </dict>
      </dict>
      <key>PayloadDisplayName</key>
      <string>Munki ManagedInstalls</string>
      <key>PayloadIdentifier</key>
      <string>au.vic.edu.woodleigh.woodstar.munki.managedinstalls</string>
      <key>PayloadType</key>
      <string>com.apple.ManagedClient.preferences</string>
      <key>PayloadUUID</key>
      <string>EF6B0B39-B2BE-44F7-A2B5-5F49282B221D</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Munki</string>
  <key>PayloadIdentifier</key>
  <string>au.vic.edu.woodleigh.woodstar.munki</string>
  <key>PayloadOrganization</key>
  <string>Woodleigh School</string>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>56E74DA2-6F02-4E85-8C95-BA51C34F88F0</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>`;
}
