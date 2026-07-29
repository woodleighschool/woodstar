import { getRouteApi } from "@tanstack/react-router";

import { RuleForm } from "./fields";
import { formFromSearch } from "./form-state";
import { useCreateSantaRule } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/rules/new");

export function RuleCreatePage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const create = useCreateSantaRule();

  return (
    <RuleForm
      initial={formFromSearch(search)}
      title="Create Rule"
      submitLabel="Create"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" ? "targets" : undefined,
          }),
        })
      }
      onCancel={() => void navigate({ to: "/santa/rules" })}
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({ to: "/santa/rules/$id", params: { id: String(id) } });
        }
      }}
    />
  );
}
