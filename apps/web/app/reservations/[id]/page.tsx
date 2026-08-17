"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "@/components/Modal";
import { useAuth } from "@/components/AuthProvider";
import { ReservationStatusBadge } from "@/components/ReservationStatusBadge";
import { WarehousePanel } from "@/components/WarehousePanel";
import { ApiError, api } from "@/lib/api";
import {
  formatCurrency,
  formatDateTime,
  formatQuoteNumber,
  formatReservationNumber,
  reservationSourceLabel,
  reservationStatusLabel,
  returnConditionLabel,
  toLocalDateTimeInput,
  warehouseActivityLabel,
} from "@/lib/format";
import type { AvailabilityResult, ReservationDetail, ReservationStatus, ReturnCondition } from "@/lib/types";

type TransitionAction = "confirm" | "prepare" | "mark-ready" | "complete" | "cancel";

type PrimaryAction = {
  action: TransitionAction | "checkout-modal" | "return-modal";
  label: string;
  confirmation?: string;
};

function primaryAction(status: ReservationStatus): PrimaryAction | null {
  const actions: Partial<Record<ReservationStatus, PrimaryAction>> = {
    PENDING: { action: "confirm", label: "Confirmar reserva", confirmation: "¿Confirmar esta reserva? El inventario continuará bloqueado." },
    CONFIRMED: { action: "prepare", label: "Comenzar preparación", confirmation: "¿Mover esta reserva a preparación?" },
    PREPARING: { action: "mark-ready", label: "Marcar lista", confirmation: "¿Confirmar que el pedido está completamente preparado?" },
    READY: { action: "checkout-modal", label: "Registrar entrega" },
    CHECKED_OUT: { action: "return-modal", label: "Registrar devolución" },
    RETURNED: { action: "complete", label: "Completar reserva", confirmation: "¿Cerrar esta reserva como completada?" },
  };
  return actions[status] || null;
}

const cancellable = new Set<ReservationStatus>(["PENDING", "CONFIRMED", "PREPARING", "READY"]);
const reschedulable = new Set<ReservationStatus>(["PENDING", "CONFIRMED", "PREPARING", "READY"]);

function asISO(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

const returnConditions: Array<{ value: ReturnCondition; label: string; help: string }> = [
  { value: "GOOD", label: "Buen estado", help: "La unidad vuelve a disponibilidad." },
  { value: "MAINTENANCE_REQUIRED", label: "Requiere mantenimiento", help: "La unidad pasa a mantenimiento." },
  { value: "DAMAGED", label: "Dañada", help: "La unidad queda marcada como dañada." },
  { value: "LOST", label: "Perdida", help: "La unidad queda marcada como perdida." },
];

type ReturnRow = { condition: ReturnCondition; notes: string };

export default function ReservationDetailPage() {
  const { can } = useAuth();
  const canManageReservation = can("reservation.manage");
  const canOperateWarehouse = can("warehouse.operate");
  const canManageBilling = can("billing.manage");
  const params = useParams<{ id: string }>();
  const [reservation, setReservation] = useState<ReservationDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [acting, setActing] = useState("");
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const [checkoutNotes, setCheckoutNotes] = useState("");
  const [returnOpen, setReturnOpen] = useState(false);
  const [returnNotes, setReturnNotes] = useState("");
  const [returnRows, setReturnRows] = useState<Record<string, ReturnRow>>({});
  const [rescheduleOpen, setRescheduleOpen] = useState(false);
  const [rescheduleMessage, setRescheduleMessage] = useState("");
  const [rescheduleFields, setRescheduleFields] = useState<Record<string, string>>({});
  const [rescheduleAvailability, setRescheduleAvailability] = useState<AvailabilityResult | null>(null);
  const [rescheduleForm, setRescheduleForm] = useState({
    block_start_at: "",
    block_end_at: "",
    event_start_at: "",
    event_end_at: "",
    reason: "",
  });

  const load = useCallback(() => {
    setLoading(true);
    api<ReservationDetail>(`/api/v1/reservations/${params.id}`)
      .then((response) => {
        setReservation(response);
        setError("");
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la reserva."))
      .finally(() => setLoading(false));
  }, [params.id]);

  useEffect(() => load(), [load]);

  const activeAssignments = useMemo(() => {
    if (!reservation) return [];
    return reservation.items.flatMap((item) => item.assignments.filter((assignment) => !assignment.released_at));
  }, [reservation]);

  async function transition(action: TransitionAction, confirmation: string) {
    if (!window.confirm(confirmation)) return;
    setActing(action);
    setError("");
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${params.id}/${action}`, { method: "POST" });
      setReservation(updated);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible cambiar el estado de la reserva.");
    } finally {
      setActing("");
    }
  }

  function openReturn() {
    const initial: Record<string, ReturnRow> = {};
    for (const assignment of activeAssignments) {
      initial[assignment.asset_id] = { condition: "GOOD", notes: "" };
    }
    setReturnRows(initial);
    setReturnNotes("");
    setReturnOpen(true);
  }

  function openReschedule() {
    if (!reservation) return;
    setRescheduleForm({
      block_start_at: toLocalDateTimeInput(reservation.block_start_at),
      block_end_at: toLocalDateTimeInput(reservation.block_end_at),
      event_start_at: toLocalDateTimeInput(reservation.event_start_at),
      event_end_at: toLocalDateTimeInput(reservation.event_end_at),
      reason: "",
    });
    setRescheduleMessage("");
    setRescheduleFields({});
    setRescheduleAvailability(null);
    setRescheduleOpen(true);
  }

  async function submitReschedule() {
    setActing("reschedule");
    setRescheduleMessage("");
    setRescheduleFields({});
    setRescheduleAvailability(null);
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${params.id}/reschedule`, {
        method: "POST",
        body: JSON.stringify({
          block_start_at: asISO(rescheduleForm.block_start_at),
          block_end_at: asISO(rescheduleForm.block_end_at),
          event_start_at: asISO(rescheduleForm.event_start_at),
          event_end_at: asISO(rescheduleForm.event_end_at),
          reason: rescheduleForm.reason,
        }),
      });
      setReservation(updated);
      setRescheduleOpen(false);
    } catch (reason) {
      if (reason instanceof ApiError) {
        setRescheduleMessage(reason.message);
        setRescheduleFields(reason.fields || {});
        const conflict = reason.payload.availability;
        if (conflict && typeof conflict === "object") {
          setRescheduleAvailability(conflict as AvailabilityResult);
        }
        if (reason.code === "asset_schedule_conflict") {
          const assetCode = typeof reason.payload.asset_code === "string" ? reason.payload.asset_code : "una unidad asignada";
          const number = typeof reason.payload.conflict_reservation_number === "number"
            ? formatReservationNumber(reason.payload.conflict_reservation_number)
            : "otra reserva";
          setRescheduleMessage(`${assetCode} ya está comprometida en ${number} durante el nuevo período.`);
        }
      } else {
        setRescheduleMessage("No fue posible reprogramar la reserva.");
      }
    } finally {
      setActing("");
    }
  }

  async function submitCheckout() {
    setActing("checkout");
    setError("");
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${params.id}/checkout`, {
        method: "POST",
        body: JSON.stringify({ notes: checkoutNotes }),
      });
      setReservation(updated);
      setCheckoutOpen(false);
      setCheckoutNotes("");
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible registrar la entrega.");
    } finally {
      setActing("");
    }
  }

  async function submitReturn() {
    setActing("return");
    setError("");
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${params.id}/return`, {
        method: "POST",
        body: JSON.stringify({
          notes: returnNotes,
          assets: activeAssignments.map((assignment) => ({
            asset_id: assignment.asset_id,
            condition: returnRows[assignment.asset_id]?.condition || "GOOD",
            notes: returnRows[assignment.asset_id]?.notes || "",
          })),
        }),
      });
      setReservation(updated);
      setReturnOpen(false);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible registrar la devolución.");
    } finally {
      setActing("");
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error && !reservation) return <div className="panel inline-error">{error}<button onClick={load}>Reintentar</button></div>;
  if (!reservation) return <div className="panel inline-error">Reserva no encontrada.</div>;

  const next = primaryAction(reservation.status);
  const nextAllowed = Boolean(next) && (next?.action === "confirm" ? canManageReservation : canOperateWarehouse);
  const nextDisabled = next?.action === "mark-ready" && !reservation.warehouse_complete;

  function runPrimaryAction() {
    if (!next) return;
    if (next.action === "checkout-modal") {
      setCheckoutNotes("");
      setCheckoutOpen(true);
      return;
    }
    if (next.action === "return-modal") {
      openReturn();
      return;
    }
    void transition(next.action, next.confirmation || "¿Continuar?");
  }

  return (
    <div className="page-stack">
      <div className="breadcrumbs">
        <Link href="/reservations">Reservas</Link>
        <span>/</span>
        <span>{formatReservationNumber(reservation.reservation_number)}</span>
      </div>

      {error && <div className="form-alert quote-action-alert">{error}</div>}

      <section className="reservation-detail-hero panel">
        <div>
          <div className="quote-detail-title-row">
            <div>
              <p className="eyebrow">BOOKING DOCUMENT</p>
              <h2>{formatReservationNumber(reservation.reservation_number)}</h2>
            </div>
            <ReservationStatusBadge status={reservation.status} />
          </div>
          <p className="quote-detail-event">{reservation.event_type || "Reserva de alquiler"}</p>
          <div className="quote-detail-meta">
            <span><small>Cliente</small><Link href={`/customers/${reservation.customer_id}`}>{reservation.customer_name}</Link></span>
            <span><small>Ubicación</small><strong>{reservation.event_location || "Sin ubicación"}</strong></span>
            <span><small>Creada</small><strong>{formatDateTime(reservation.created_at)}</strong></span>
          </div>
        </div>
        <div className="quote-detail-actions">
          {canManageReservation && reschedulable.has(reservation.status) && (
            <button className="button button-secondary" disabled={Boolean(acting)} onClick={openReschedule}>
              Reprogramar
            </button>
          )}
          {next && nextAllowed && (
            <button
              className="button button-primary"
              disabled={Boolean(acting) || nextDisabled}
              onClick={runPrimaryAction}
              title={nextDisabled ? "Asigna todas las unidades físicas antes de continuar." : undefined}
            >
              {acting === next.action || (next.action === "checkout-modal" && acting === "checkout") || (next.action === "return-modal" && acting === "return")
                ? "Procesando…"
                : next.label}
            </button>
          )}
          {canManageBilling && reservation.status !== "CANCELLED" && (
            <Link className="button button-secondary" href={`/invoices/new?source_type=RESERVATION&source_id=${reservation.id}`}>
              Crear factura
            </Link>
          )}
          {canManageReservation && cancellable.has(reservation.status) && (
            <button
              className="button button-danger-ghost"
              disabled={Boolean(acting)}
              onClick={() => void transition("cancel", "¿Cancelar esta reserva y liberar las unidades asignadas?")}
            >
              {acting === "cancel" ? "Procesando…" : "Cancelar reserva"}
            </button>
          )}
        </div>
      </section>

      <WarehousePanel reservation={reservation} onChanged={setReservation} canOperate={canOperateWarehouse} />

      <div className="reservation-detail-grid">
        <section className="panel quote-document-panel">
          <div className="reservation-period-grid">
            <article>
              <p className="eyebrow">PERÍODO BLOQUEADO</p>
              <h3>{formatDateTime(reservation.block_start_at)}</h3>
              <span>hasta {formatDateTime(reservation.block_end_at)}</span>
              <small>Este intervalo participa en disponibilidad.</small>
            </article>
            <article>
              <p className="eyebrow">HORARIO DEL EVENTO</p>
              <h3>{formatDateTime(reservation.event_start_at)}</h3>
              <span>hasta {formatDateTime(reservation.event_end_at)}</span>
              <small>Se utiliza para la agenda y la operación del evento.</small>
            </article>
          </div>

          <div className="data-table-wrap quote-items-table-wrap">
            <table className="data-table quote-items-table">
              <thead><tr><th>Descripción</th><th>Cantidad</th><th>Asignadas</th><th>Precio</th><th>Descuento</th><th>Total</th></tr></thead>
              <tbody>
                {reservation.items.map((item) => (
                  <tr key={item.id}>
                    <td><strong className="table-primary-copy">{item.description}</strong><span className="table-subline">{item.resource_name}</span></td>
                    <td>{item.quantity}</td>
                    <td>{item.track_individual_assets ? `${item.assigned_quantity}/${item.quantity}` : "N/A"}</td>
                    <td>{formatCurrency(item.unit_price)}</td>
                    <td>{formatCurrency(item.discount_amount)}</td>
                    <td><strong className="table-primary-copy">{formatCurrency(item.line_total)}</strong></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="quote-document-footer">
            <div className="quote-notes-block">
              <small>Notas operativas</small>
              <p>{reservation.notes || "Sin notas adicionales."}</p>
              {reservation.checkout_notes && <><small>Notas de entrega</small><p>{reservation.checkout_notes}</p></>}
              {reservation.return_notes && <><small>Notas de devolución</small><p>{reservation.return_notes}</p></>}
            </div>
            <dl className="quote-total-list">
              <div><dt>Subtotal</dt><dd>{formatCurrency(reservation.subtotal)}</dd></div>
              <div><dt>Descuento</dt><dd>− {formatCurrency(reservation.discount_amount)}</dd></div>
              <div><dt>Cargos adicionales</dt><dd>{formatCurrency(reservation.extra_charges)}</dd></div>
              <div className="quote-total-final"><dt>Total</dt><dd>{formatCurrency(reservation.total)}</dd></div>
            </dl>
          </div>
        </section>

        <aside className="page-stack reservation-side-column">
          <section className="panel quote-side-card">
            <p className="eyebrow">ORIGEN COMERCIAL</p>
            {reservation.quote_id && reservation.quote_number ? (
              <>
                <h3>{formatQuoteNumber(reservation.quote_number)}</h3>
                <p>La reserva conserva los precios históricos y líneas aceptadas por el cliente.</p>
                <Link href={`/quotes/${reservation.quote_id}`} className="text-link">Ver cotización →</Link>
              </>
            ) : (
              <><h3>{reservationSourceLabel(reservation.source)}</h3><p>No está vinculada a una cotización. Fue registrada directamente desde el canal indicado.</p></>
            )}
          </section>

          <section className="panel quote-side-card">
            <p className="eyebrow">ESTADO OPERATIVO</p>
            <h3>{reservationStatusLabel(reservation.status)}</h3>
            <p>{reservation.status === "RETURNED" || reservation.status === "COMPLETED" || reservation.status === "CANCELLED"
              ? "Esta reserva ya no bloquea inventario temporal."
              : "Esta reserva mantiene comprometidas sus cantidades durante el período bloqueado."}</p>
            {reservation.checked_out_at && <p className="warehouse-side-timestamp"><strong>Entregada:</strong> {formatDateTime(reservation.checked_out_at)} · {reservation.checked_out_by}</p>}
            {reservation.returned_at && <p className="warehouse-side-timestamp"><strong>Devuelta:</strong> {formatDateTime(reservation.returned_at)} · {reservation.returned_by}</p>}
          </section>

          <section className="panel reservation-history-card">
            <p className="eyebrow">HISTORIAL DE ESTADOS</p>
            <div className="reservation-timeline">
              {reservation.status_history.map((event) => (
                <article key={event.id}>
                  <span className="timeline-dot" />
                  <div>
                    <strong>{reservationStatusLabel(event.to_status)}</strong>
                    <small>{formatDateTime(event.created_at)} · {event.actor_id}</small>
                    {event.note && <p>{event.note}</p>}
                  </div>
                </article>
              ))}
            </div>
          </section>

          {reservation.schedule_history.length > 0 && (
            <section className="panel reservation-history-card schedule-history-card">
              <p className="eyebrow">REPROGRAMACIONES</p>
              <div className="reservation-timeline schedule-timeline">
                {reservation.schedule_history.map((event) => (
                  <article key={event.id}>
                    <span className="timeline-dot schedule-timeline-dot" />
                    <div>
                      <strong>Nueva fecha: {formatDateTime(event.new_block_start_at)}</strong>
                      <small>{formatDateTime(event.created_at)} · {event.actor_id}</small>
                      <p>Anterior: {formatDateTime(event.previous_block_start_at)} → {formatDateTime(event.previous_block_end_at)}</p>
                      <p>Nuevo: {formatDateTime(event.new_block_start_at)} → {formatDateTime(event.new_block_end_at)}</p>
                      {event.reason && <p><b>Motivo:</b> {event.reason}</p>}
                    </div>
                  </article>
                ))}
              </div>
            </section>
          )}

          {reservation.activity_history.length > 0 && (
            <section className="panel reservation-history-card warehouse-history-card">
              <p className="eyebrow">ACTIVIDAD DE INVENTARIO</p>
              <div className="reservation-timeline warehouse-timeline">
                {reservation.activity_history.map((event) => (
                  <article key={event.id}>
                    <span className="timeline-dot warehouse-timeline-dot" />
                    <div>
                      <strong>{warehouseActivityLabel(event.event_type)}</strong>
                      <small>{formatDateTime(event.created_at)} · {event.actor_id}</small>
                      {event.asset_code && <p><b>{event.asset_code}</b>{event.resource_name ? ` · ${event.resource_name}` : ""}</p>}
                      {typeof event.metadata.condition === "string" && <p>{returnConditionLabel(event.metadata.condition)}</p>}
                      {event.note && <p>{event.note}</p>}
                    </div>
                  </article>
                ))}
              </div>
            </section>
          )}
        </aside>
      </div>

      <Modal open={rescheduleOpen} title="Reprogramar reserva" eyebrow={formatReservationNumber(reservation.reservation_number)} onClose={() => setRescheduleOpen(false)} width="780px">
        <div className="form-stack reschedule-modal">
          <div className="reschedule-intro">
            <strong>RentStage volverá a validar cantidades y unidades físicas asignadas.</strong>
            <p>La reserva actual se excluye del cálculo; cualquier conflicto con otras reservas impedirá el cambio.</p>
          </div>
          {rescheduleMessage && <div className="form-alert">{rescheduleMessage}</div>}
          <div className="schedule-editor-grid">
            <label className="field"><span>Inicio del bloqueo</span><input type="datetime-local" value={rescheduleForm.block_start_at} onChange={(event) => setRescheduleForm({ ...rescheduleForm, block_start_at: event.target.value })} />{rescheduleFields.block_start_at && <small className="field-error">{rescheduleFields.block_start_at}</small>}</label>
            <label className="field"><span>Fin del bloqueo</span><input type="datetime-local" value={rescheduleForm.block_end_at} onChange={(event) => setRescheduleForm({ ...rescheduleForm, block_end_at: event.target.value })} />{rescheduleFields.block_end_at && <small className="field-error">{rescheduleFields.block_end_at}</small>}</label>
            <label className="field"><span>Inicio del evento</span><input type="datetime-local" value={rescheduleForm.event_start_at} onChange={(event) => setRescheduleForm({ ...rescheduleForm, event_start_at: event.target.value })} />{rescheduleFields.event_start_at && <small className="field-error">{rescheduleFields.event_start_at}</small>}</label>
            <label className="field"><span>Fin del evento</span><input type="datetime-local" value={rescheduleForm.event_end_at} onChange={(event) => setRescheduleForm({ ...rescheduleForm, event_end_at: event.target.value })} />{rescheduleFields.event_end_at && <small className="field-error">{rescheduleFields.event_end_at}</small>}</label>
          </div>
          <label className="field"><span>Motivo del cambio</span><textarea value={rescheduleForm.reason} onChange={(event) => setRescheduleForm({ ...rescheduleForm, reason: event.target.value })} placeholder="Ej. El cliente cambió la fecha del evento." />{rescheduleFields.reason && <small className="field-error">{rescheduleFields.reason}</small>}</label>
          {rescheduleAvailability && (
            <div className="reschedule-conflicts">
              <strong>Disponibilidad del nuevo período</strong>
              {rescheduleAvailability.items.map((item) => <div key={item.resource_id}><span>{item.resource_name}</span><b className={item.can_fulfill ? "availability-ok" : "availability-fail"}>{item.available_quantity}/{item.requested_quantity}</b></div>)}
            </div>
          )}
          <div className="form-actions">
            <button className="button button-secondary" onClick={() => setRescheduleOpen(false)} disabled={Boolean(acting)}>Cancelar</button>
            <button className="button button-primary" onClick={() => void submitReschedule()} disabled={Boolean(acting)}>{acting === "reschedule" ? "Validando…" : "Confirmar nueva fecha"}</button>
          </div>
        </div>
      </Modal>

      <Modal open={checkoutOpen} title="Registrar entrega" eyebrow={formatReservationNumber(reservation.reservation_number)} onClose={() => setCheckoutOpen(false)} width="620px">
        <div className="form-stack">
          <div className="warehouse-modal-summary">
            <span>{activeAssignments.length}</span>
            <div><strong>unidades físicas listas para salir</strong><p>RentStage registrará la hora y el responsable del check-out.</p></div>
          </div>
          <div className="warehouse-modal-assets">
            {activeAssignments.map((assignment) => <span key={assignment.asset_id}>{assignment.asset_code}</span>)}
          </div>
          <label className="field">
            <span>Notas de entrega</span>
            <textarea value={checkoutNotes} onChange={(event) => setCheckoutNotes(event.target.value)} placeholder="Ej. Equipo entregado al cliente en almacén." />
          </label>
          <div className="form-actions">
            <button className="button button-secondary" onClick={() => setCheckoutOpen(false)} disabled={Boolean(acting)}>Cancelar</button>
            <button className="button button-primary" onClick={() => void submitCheckout()} disabled={Boolean(acting)}>{acting === "checkout" ? "Registrando…" : "Confirmar check-out"}</button>
          </div>
        </div>
      </Modal>

      <Modal open={returnOpen} title="Devolución e inspección" eyebrow={formatReservationNumber(reservation.reservation_number)} onClose={() => setReturnOpen(false)} width="860px">
        <div className="form-stack">
          <div className="return-intro"><strong>Inspecciona cada unidad antes de cerrar la devolución.</strong><p>La condición seleccionada actualizará automáticamente el estado físico del asset.</p></div>
          <div className="return-inspection-list">
            {activeAssignments.map((assignment) => {
              const row = returnRows[assignment.asset_id] || { condition: "GOOD" as ReturnCondition, notes: "" };
              return (
                <article key={assignment.asset_id} className={`return-inspection-card return-inspection-${row.condition.toLowerCase()}`}>
                  <header><div><strong>{assignment.asset_code}</strong><small>{assignment.serial_number || "Sin número de serie"}</small></div><span>{returnConditionLabel(row.condition)}</span></header>
                  <div className="form-grid two-columns return-inspection-fields">
                    <label className="field">
                      <span>Condición al regresar</span>
                      <select value={row.condition} onChange={(event) => setReturnRows((current) => ({ ...current, [assignment.asset_id]: { ...row, condition: event.target.value as ReturnCondition } }))}>
                        {returnConditions.map((condition) => <option value={condition.value} key={condition.value}>{condition.label}</option>)}
                      </select>
                      <small className="field-help">{returnConditions.find((condition) => condition.value === row.condition)?.help}</small>
                    </label>
                    <label className="field">
                      <span>Observaciones de la unidad</span>
                      <textarea value={row.notes} onChange={(event) => setReturnRows((current) => ({ ...current, [assignment.asset_id]: { ...row, notes: event.target.value } }))} placeholder="Detalle daños, mantenimiento o accesorios faltantes." />
                    </label>
                  </div>
                </article>
              );
            })}
          </div>
          <label className="field">
            <span>Notas generales de devolución</span>
            <textarea value={returnNotes} onChange={(event) => setReturnNotes(event.target.value)} placeholder="Resumen general de la recepción del pedido." />
          </label>
          <div className="form-actions">
            <button className="button button-secondary" onClick={() => setReturnOpen(false)} disabled={Boolean(acting)}>Cancelar</button>
            <button className="button button-primary" onClick={() => void submitReturn()} disabled={Boolean(acting)}>{acting === "return" ? "Procesando…" : "Confirmar devolución"}</button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
