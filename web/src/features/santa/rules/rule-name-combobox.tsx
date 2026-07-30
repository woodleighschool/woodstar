import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

import { FreeTextCombobox } from "@components/free-text-combobox";
import { Badge } from "@components/ui/badge";
import { useSoftware } from "@features/software/queries";
import { useDebouncedCallback } from "@hooks/use-debounced-callback";
import type { SoftwareTitle } from "@lib/api";
import { cn } from "@lib/utils";

import { RULE_TYPES, type SantaRuleType } from "./metadata";

type CandidateRuleType = Extract<SantaRuleType, "signingid" | "teamid">;

interface RuleNameCandidate {
  key: string;
  name: string;
  ruleType: CandidateRuleType;
  identifier: string;
}

export function RuleNameCombobox({
  id,
  name,
  value,
  invalid,
  onBlur,
  onChange,
  onSelect,
}: {
  id: string;
  name: string;
  value: string;
  invalid?: boolean;
  onBlur: () => void;
  onChange: (value: string) => void;
  onSelect: (candidate: RuleNameCandidate) => void;
}) {
  const input = value.trim();
  const [q, setQ] = useState(input);
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const updateQuery = useDebouncedCallback((next: string) => setQ(next), 200);
  useEffect(() => {
    if (input === "") {
      updateQuery.cancel();
      setQ("");
      return;
    }
    updateQuery(input);
  }, [input, updateQuery]);

  const query = useSoftware({ q, per_page: 20 }, { enabled: q !== "" });
  const searchPending = q !== input || query.isPending || query.isPlaceholderData;
  const candidates = useMemo(
    () =>
      q === "" || searchPending || !query.isSuccess || selectedName === input
        ? []
        : ruleNameCandidates(query.data.items),
    [input, q, query.data, query.isSuccess, searchPending, selectedName],
  );

  return (
    <FreeTextCombobox
      mode="free-text"
      id={id}
      name={name}
      value={value}
      items={candidates}
      placeholder="Visual Studio Code"
      invalid={invalid}
      loading={input !== "" && searchPending}
      filterItems={false}
      itemToStringValue={candidateName}
      itemKey={candidateValue}
      renderItem={renderCandidate}
      onBlur={onBlur}
      onChange={(next) => {
        if (next !== selectedName) setSelectedName(null);
        onChange(next);
      }}
      onSelectItem={(candidate) => {
        setSelectedName(candidate.name);
        onSelect(candidate);
      }}
    />
  );
}

function ruleNameCandidates(titles: SoftwareTitle[]): RuleNameCandidate[] {
  const candidates: RuleNameCandidate[] = [];
  const seen = new Set<string>();
  for (const title of titles) {
    for (const identity of title.signing_identities.items) {
      if (identity.signing_identifier) {
        addCandidate(candidates, seen, title.name, "signingid", identity.signing_identifier);
      }
      if (identity.team_identifier && identity.developer_name) {
        addCandidate(candidates, seen, identity.developer_name, "teamid", identity.team_identifier);
      }
    }
  }
  return candidates;
}

function addCandidate(
  candidates: RuleNameCandidate[],
  seen: Set<string>,
  name: string,
  ruleType: CandidateRuleType,
  identifier: string,
): void {
  const key = `${ruleType}:${identifier}`;
  if (seen.has(key)) return;
  seen.add(key);
  candidates.push({
    key,
    name,
    ruleType,
    identifier,
  });
}

function candidateName(candidate: RuleNameCandidate): string {
  return candidate.name;
}

function candidateValue(candidate: RuleNameCandidate): string {
  return candidate.key;
}

function renderCandidate(candidate: RuleNameCandidate): ReactNode {
  const metadata = RULE_TYPES[candidate.ruleType];
  const badgeClassName = "badgeClassName" in metadata ? metadata.badgeClassName : undefined;
  return (
    <>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate font-medium">{candidate.name}</span>
        <span className="truncate text-xs text-muted-foreground">{candidate.identifier}</span>
      </span>
      <Badge
        variant={metadata.variant ?? "secondary"}
        className={cn("shrink-0 self-center", badgeClassName)}
      >
        {metadata.name}
      </Badge>
    </>
  );
}
