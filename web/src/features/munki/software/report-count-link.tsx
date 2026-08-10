import { TextLink } from "@components/link";

export function MunkiSoftwareReportCountLink({
  softwareID,
  installed,
  expected,
}: {
  softwareID: number;
  installed: number;
  expected: number;
}) {
  return (
    <TextLink
      to="/munki/software/$id"
      params={{ id: String(softwareID) }}
      search={{ tab: "report", status: ["installed"] }}
      className="w-fit tabular-nums"
    >
      {installed} / {expected}
    </TextLink>
  );
}
