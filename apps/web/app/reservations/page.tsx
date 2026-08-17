"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { ReservationStatusBadge } from "@/components/ReservationStatusBadge";
import { api } from "@/lib/api";
import { formatCurrency, formatDateTime, formatReservationNumber, reservationSourceLabel } from "@/lib/format";
import type { ReservationStatus, ReservationSummary } from "@/lib/types";

const statuses: Array<{ value: "" | ReservationStatus; label: string }> = [
  { value: "", label: "Todos los estados" },
  { value: "PENDING", label: "Pendientes" },
  { value: "CONFIRMED", label: "Confirmadas" },
  { value: "PREPARING", label: "Preparando" },
  { value: "READY", label: "Listas" },
  { value: "CHECKED_OUT", label: "Entregadas" },
  { value: "RETURNED", label: "Devueltas" },
  { value: "COMPLETED", label: "Completadas" },
  { value: "CANCELLED", label: "Canceladas" },
];

const activeStatuses = new Set<ReservationStatus>([
  "PENDING",
  "CONFIRMED",
  "PREPARING",
  "READY",
  "CHECKED_OUT",
]);

export default function ReservationsPage() {
  const { can } = useAuth();
  const canManage = can("reservation.manage");
  const [reservations, setReservations] = useState<ReservationSummary[]>([]);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<"" | ReservationStatus>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setLoading(true);
      const params = new URLSearchParams();
      if (search.trim()) params.set("q", search.trim());
      if (status) params.set("status", status);
      api<{ items: ReservationSummary[] }>(`/api/v1/reservations?${params.toString()}`)
        .then((response) => {
          setReservations(response.items);
          setError("");
        })
        .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar las reservas."))
        .finally(() => setLoading(false));
    }, 220);
    return () => window.clearTimeout(timer);
  }, [search, status]);

  const metrics = useMemo(() => ({
    total: reservations.length,
    active: reservations.filter((item) => activeStatuses.has(item.status)).length,
    checkedOut: reservations.filter((item) => item.status === "CHECKED_OUT").length,
    activeValue: reservations
      .filter((item) => activeStatuses.has(item.status))
      .reduce((sum, item) => sum + item.total, 0),
  }), [reservations]);

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div>
          <p className="eyebrow">BOOKING CORE</p>
          <h2>Reservas</h2>
          <p>Controla compromisos reales de inventario, períodos bloqueados y el avance operativo de cada alquiler.</p>
        </div>
        <div className="page-heading-actions">
          <Link className="button button-secondary" href="/quotes">Convertir cotización</Link>
          {canManage && <Link className="button button-primary" href="/reservations/new">+ Nueva reserva</Link>}
        </div>
      </section>

      <section className="quote-metric-strip reservation-metric-strip">
        <article><span>Visibles</span><strong>{metrics.total}</strong><small>Según filtros actuales</small></article>
        <article><span>Activas</span><strong>{metrics.active}</strong><small>Bloquean inventario</small></article>
        <article><span>Equipo afuera</span><strong>{metrics.checkedOut}</strong><small>Reservas entregadas</small></article>
        <article><span>Valor activo</span><strong>{formatCurrency(metrics.activeValue)}</strong><small>Compromisos no finalizados</small></article>
      </section>

      <section className="panel inventory-panel">
        <div className="inventory-toolbar">
          <label className="search-box">
            <span aria-hidden="true">⌕</span>
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Buscar por número, cliente, evento o ubicación"
            />
          </label>
          <select value={status} onChange={(event) => setStatus(event.target.value as "" | ReservationStatus)}>
            {statuses.map((option) => <option key={option.value || "all"} value={option.value}>{option.label}</option>)}
          </select>
          <span className="toolbar-count">{reservations.length} reservas</span>
        </div>

        {loading ? (
          <div className="table-skeleton">Cargando reservas…</div>
        ) : error ? (
          <div className="inline-error">{error}</div>
        ) : reservations.length === 0 ? (
          <EmptyState
            icon="▣"
            title="Aún no hay reservas"
            description={search || status ? "Prueba con otros filtros." : "Crea una reserva manual o convierte una cotización aceptada para bloquear inventario de forma segura."}
            action={!search && !status && canManage ? <Link className="button button-primary" href="/reservations/new">Crear reserva</Link> : undefined}
          />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table reservations-table">
              <thead>
                <tr>
                  <th>Reserva</th>
                  <th>Cliente</th>
                  <th>Evento</th>
                  <th>Período bloqueado</th>
                  <th>Estado</th>
                  <th>Items</th>
                  <th>Total</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {reservations.map((reservation) => (
                  <tr key={reservation.id}>
                    <td>
                      <strong className="mono-copy quote-number-copy">{formatReservationNumber(reservation.reservation_number)}</strong>
                      <span className="table-subline">
                        {reservation.quote_number
                          ? `Desde QT-${String(reservation.quote_number).padStart(6, "0")}`
                          : reservationSourceLabel(reservation.source)}
                      </span>
                    </td>
                    <td>
                      <Link className="table-link" href={`/customers/${reservation.customer_id}`}>{reservation.customer_name}</Link>
                      <span className="table-subline">{reservation.customer_phone || "Sin teléfono"}</span>
                    </td>
                    <td>
                      <strong className="table-primary-copy">{reservation.event_type || "Sin tipo"}</strong>
                      <span className="table-subline">{reservation.event_location || "Sin ubicación"}</span>
                    </td>
                    <td>
                      <span>{formatDateTime(reservation.block_start_at)}</span>
                      <span className="table-subline">hasta {formatDateTime(reservation.block_end_at)}</span>
                    </td>
                    <td><ReservationStatusBadge status={reservation.status} /></td>
                    <td><strong className="table-primary-copy">{reservation.item_count}</strong></td>
                    <td><strong className="table-primary-copy quote-total-copy">{formatCurrency(reservation.total)}</strong></td>
                    <td><div className="row-actions"><Link className="icon-action" href={`/reservations/${reservation.id}`} title="Ver reserva">→</Link></div></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="architecture-note">
        <span>i</span>
        <div>
          <strong>Las reservas activas sí bloquean inventario</strong>
          <p>Pendiente, confirmada, preparando, lista y entregada participan en el cálculo temporal de disponibilidad. Cancelada, devuelta y completada liberan las cantidades.</p>
        </div>
      </section>
    </div>
  );
}
