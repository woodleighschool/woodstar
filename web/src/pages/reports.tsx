import { PendingBanner } from "@/components/feedback/pending-banner";
import { PageHeader } from "@/components/ui/page-header";

export function ReportsPage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Reports"
        description="Saved query outputs and rollups. Reports surface state without making automation decisions."
      />
      <div className="p-6">
        <PendingBanner endpoint="/api/v1/reports" />
      </div>
    </div>
  );
}
