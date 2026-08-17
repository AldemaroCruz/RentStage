"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { api } from "@/lib/api";
import { formatCurrency, formatDateTime, quoteRequestStatusLabel, quoteRequestStatusTone } from "@/lib/format";
import type { QuoteRequestList, QuoteRequestStatus } from "@/lib/types";

const statuses: Array<{ value: "" | QuoteRequestStatus; label: string }> = [
  { value: "", label: "Todos los estados" },
  { value: "NEW", label: "Nuevas" },
  { value: "IN_REVIEW", label: "En revisión" },
  { value: "CONVERTED", label: "Convertidas" },
  { value: "CLOSED", label: "Cerradas" },
  { value: "SPAM", label: "Spam" },
];

export default function QuoteRequestsPage() {
  const { me } = useAuth();
  const fallbackCurrency = me?.active_workspace?.currency || "USD";
  const [result, setResult] = useState<QuoteRequestList>({ items: [], counts: {} });
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | QuoteRequestStatus>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback((search = query, selectedStatus = status) => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams();
    if (search.trim()) params.set("q", search.trim());
    if (selectedStatus) params.set("status", selectedStatus);
    api<QuoteRequestList>(`/api/v1/quote-requests${params.size ? `?${params.toString()}` : ""}`)
      .then(setResult)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar las solicitudes."))
      .finally(() => setLoading(false));
  }, [query, status]);

  useEffect(() => { load("", ""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const activeCount = useMemo(() => (result.counts.NEW || 0) + (result.counts.IN_REVIEW || 0), [result.counts]);
  const totalCount = useMemo(() => Object.values(result.counts).reduce((total, value) => total + (value || 0), 0), [result.counts]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    load();
  }

  function changeStatus(value: "" | QuoteRequestStatus) {
    setStatus(value);
    load(query, value);
  }

  return (
    <div className="page-stack quote-request-page">
      <div className="page-heading quote-request-heading">
        <div><p className="eyebrow">PUBLIC LEADS</p><h2>Solicitudes web</h2><p>Revisa las solicitudes enviadas desde el catálogo público y conviértelas en cotizaciones editables.</p></div>
        <Link className="button button-secondary" href="/settings/public-catalog">Configurar catálogo</Link>
      </div>

      <section className="quote-request-stat-grid">
        <div className="panel"><small>NUEVAS</small><strong>{result.counts.NEW || 0}</strong><span>pendientes de primera revisión</span></div>
        <div className="panel"><small>EN REVISIÓN</small><strong>{result.counts.IN_REVIEW || 0}</strong><span>en seguimiento comercial</span></div>
        <div className="panel"><small>CONVERTIDAS</small><strong>{result.counts.CONVERTED || 0}</strong><span>ya generaron cotización</span></div>
        <div className="panel"><small>ACTIVAS</small><strong>{activeCount}</strong><span>de {totalCount} solicitudes totales</span></div>
      </section>

      <section className="panel quote-request-list-panel">
        <form className="quote-request-toolbar" onSubmit={submitSearch}>
          <label className="search-box"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Buscar referencia, cliente, correo, teléfono o ubicación…" /></label>
          <select value={status} onChange={(event) => changeStatus(event.target.value as "" | QuoteRequestStatus)}>{statuses.map((item) => <option key={item.value || "all"} value={item.value}>{item.label}</option>)}</select>
          <button className="button button-secondary" type="submit">Buscar</button>
          <span className="table-result-count">{result.items.length} resultados</span>
        </form>

        {loading ? (
          <div className="table-skeleton">Cargando solicitudes…</div>
        ) : error ? (
          <div className="inline-error">{error}<button type="button" onClick={() => load()}>Reintentar</button></div>
        ) : result.items.length === 0 ? (
          <EmptyState icon="✦" title={totalCount ? "No encontramos coincidencias" : "Aún no hay solicitudes"} description={totalCount ? "Prueba otro término o estado." : "Cuando un cliente complete el formulario público, aparecerá aquí."} action={!totalCount ? <Link className="button button-primary" href="/settings/public-catalog">Abrir configuración</Link> : undefined} />
        ) : (
          <div className="quote-request-table-wrap">
            <table className="data-table quote-request-table">
              <thead><tr><th>Solicitud</th><th>Cliente</th><th>Evento</th><th>Período</th><th>Estado</th><th>Estimado</th><th /></tr></thead>
              <tbody>
                {result.items.map((item) => (
                  <tr key={item.id}>
                    <td data-label="Solicitud"><Link className="quote-request-reference" href={`/quote-requests/${item.id}`}>{item.reference_code}</Link><small>{formatDateTime(item.created_at)} · {item.package_count} paquete{item.package_count === 1 ? "" : "s"}</small></td>
                    <td data-label="Cliente"><strong>{item.customer_name}</strong><small>{item.email || item.phone || "Sin contacto"}</small></td>
                    <td data-label="Evento"><strong>{item.event_type || "Evento sin tipo"}</strong><small>{item.event_location || "Ubicación por confirmar"}</small></td>
                    <td data-label="Período"><strong>{formatDateTime(item.start_at)}</strong><small>hasta {formatDateTime(item.end_at)}</small></td>
                    <td data-label="Estado"><span className={`quote-request-status ${quoteRequestStatusTone(item.status)}`}>{quoteRequestStatusLabel(item.status)}</span>{!item.availability_available && <small className="availability-warning">Disponibilidad por revisar</small>}</td>
                    <td data-label="Estimado"><strong>{formatCurrency(item.estimated_total, item.currency || fallbackCurrency)}</strong></td>
                    <td className="table-actions"><Link href={`/quote-requests/${item.id}`} aria-label={`Abrir ${item.reference_code}`}>→</Link></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
