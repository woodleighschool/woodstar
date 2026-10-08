import { useBreadcrumbs, type ResourceName } from "@components/layout/app-breadcrumbs";
import { runtime } from "@lib/runtime";

export function DocumentTitle({ title }: { title: string }) {
  return <title>{`${title} | ${runtime.name}`}</title>;
}

// Titles the tab from the route's crumbs: the open resource's name and any
// action under it, or the page's own label when no resource is open.
export function RouteTitle() {
  const crumbs = useBreadcrumbs();

  let resource: { Name: ResourceName; id: number } | undefined;
  let actions: string[] = [];
  for (const crumb of crumbs) {
    if ("id" in crumb) {
      resource = { Name: crumb.label, id: crumb.id };
      actions = [];
    } else {
      actions.push(crumb.label);
    }
  }

  if (!resource) {
    const title = actions.at(-1);
    return title ? <DocumentTitle title={title} /> : null;
  }

  const { Name, id } = resource;
  return (
    <Name id={id}>
      {(name) => (name ? <DocumentTitle title={[name, ...actions].join(" - ")} /> : null)}
    </Name>
  );
}
