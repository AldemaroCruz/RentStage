"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { api } from "@/lib/api";
import { formatCurrency, formatDate, invoiceStatusLabel, invoiceStatusTone } from "@/lib/format";
import type { InvoiceDisplayStatus, InvoiceSummary } from "@/lib/types";

const statuses: Array<{ value: "" | InvoiceDisplayStatus; label: string }> = [
  { value: "", label: "Todos los estados" },
  { value: "DRAFT", label: "Borradores" },
  { value: "ISSUED", label: "Emitidas" },
  { value: "PARTIALLY_PAID", label: "Pago parcial" },
  { value: "OVERDUE", label: "Vencidas" },
  { value: "PAID", label: "Pagadas" },
  { value: "VOID", label: "Anuladas" },
];

export default function InvoicesPage() {
  const { can } = useAuth();
  const canManage = can("billing.manage");
  const [items, setItems] = useState<InvoiceSummary[]>([]);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | InvoiceDisplayStatus>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback((search = query, selectedStatus = status) => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams();
    if (search.trim()) params.set("q", search.trim());
    if (selectedStatus) params.set("status", selectedStatus);
    api<{ items: InvoiceSummary[] }>(`/api/v1/invoices${params.size ? `?${params.toString()}` : ""}`)
      .then((result) => setItems(result.items))
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar las facturas."))
      .finally(() => setLoading(false));
  }, [query, status]);

  useEffect(() => { load("", ""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const totals = useMemo(() => items.reduce((result, item) => ({
    total: result.total + (item.status === "VOID" || item.status === "DRAFT" ? 0 : item.total_amount),
    paid: result.paid + item.paid_amount,
    balance: result.balance + (item.status === "VOID" ? 0 : item.balance_due),
  }), { total: 0, paid: 0, balance: 0 }), [items]);

  function submit(event: FormEvent) {
    event.preventDefault();
    load();
  }

  function selectStatus(value: "" | InvoiceDisplayStatus) {
    setStatus(value);
    load(query, value);
  }

  return (
    <div className="page-stack invoices-page">
      <section className="page-heading">
        <div><p className="eyebrow">ACCOUNTS RECEIVABLE</p><h2>Facturas</h2><p>Convierte ventas aceptadas en facturas internas, controla saldos y prepara o consulta sus documentos DTE.</p></div>
        {canManage && <Link href="/invoices/new" className="button button-primary"><span className="button-plus">+</span> Nueva factura</Link>}
      </section>

      <section className="invoice-summary-grid">
        <article className="panel"><span>Facturado visible</span><strong>{formatCurrency(totals.total)}</strong></article>
        <article className="panel"><span>Cobrado</span><strong>{formatCurrency(totals.paid)}</strong></article>
        <article className="panel"><span>Saldo visible</span><strong>{formatCurrency(totals.balance)}</strong></article>
        <article className="panel"><span>Documentos</span><strong>{items.length}</strong></article>
      </section>

      <section className="panel invoice-list-panel">
        <form className="invoice-toolbar" onSubmit={submit}>
          <label className="search-box"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Buscar número, cliente, cotización o reserva…" /></label>
          <select value={status} onChange={(event) => selectStatus(event.target.value as "" | InvoiceDisplayStatus)}>{statuses.map((item) => <option key={item.value || "all"} value={item.value}>{item.label}</option>)}</select>
          <button className="button button-secondary" type="submit">Buscar</button>
          <span className="table-result-count">{items.length} resultados</span>
        </form>

        {loading ? <div className="table-skeleton">Cargando facturas…</div> : error ? <div className="inline-error">{error}<button type="button" onClick={() => load()}>Reintentar</button></div> : items.length === 0 ? <EmptyState icon="▤" title="Aún no hay facturas" description="Crea una factura manual o desde una cotización aceptada o reserva." action={canManage ? <Link className="button button-primary" href="/invoices/new">Crear factura</Link> : undefined} /> : (
          <div className="data-table-wrap">
            <table className="data-table invoice-table">
              <thead><tr><th>Documento</th><th>Cliente</th><th>Origen</th><th>Emisión / vence</th><th>Estado</th><th>Total</th><th>Saldo</th><th /></tr></thead>
              <tbody>{items.map((item) => <tr key={item.id}>
                <td data-label="Documento"><Link className="invoice-number-link" href={`/invoices/${item.id}`}>{item.display_number}</Link><small>{item.item_count} línea{item.item_count === 1 ? "" : "s"}</small></td>
                <td data-label="Cliente"><strong>{item.customer_name}</strong></td>
                <td data-label="Origen"><span className="category-pill">{item.source_type === "QUOTE" ? `Cotización ${item.quote_number || ""}` : item.source_type === "RESERVATION" ? `Reserva ${item.reservation_number || ""}` : "Manual"}</span></td>
                <td data-label="Fechas"><strong>{formatDate(item.issue_date)}</strong><small>vence {formatDate(item.due_date)}</small></td>
                <td data-label="Estado"><span className={`billing-status-chip ${invoiceStatusTone(item.display_status)}`}>{invoiceStatusLabel(item.display_status)}</span></td>
                <td data-label="Total"><strong>{formatCurrency(item.total_amount, item.currency)}</strong></td>
                <td data-label="Saldo"><strong className={item.display_status === "OVERDUE" ? "text-danger" : ""}>{formatCurrency(item.balance_due, item.currency)}</strong><small>{item.paid_amount > 0 ? `${formatCurrency(item.paid_amount, item.currency)} pagado` : "sin pagos"}</small></td>
                <td><Link className="icon-action" href={`/invoices/${item.id}`}>→</Link></td>
              </tr>)}</tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
