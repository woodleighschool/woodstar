import { PendingBanner } from "@/components/feedback/pending-banner";
import { PageHeader } from "@/components/ui/page-header";

export function ChecksPage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Checks"
        description="Boolean query-backed pass/fail evaluation. No automation in MVP."
      />
      <div className="p-6">
        <PendingBanner endpoint="/api/v1/checks" />
      </div>
    </div>
  );
}
