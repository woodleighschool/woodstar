import { PendingBanner } from "@/components/feedback/pending-banner";
import { PageHeader } from "@/components/ui/page-header";

export function LabelsPage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Labels"
        description="Built-in, manual, and dynamic labels used to scope reports, checks, and module rules."
      />
      <div className="p-6">
        <PendingBanner endpoint="/api/v1/labels" />
      </div>
    </div>
  );
}
