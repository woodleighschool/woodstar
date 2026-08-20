import type { ReactNode } from "react";

import { TextLink } from "@components/link";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";
import type { Host, HostDetail, PublicIpDetails } from "@lib/api";

export function HostPublicIP({
  host,
  showAddress = false,
}: {
  host: Host | HostDetail;
  showAddress?: boolean;
}) {
  const address = host.public_ip;
  const details = host.public_ip_details;
  const distributionPoint = details?.distribution_point;
  const hasGeoIP = hasGeoIPDetails(details);

  if (!address) return "-";
  if (!distributionPoint && !hasGeoIP) {
    return <span className="block truncate whitespace-nowrap">{address}</span>;
  }

  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <button
            type="button"
            className="block max-w-full cursor-default truncate underline decoration-dotted underline-offset-4"
          >
            {showAddress ? address : (distributionPoint?.name ?? address)}
          </button>
        }
      />
      <HoverCardContent align="start" className="w-80">
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
          <PublicIPCardRow label="Public IP" value={address} />
          {distributionPoint ? (
            <PublicIPCardRow
              label="Distribution Point"
              value={
                <TextLink
                  to="/munki/distribution-points/$id"
                  params={{ id: String(distributionPoint.id) }}
                  className="font-medium"
                >
                  {distributionPoint.name}
                </TextLink>
              }
            />
          ) : null}
          {hasGeoIP ? (
            <>
              <PublicIPCardRow label="City" value={details.city} />
              <PublicIPCardRow label="Region" value={details.region} />
              <PublicIPCardRow
                label="Country"
                value={`${details.country} (${details.country_code})`}
              />
              <PublicIPCardRow label="Coordinates" value={formatCoordinates(details)} />
              <PublicIPCardRow label="Network" value={`AS${details.asn} ${details.organization}`} />
            </>
          ) : null}
        </dl>
        {hasGeoIP ? <DBIPAttribution className="mt-3 block" /> : null}
      </HoverCardContent>
    </HoverCard>
  );
}

function hasGeoIPDetails(
  details: PublicIpDetails | undefined,
): details is Required<Omit<PublicIpDetails, "distribution_point">> & PublicIpDetails {
  return Boolean(
    details?.city &&
    details.region &&
    details.country &&
    details.country_code &&
    details.latitude !== undefined &&
    details.longitude !== undefined &&
    details.asn &&
    details.organization,
  );
}

function formatCoordinates(details: Pick<PublicIpDetails, "latitude" | "longitude">) {
  if (details.latitude === undefined || details.longitude === undefined) return "";
  return `${details.latitude.toFixed(4)}, ${details.longitude.toFixed(4)}`;
}

function DBIPAttribution({ className }: { className?: string }) {
  return (
    <a
      href="https://db-ip.com"
      target="_blank"
      rel="noreferrer"
      className={`text-primary hover:underline ${className ?? ""}`}
    >
      IP Geolocation by DB-IP
    </a>
  );
}

function PublicIPCardRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="contents">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-right wrap-break-word">{value}</dd>
    </div>
  );
}
