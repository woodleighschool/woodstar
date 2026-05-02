import { PendingBanner } from "@/components/feedback/pending-banner";
import { PageHeader } from "@/components/ui/page-header";

export function SoftwarePage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Software"
        description="Observed software inventory across enrolled hosts."
      />
      <div className="p-6">
        <PendingBanner endpoint="/api/v1/software" />
      </div>
    </div>
  );
}
