import { quoteStatusLabel } from "@/lib/format";
import type { QuoteStatus } from "@/lib/types";

export function QuoteStatusBadge({ status }: { status: QuoteStatus }) {
  return (
    <span className={`status-badge quote-status quote-status-${status.toLowerCase()}`}>
      <span className="status-dot" />
      {quoteStatusLabel(status)}
    </span>
  );
}
