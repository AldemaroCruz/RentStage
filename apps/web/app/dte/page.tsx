"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { DTEDocumentSummary, DTESettings, DTEStatus } from "@/lib/types";

const statusOptions: Array<{ value: "" | DTEStatus; label: string }> = [
  { value: "", label: "Todos los estados" },
  { value: "READY_TO_SIGN", label: "Preparados" },
  { value: "SUBMITTING", label: "Transmitiendo" },
  { value: "ACCEPTED", label: "Aceptados" },
  { value: "REJECTED", label: "Rechazados" },
  { value: "RETRY_REQUIRED", label: "Reintento requerido" },
  { value: "INVALIDATION_PENDING", label: "Invalidando" },
  { value: "INVALIDATED", label: "Invalidados" },
  { value: "CANCELLED", label: "Cancelados" },
];

function statusLabel(status: DTEStatus): string {
  const labels: Record<DTEStatus, string> = {
    READY_TO_SIGN: "Preparado",
    SUBMITTING: "Transmitiendo",
    ACCEPTED: "Aceptado",
    REJECTED: "Rechazado",
    RETRY_REQUIRED: "Reintento",
    INVALIDATION_PENDING: "Invalidando",
    INVALIDATED: "Invalidado",
    CANCELLED: "Cancelado",
  };
  return labels[status];
}

function statusTone(status: DTEStatus): string {
  if (status === "ACCEPTED") return "accepted";
  if (status === "INVALIDATED" || status === "CANCELLED") return "neutral";
  if (status === "REJECTED") return "rejected";
  if (status === "RETRY_REQUIRED") return "retry";
  if (status === "SUBMITTING" || status === "INVALIDATION_PENDING") return "processing";
  return "ready";
}

export default function DTEPage() {
  const [items, setItems] = useState<DTEDocumentSummary[]>([]);
  const [settings, setSettings] = useState<DTESettings | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | DTEStatus>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback((search = query, selectedStatus = status) => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams();
    if (search.trim()) params.set("q", search.trim());
    if (selectedStatus) params.set("status", selectedStatus);
    Promise.all([
      api<{ items: DTEDocumentSummary[] }>(`/api/v1/dte${params.size ? `?${params.toString()}` : ""}`),
      api<DTESettings>("/api/v1/dte-settings"),
    ])
      .then(([documents, configuration]) => {
        setItems(documents.items);
        setSettings(configuration);
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar los documentos electrónicos."))
      .finally(() => setLoading(false));
  }, [query, status]);

  useEffect(() => { load("", ""); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const totals = useMemo(() => ({
    accepted: items.filter((item) => item.status === "ACCEPTED").length,
    pending: items.filter((item) => ["READY_TO_SIGN", "SUBMITTING", "RETRY_REQUIRED"].includes(item.status)).length,
    rejected: items.filter((item) => item.status === "REJECTED").length,
    invalidated: items.filter((item) => item.status === "INVALIDATED").length,
  }), [items]);

  function submit(event: FormEvent) {
    event.preventDefault();
    load();
  }

  function selectStatus(value: "" | DTEStatus) {
    setStatus(value);
    load(query, value);
  }

  return (
    <div className="page-stack dte-page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">EL SALVADOR · DTE</p>
          <h2>Documentos tributarios electrónicos</h2>
          <p>Prepara, transmite, consulta e invalida documentos fiscales sin mezclar su estado con cotizaciones, reservas o pagos.</p>
        </div>
        <Link className="button button-secondary" href="/settings/dte">Configurar integración</Link>
      </section>

      {settings && (
        <section className={`panel dte-provider-banner ${settings.provider_mode.toLowerCase()}`}>
          <div>
            <span className={`dte-live-dot ${settings.enabled ? "enabled" : ""}`} />
            <div>
              <strong>{settings.provider_mode === "MOCK" ? "Proveedor local MOCK" : "Adaptador MH_HTTP"}</strong>
              <p>{settings.provider_mode === "MOCK" ? "Ejecuta el ciclo completo en pruebas sin transmitir a Hacienda." : `${settings.environment} · ${settings.configuration_ready ? "configuración completa" : "configuración incompleta"}`}</p>
            </div>
          </div>
          <span className={`dte-environment-chip ${settings.environment.toLowerCase()}`}>{settings.environment}</span>
        </section>
      )}

      <section className="dte-summary-grid">
        <article className="panel"><span>Aceptados</span><strong>{totals.accepted}</strong><small>Con sello del proveedor</small></article>
        <article className="panel"><span>En proceso</span><strong>{totals.pending}</strong><small>Preparados o por reintentar</small></article>
        <article className="panel"><span>Rechazados</span><strong>{totals.rejected}</strong><small>Requieren corrección</small></article>
        <article className="panel"><span>Invalidados</span><strong>{totals.invalidated}</strong><small>Con historial conservado</small></article>
      </section>

      <section className="panel dte-list-panel">
        <form className="invoice-toolbar" onSubmit={submit}>
          <label className="search-box"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Buscar control, generación, factura o cliente…" /></label>
          <select value={status} onChange={(event) => selectStatus(event.target.value as "" | DTEStatus)}>{statusOptions.map((item) => <option key={item.value || "all"} value={item.value}>{item.label}</option>)}</select>
          <button className="button button-secondary" type="submit">Buscar</button>
          <span className="table-result-count">{items.length} documentos</span>
        </form>

        {loading ? <div className="table-skeleton">Cargando documentos electrónicos…</div> : error ? <div className="inline-error">{error}<button type="button" onClick={() => load()}>Reintentar</button></div> : items.length === 0 ? (
          <EmptyState icon="◇" title="Aún no hay DTE" description="Emite una factura con perfil fiscal completo y prepara su documento desde el detalle de factura." />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table dte-table">
              <thead><tr><th>Documento</th><th>Factura / cliente</th><th>Proveedor</th><th>Estado</th><th>Intentos</th><th>Última actividad</th><th /></tr></thead>
              <tbody>{items.map((item) => <tr key={item.id}>
                <td data-label="Documento"><Link className="invoice-number-link" href={`/dte/${item.id}`}>{item.control_number}</Link><small>{item.document_type_label} · v{item.schema_version}</small></td>
                <td data-label="Factura"><strong>{item.invoice_display_number}</strong><small>{item.customer_name}</small></td>
                <td data-label="Proveedor"><span className="category-pill">{item.provider_mode}</span><small>{item.environment}</small></td>
                <td data-label="Estado"><span className={`dte-status-chip ${statusTone(item.status)}`}>{statusLabel(item.status)}</span>{item.receipt_seal && <small className="mono-copy dte-seal-preview">{item.receipt_seal}</small>}</td>
                <td data-label="Intentos"><strong>{item.attempt_count}</strong>{item.next_attempt_at && <small>próximo {formatDateTime(item.next_attempt_at)}</small>}</td>
                <td data-label="Actividad"><strong>{formatDateTime(item.updated_at)}</strong>{item.provider_status && <small>{item.provider_status}</small>}</td>
                <td><Link className="icon-action" href={`/dte/${item.id}`}>→</Link></td>
              </tr>)}</tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
