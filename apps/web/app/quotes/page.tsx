"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { QuoteStatusBadge } from "@/components/QuoteStatusBadge";
import { api } from "@/lib/api";
import { formatCurrency, formatDateTime, formatQuoteNumber } from "@/lib/format";
import type { QuoteStatus, QuoteSummary } from "@/lib/types";

const statuses: Array<{ value: "" | QuoteStatus; label: string }> = [
  { value: "", label: "Todos los estados" },
  { value: "DRAFT", label: "Borradores" },
  { value: "SENT", label: "Enviadas" },
  { value: "ACCEPTED", label: "Aceptadas" },
  { value: "REJECTED", label: "Rechazadas" },
  { value: "EXPIRED", label: "Expiradas" },
  { value: "CANCELLED", label: "Canceladas" },
];

export default function QuotesPage() {
  const { can } = useAuth();
  const canManage = can("quote.manage");
  const [quotes, setQuotes] = useState<QuoteSummary[]>([]);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<"" | QuoteStatus>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setLoading(true);
      const params = new URLSearchParams();
      if (search.trim()) params.set("q", search.trim());
      if (status) params.set("status", status);
      api<{ items: QuoteSummary[] }>(`/api/v1/quotes?${params.toString()}`)
        .then((response) => {
          setQuotes(response.items);
          setError("");
        })
        .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar las cotizaciones."))
        .finally(() => setLoading(false));
    }, 220);
    return () => window.clearTimeout(timer);
  }, [search, status]);

  const metrics = useMemo(() => ({
    total: quotes.length,
    draft: quotes.filter((quote) => quote.status === "DRAFT").length,
    sent: quotes.filter((quote) => quote.status === "SENT").length,
    acceptedValue: quotes.filter((quote) => quote.status === "ACCEPTED").reduce((sum, quote) => sum + quote.total, 0),
  }), [quotes]);

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div>
          <p className="eyebrow">QUOTE CORE</p>
          <h2>Cotizaciones</h2>
          <p>Crea propuestas con precios históricos, recursos del inventario y transiciones comerciales auditables.</p>
        </div>
        {canManage && <Link className="button button-primary" href="/quotes/new"><span className="button-plus">+</span> Nueva cotización</Link>}
      </section>

      <section className="quote-metric-strip">
        <article><span>Visibles</span><strong>{metrics.total}</strong><small>Según filtros actuales</small></article>
        <article><span>Borradores</span><strong>{metrics.draft}</strong><small>Pendientes de envío</small></article>
        <article><span>Enviadas</span><strong>{metrics.sent}</strong><small>Esperando respuesta</small></article>
        <article><span>Valor aceptado</span><strong>{formatCurrency(metrics.acceptedValue)}</strong><small>Dentro del resultado filtrado</small></article>
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
          <select value={status} onChange={(event) => setStatus(event.target.value as "" | QuoteStatus)}>
            {statuses.map((option) => <option key={option.value || "all"} value={option.value}>{option.label}</option>)}
          </select>
          <span className="toolbar-count">{quotes.length} cotizaciones</span>
        </div>

        {loading ? (
          <div className="table-skeleton">Cargando cotizaciones…</div>
        ) : error ? (
          <div className="inline-error">{error}</div>
        ) : quotes.length === 0 ? (
          <EmptyState
            icon="▤"
            title="No encontramos cotizaciones"
            description={search || status ? "Prueba con otros filtros." : "Crea el primer borrador para comenzar el flujo comercial."}
            action={!search && !status && canManage ? <Link className="button button-primary" href="/quotes/new">Crear cotización</Link> : undefined}
          />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table quotes-table">
              <thead>
                <tr>
                  <th>Cotización</th>
                  <th>Cliente</th>
                  <th>Evento</th>
                  <th>Período</th>
                  <th>Estado</th>
                  <th>Items</th>
                  <th>Total</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {quotes.map((quote) => (
                  <tr key={quote.id}>
                    <td><strong className="mono-copy quote-number-copy">{formatQuoteNumber(quote.quote_number)}</strong><span className="table-subline">Creada {formatDateTime(quote.created_at)}</span></td>
                    <td><Link className="table-link" href={`/customers/${quote.customer_id}`}>{quote.customer_name}</Link><span className="table-subline">{quote.customer_phone || "Sin teléfono"}</span></td>
                    <td><strong className="table-primary-copy">{quote.event_type || "Sin tipo"}</strong><span className="table-subline">{quote.event_location || "Sin ubicación"}</span></td>
                    <td><span>{formatDateTime(quote.start_at)}</span><span className="table-subline">hasta {formatDateTime(quote.end_at)}</span></td>
                    <td><QuoteStatusBadge status={quote.status} /></td>
                    <td><strong className="table-primary-copy">{quote.item_count}</strong></td>
                    <td><strong className="table-primary-copy quote-total-copy">{formatCurrency(quote.total)}</strong></td>
                    <td><div className="row-actions"><Link className="icon-action" href={`/quotes/${quote.id}`} title="Ver cotización">→</Link></div></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="architecture-note">
        <span>i</span>
        <div><strong>Las cotizaciones no reservan equipo por sí solas</strong><p>Cuando una cotización aceptada se convierte, Booking Core vuelve a validar disponibilidad dentro de una transacción y crea la reserva que bloquea las cantidades.</p></div>
      </section>
    </div>
  );
}
