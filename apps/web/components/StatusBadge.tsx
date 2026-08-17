import type { AssetStatus } from "@/lib/types";

const labels: Record<string, string> = {
  AVAILABLE: "Disponible",
  MAINTENANCE: "Mantenimiento",
  DAMAGED: "Dañado",
  LOST: "Perdido",
  RETIRED: "Retirado",
  ACTIVE: "Activo",
  INACTIVE: "Archivado",
};

export function StatusBadge({ status }: { status: AssetStatus | "ACTIVE" | "INACTIVE" }) {
  return (
    <span className={`status-badge status-${status.toLowerCase()}`}>
      <span className="status-dot" />
      {labels[status] || status}
    </span>
  );
}
