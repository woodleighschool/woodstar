import { getRouteApi } from "@tanstack/react-router";

import { RuleForm } from "./fields";
import { formFromSearch } from "./form-state";
import { useCreateSantaRule } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/$id/rules/new");

export function RuleCreatePage() {
  const navigate = routeApi.useNavigate();
  const { id: configurationID } = routeApi.useParams();
  const search = routeApi.useSearch();
  const create = useCreateSantaRule();

  return (
    <RuleForm
      initial={formFromSearch(search, configurationID)}
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
      onCancel={() =>
        void navigate({
          to: "/santa/configurations/$id/rules",
          params: { id: configurationID },
        })
      }
      onSubmit={async (body) =>
        (await create.mutateAsync({ ...body, configuration_id: configurationID })).id
      }
      onSuccess={(ruleID) => {
        if (ruleID !== undefined) {
          void navigate({
            to: "/santa/configurations/$id/rules/$ruleId",
            params: { id: configurationID, ruleId: ruleID },
          });
        }
      }}
    />
  );
}
