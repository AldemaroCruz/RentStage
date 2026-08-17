import { reservationStatusLabel } from "@/lib/format";
import type { ReservationStatus } from "@/lib/types";

export function ReservationStatusBadge({ status }: { status: ReservationStatus }) {
  return (
    <span className={`status-badge reservation-status reservation-status-${status.toLowerCase().replace("_", "-")}`}>
      <span className="status-dot" />
      {reservationStatusLabel(status)}
    </span>
  );
}
