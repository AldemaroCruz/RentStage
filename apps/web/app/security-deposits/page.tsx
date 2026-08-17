"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { depositStatusLabel, formatCurrency, formatDateTime, formatReservationNumber, paymentMethodLabel } from "@/lib/format";
import type { PaymentMethod, ReservationSummary, SecurityDeposit, SecurityDepositStatus } from "@/lib/types";

export default function SecurityDepositsPage() {
  const { can } = useAuth();
  const canManage = can("payment.manage");
  const [items, setItems] = useState<SecurityDeposit[]>([]);
  const [reservations, setReservations] = useState<ReservationSummary[]>([]);
  const [status, setStatus] = useState<"" | SecurityDepositStatus>("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ reservation_id: "", amount: 0, currency: "USD", method: "BANK_TRANSFER" as PaymentMethod, reference: "", notes: "", mark_received: true });
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState("");

  const load = useCallback((selectedStatus = status) => {
    setLoading(true);
    Promise.all([
      api<{ items: SecurityDeposit[] }>(`/api/v1/security-deposits${selectedStatus ? `?status=${selectedStatus}` : ""}`),
      api<{ items: ReservationSummary[] }>("/api/v1/reservations"),
    ]).then(([deposits, reservationList]) => {
      setItems(deposits.items);
      setReservations(reservationList.items.filter((item) => item.status !== "CANCELLED"));
      setMessage("");
    }).catch((reason) => setMessage(reason instanceof Error ? reason.message : "No fue posible cargar los depósitos."))
      .finally(() => setLoading(false));
  }, [status]);

  useEffect(() => { load(""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const metrics = useMemo(() => items.reduce((result, item) => ({
    held: result.held + (item.status === "RECEIVED" || item.status === "PARTIALLY_SETTLED" ? item.balance_amount : 0),
    returned: result.returned + item.returned_amount,
    retained: result.retained + item.retained_amount,
  }), { held: 0, returned: 0, retained: 0 }), [items]);

  async function create(event: FormEvent) {
    event.preventDefault();
    setWorking("create");
    setMessage("");
    try {
      const created = await api<SecurityDeposit>("/api/v1/security-deposits", { method: "POST", body: JSON.stringify({ ...form, received_at: form.mark_received ? new Date().toISOString() : null }) });
      setMessage(`Depósito ${created.display_number} registrado.`);
      setOpen(false);
      setForm({ reservation_id: "", amount: 0, currency: "USD", method: "BANK_TRANSFER", reference: "", notes: "", mark_received: true });
      load();
    } catch (error) {
      if (error instanceof ApiError && error.fields) setMessage(Object.values(error.fields).join(" "));
      else setMessage(error instanceof Error ? error.message : "No fue posible crear el depósito.");
    } finally {
      setWorking("");
    }
  }

  async function receive(item: SecurityDeposit) {
    setWorking(item.id);
    try {
      await api<SecurityDeposit>(`/api/v1/security-deposits/${item.id}/receive`, { method: "POST", body: JSON.stringify({ received_at: new Date().toISOString(), method: item.method || "OTHER", reference: item.reference, notes: item.notes }) });
      load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : "No fue posible recibir el depósito.");
    } finally {
      setWorking("");
    }
  }

  async function settle(item: SecurityDeposit) {
    const returnedRaw = window.prompt(`Monto a devolver (máximo ${item.balance_amount.toFixed(2)}):`, item.balance_amount.toFixed(2));
    if (returnedRaw === null) return;
    const returned = Number(returnedRaw);
    const retained = Math.max(0, Number((item.balance_amount - returned).toFixed(2)));
    const reason = retained > 0 ? window.prompt("Motivo de retención:") || "" : "Devolución de garantía";
    if (!Number.isFinite(returned) || returned < 0 || returned > item.balance_amount || (retained > 0 && !reason.trim())) {
      setMessage("Los montos o el motivo de liquidación no son válidos.");
      return;
    }
    setWorking(item.id);
    try {
      await api<SecurityDeposit>(`/api/v1/security-deposits/${item.id}/settle`, { method: "POST", body: JSON.stringify({ returned_amount: item.returned_amount + returned, retained_amount: item.retained_amount + retained, settled_at: new Date().toISOString(), reason }) });
      load();
    } catch (reasonValue) {
      setMessage(reasonValue instanceof Error ? reasonValue.message : "No fue posible liquidar el depósito.");
    } finally {
      setWorking("");
    }
  }

  return <div className="page-stack deposits-page">
    <section className="page-heading"><div><p className="eyebrow">SECURITY DEPOSITS</p><h2>Depósitos de garantía</h2><p>Controla fondos recibidos, devoluciones y retenciones sin mezclarlos con ingreso por alquiler ni IVA.</p></div>{canManage && <button className="button button-primary" onClick={() => setOpen((value) => !value)}>+ Nuevo depósito</button>}</section>
    {message && <div className={message.includes("registrado") ? "success-banner" : "form-alert"}>{message}</div>}

    <section className="deposit-metric-grid"><article className="panel"><span>Garantías en custodia</span><strong>{formatCurrency(metrics.held)}</strong></article><article className="panel"><span>Devuelto</span><strong>{formatCurrency(metrics.returned)}</strong></article><article className="panel"><span>Retenido</span><strong>{formatCurrency(metrics.retained)}</strong></article><article className="panel"><span>Registros visibles</span><strong>{items.length}</strong></article></section>

    {open && <form className="panel deposit-create-panel" onSubmit={create}><div className="panel-header"><div><p className="eyebrow">NEW DEPOSIT</p><h2>Registrar garantía</h2></div><button type="button" className="icon-button" onClick={() => setOpen(false)}>×</button></div><div className="form-grid three-columns"><label className="field deposit-reservation-field"><span>Reserva *</span><select value={form.reservation_id} onChange={(event) => setForm({ ...form, reservation_id: event.target.value })}><option value="">Seleccionar…</option>{reservations.map((item) => <option key={item.id} value={item.id}>{formatReservationNumber(item.reservation_number)} · {item.customer_name}</option>)}</select></label><label className="field"><span>Monto</span><input type="number" min="0.01" step="0.01" value={form.amount} onChange={(event) => setForm({ ...form, amount: Number(event.target.value) })} /></label><label className="field"><span>Método</span><select value={form.method} onChange={(event) => setForm({ ...form, method: event.target.value as PaymentMethod })}><option value="BANK_TRANSFER">Transferencia</option><option value="CASH">Efectivo</option><option value="CARD">Tarjeta</option><option value="CHECK">Cheque</option><option value="OTHER">Otro</option></select></label><label className="field"><span>Referencia</span><input value={form.reference} onChange={(event) => setForm({ ...form, reference: event.target.value })} /></label><label className="field"><span>Notas</span><input value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} /></label><label className="toggle-row deposit-received-toggle"><input type="checkbox" checked={form.mark_received} onChange={(event) => setForm({ ...form, mark_received: event.target.checked })} /><span>Marcar como recibido ahora</span></label></div><div className="form-actions"><button className="button button-primary" disabled={working === "create"}>{working === "create" ? "Guardando…" : "Crear depósito"}</button></div></form>}

    <section className="panel deposit-list-panel"><div className="deposit-toolbar"><select value={status} onChange={(event) => { const value = event.target.value as "" | SecurityDepositStatus; setStatus(value); load(value); }}><option value="">Todos los estados</option><option value="PENDING">Pendientes</option><option value="RECEIVED">Recibidos</option><option value="PARTIALLY_SETTLED">Parciales</option><option value="RETURNED">Devueltos</option><option value="RETAINED">Retenidos</option><option value="SETTLED">Liquidados</option></select><span>{items.length} depósitos</span></div>{loading ? <div className="table-skeleton">Cargando depósitos…</div> : items.length === 0 ? <EmptyState icon="◇" title="Aún no hay depósitos" description="Asocia una garantía a una reserva activa." /> : <div className="data-table-wrap"><table className="data-table deposit-table"><thead><tr><th>Depósito</th><th>Reserva</th><th>Cliente</th><th>Estado</th><th>Recibido</th><th>Saldo</th><th>Método</th><th>Acciones</th></tr></thead><tbody>{items.map((item) => <tr key={item.id}><td><strong className="invoice-number-link">{item.display_number}</strong><small>{formatDateTime(item.created_at)}</small></td><td><Link href={`/reservations/${item.reservation_id}`}>{formatReservationNumber(item.reservation_number)}</Link></td><td><strong>{item.customer_name}</strong></td><td><span className={`billing-status-chip deposit-${item.status.toLowerCase()}`}>{depositStatusLabel(item.status)}</span></td><td>{formatCurrency(item.amount, item.currency)}<small>{item.received_at ? formatDateTime(item.received_at) : "No recibido"}</small></td><td><strong>{formatCurrency(item.balance_amount, item.currency)}</strong><small>{item.retained_amount > 0 ? `${formatCurrency(item.retained_amount)} retenido` : ""}</small></td><td>{paymentMethodLabel(item.method)}</td><td><div className="row-actions">{canManage && item.status === "PENDING" && <button className="button button-small" disabled={working === item.id} onClick={() => void receive(item)}>Recibir</button>}{canManage && (item.status === "RECEIVED" || item.status === "PARTIALLY_SETTLED") && item.balance_amount > 0 && <button className="button button-small" disabled={working === item.id} onClick={() => void settle(item)}>Liquidar</button>}</div></td></tr>)}</tbody></table></div>}</section>
    <section className="architecture-note"><span>i</span><div><strong>Separación contable</strong><p>Un depósito de garantía permanece fuera del total facturado hasta que exista una decisión de retención. v0.11 registra la custodia y liquidación sin asumir automáticamente que es una venta gravada.</p></div></section>
  </div>;
}
