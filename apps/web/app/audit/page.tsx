"use client";

import { useEffect, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { AuditEvent } from "@/lib/types";

const actionLabels: Record<string, string> = {
  CATEGORY_CREATED: "Categoría creada",
  CATEGORY_DELETED: "Categoría eliminada",
  RESOURCE_CREATED: "Recurso creado",
  RESOURCE_UPDATED: "Recurso actualizado",
  ASSET_CREATED: "Unidad creada",
  ASSET_UPDATED: "Unidad actualizada",
  ASSET_RETIRED: "Unidad retirada",
  CUSTOMER_CREATED: "Cliente creado",
  CUSTOMER_UPDATED: "Cliente actualizado",
  PACKAGE_CREATED: "Paquete creado",
  PACKAGE_UPDATED: "Paquete actualizado",
  PACKAGE_ARCHIVED: "Paquete archivado",
  QUOTE_CREATED: "Cotización creada",
  QUOTE_UPDATED: "Cotización actualizada",
  QUOTE_SENT: "Cotización enviada",
  QUOTE_ACCEPTED: "Cotización aceptada",
  QUOTE_REJECTED: "Cotización rechazada",
  QUOTE_CANCELLED: "Cotización cancelada",
  TENANT_CREATED: "Workspace creado",
  TENANT_UPDATED: "Workspace actualizado",
  MEMBERSHIP_INVITATION_CREATED: "Invitación creada",
  MEMBERSHIP_INVITATION_ACCEPTED: "Invitación aceptada",
  MEMBERSHIP_INVITATION_REVOKED: "Invitación revocada",
  MEMBERSHIP_UPDATED: "Membresía actualizada",
};
function eventSubject(event: AuditEvent): string {
  if (typeof event.metadata?.name === "string") return event.metadata.name;
  if (typeof event.metadata?.asset_code === "string") return event.metadata.asset_code;
  if (typeof event.metadata?.quote_number === "number") {
    return `QT-${String(event.metadata.quote_number).padStart(6, "0")}`;
  }
  if (typeof event.metadata?.customer === "string") return event.metadata.customer;
  return event.entity_type;
}


export default function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ items: AuditEvent[] }>("/api/v1/audit?limit=100")
      .then((response) => setEvents(response.items))
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la auditoría."))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div><p className="eyebrow">WHO DID WHAT, AND WHEN</p><h2>Registro de auditoría</h2><p>Base de trazabilidad para acciones humanas, integraciones y el futuro agente de IA.</p></div>
        <span className="read-only-chip">Solo lectura</span>
      </section>

      <section className="panel audit-panel">
        {loading ? (
          <div className="table-skeleton">Cargando actividad…</div>
        ) : error ? (
          <div className="inline-error">{error}</div>
        ) : events.length === 0 ? (
          <EmptyState icon="↺" title="Aún no hay actividad" description="Las altas y modificaciones del inventario aparecerán aquí." />
        ) : (
          <div className="audit-timeline">
            {events.map((event) => {
              const name = eventSubject(event);
              return (
                <article className="audit-event" key={event.id}>
                  <span className="audit-marker" />
                  <div className="audit-event-card">
                    <div className="audit-event-main"><strong>{actionLabels[event.action] || event.action}</strong><p><b>{name}</b> · actor <span className="mono-copy">{event.actor_name || event.actor_email || event.actor_id}</span></p>{event.actor_email && <small>{event.actor_email}</small>}</div>
                    <div className="audit-event-meta"><span>{formatDateTime(event.created_at)}</span><small>{event.actor_type}</small></div>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
