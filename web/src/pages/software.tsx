import { PageHeader } from "@/components/ui/page-header";

export function SoftwarePage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Software"
        description="Observed software inventory across enrolled hosts."
      />
    </div>
  );
}
