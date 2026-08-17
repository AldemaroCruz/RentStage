"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { formatCurrency, formatDateTime, paymentMethodLabel } from "@/lib/format";
import type { InvoiceSummary, PaymentDetail, PaymentMethod, PaymentSummary } from "@/lib/types";

export default function PaymentsPage() {
  const { can } = useAuth();
  const canManage = can("payment.manage");
  const [items, setItems] = useState<PaymentSummary[]>([]);
  const [invoices, setInvoices] = useState<InvoiceSummary[]>([]);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [open, setOpen] = useState(false);
  const [invoiceID, setInvoiceID] = useState("");
  const [amount, setAmount] = useState(0);
  const [method, setMethod] = useState<PaymentMethod>("BANK_TRANSFER");
  const [reference, setReference] = useState("");
  const [notes, setNotes] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback((selectedStatus = status, selectedQuery = query) => {
    setLoading(true);
    const params = new URLSearchParams();
    if (selectedQuery.trim()) params.set("q", selectedQuery.trim());
    if (selectedStatus) params.set("status", selectedStatus);
    Promise.all([
      api<{ items: PaymentSummary[] }>(`/api/v1/payments${params.size ? `?${params}` : ""}`),
      api<{ items: InvoiceSummary[] }>("/api/v1/invoices"),
    ]).then(([paymentResult, invoiceResult]) => {
      setItems(paymentResult.items);
      setInvoices(invoiceResult.items.filter((item) => (item.status === "ISSUED" || item.status === "PARTIALLY_PAID") && item.balance_due > 0));
      setMessage("");
    }).catch((reason) => setMessage(reason instanceof Error ? reason.message : "No fue posible cargar los pagos."))
      .finally(() => setLoading(false));
  }, [query, status]);

  useEffect(() => { load("", ""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const selectedInvoice = useMemo(() => invoices.find((item) => item.id === invoiceID), [invoices, invoiceID]);

  function chooseInvoice(value: string) {
    setInvoiceID(value);
    const selected = invoices.find((item) => item.id === value);
    if (selected) setAmount(selected.balance_due);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!selectedInvoice) return;
    setSaving(true);
    setMessage("");
    try {
      const created = await api<PaymentDetail>("/api/v1/payments", {
        method: "POST",
        body: JSON.stringify({
          customer_id: selectedInvoice.customer_id,
          amount: Number(amount),
          currency: selectedInvoice.currency,
          method,
          reference,
          notes,
          received_at: new Date().toISOString(),
          allocations: [{ invoice_id: selectedInvoice.id, amount: Number(amount) }],
        }),
      });
      setOpen(false);
      setInvoiceID("");
      setReference("");
      setNotes("");
      setMessage(`Pago ${created.display_number} registrado.`);
      load();
    } catch (error) {
      if (error instanceof ApiError && error.fields) setMessage(Object.values(error.fields).join(" "));
      else setMessage(error instanceof Error ? error.message : "No fue posible registrar el pago.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page-stack payments-page">
      <section className="page-heading"><div><p className="eyebrow">CASH MANAGEMENT</p><h2>Pagos</h2><p>Registra cobros parciales o totales y aplica cada movimiento al saldo de una factura emitida.</p></div>{canManage && <button className="button button-primary" onClick={() => setOpen((value) => !value)}>+ Registrar pago</button>}</section>
      {message && <div className={message.includes("registrado") ? "success-banner" : "form-alert"}>{message}</div>}

      {open && <form className="panel payment-create-panel" onSubmit={submit}>
        <div className="panel-header"><div><p className="eyebrow">NEW PAYMENT</p><h2>Registrar cobro</h2></div><button type="button" className="icon-button" onClick={() => setOpen(false)}>×</button></div>
        <div className="form-grid three-columns">
          <label className="field payment-invoice-field"><span>Factura abierta *</span><select value={invoiceID} onChange={(event) => chooseInvoice(event.target.value)}><option value="">Seleccionar factura…</option>{invoices.map((item) => <option key={item.id} value={item.id}>{item.display_number} · {item.customer_name} · saldo {formatCurrency(item.balance_due, item.currency)}</option>)}</select></label>
          <label className="field"><span>Monto</span><input type="number" min="0.01" max={selectedInvoice?.balance_due || undefined} step="0.01" value={amount} onChange={(event) => setAmount(Number(event.target.value))} /></label>
          <label className="field"><span>Método</span><select value={method} onChange={(event) => setMethod(event.target.value as PaymentMethod)}><option value="BANK_TRANSFER">Transferencia</option><option value="CASH">Efectivo</option><option value="CARD">Tarjeta</option><option value="CHECK">Cheque</option><option value="OTHER">Otro</option></select></label>
          <label className="field"><span>Referencia</span><input value={reference} onChange={(event) => setReference(event.target.value)} /></label>
          <label className="field payment-notes-field"><span>Notas</span><input value={notes} onChange={(event) => setNotes(event.target.value)} /></label>
        </div>
        <div className="form-actions"><button className="button button-primary" disabled={!selectedInvoice || saving}>{saving ? "Registrando…" : "Registrar pago"}</button></div>
      </form>}

      <section className="panel payment-list-panel">
        <div className="payment-toolbar"><label className="search-box"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Buscar pago, cliente o referencia…" /></label><select value={status} onChange={(event) => { const value = event.target.value; setStatus(value); load(value, query); }}><option value="">Todos los estados</option><option value="CONFIRMED">Confirmados</option><option value="VOIDED">Revertidos</option></select><button className="button button-secondary" onClick={() => load()}>Buscar</button><span>{items.length} pagos</span></div>
        {loading ? <div className="table-skeleton">Cargando pagos…</div> : items.length === 0 ? <EmptyState icon="$" title="Aún no hay pagos" description="Emite una factura y registra su primer cobro." /> : <div className="data-table-wrap"><table className="data-table payment-table"><thead><tr><th>Pago</th><th>Cliente</th><th>Fecha</th><th>Método</th><th>Referencia</th><th>Estado</th><th>Monto</th><th /></tr></thead><tbody>{items.map((item) => <tr key={item.id}><td><Link className="invoice-number-link" href={`/payments/${item.id}`}>{item.display_number}</Link><small>{item.allocation_count} asignación{item.allocation_count === 1 ? "" : "es"}</small></td><td><strong>{item.customer_name}</strong></td><td>{formatDateTime(item.received_at)}</td><td>{paymentMethodLabel(item.method)}</td><td>{item.reference || "—"}</td><td><span className={`billing-status-chip ${item.status === "CONFIRMED" ? "paid" : "void"}`}>{item.status === "CONFIRMED" ? "Confirmado" : "Revertido"}</span></td><td><strong>{formatCurrency(item.amount, item.currency)}</strong></td><td><Link className="icon-action" href={`/payments/${item.id}`}>→</Link></td></tr>)}</tbody></table></div>}
      </section>
    </div>
  );
}
