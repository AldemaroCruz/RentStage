import { ReactNode } from "react";

export function StatCard({
  label,
  value,
  hint,
  icon,
  tone = "default",
}: {
  label: string;
  value: string | number;
  hint: string;
  icon: ReactNode;
  tone?: "default" | "positive" | "warning" | "purple";
}) {
  return (
    <article className={`stat-card stat-${tone}`}>
      <div className="stat-card-top">
        <span className="stat-icon">{icon}</span>
        <span className="stat-sparkline" aria-hidden="true">
          <i />
          <i />
          <i />
          <i />
          <i />
        </span>
      </div>
      <p>{label}</p>
      <strong>{value}</strong>
      <small>{hint}</small>
    </article>
  );
}
