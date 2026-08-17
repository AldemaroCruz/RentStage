"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { useAuth } from "@/components/AuthProvider";
import type { DTEDocumentDetail, DTEStatus } from "@/lib/types";

function statusLabel(status: DTEStatus): string {
  return ({
    READY_TO_SIGN: "Preparado para firmar",
    SUBMITTING: "Transmitiendo",
    ACCEPTED: "Aceptado",
    REJECTED: "Rechazado",
    RETRY_REQUIRED: "Reintento requerido",
    INVALIDATION_PENDING: "Invalidación en curso",
    INVALIDATED: "Invalidado",
    CANCELLED: "Cancelado",
  } as Record<DTEStatus, string>)[status];
}

function statusTone(status: DTEStatus): string {
  if (status === "ACCEPTED") return "accepted";
  if (status === "REJECTED") return "rejected";
  if (status === "RETRY_REQUIRED") return "retry";
  if (status === "SUBMITTING" || status === "INVALIDATION_PENDING") return "processing";
  if (status === "INVALIDATED" || status === "CANCELLED") return "neutral";
  return "ready";
}

function pretty(value: Record<string, unknown>): string {
  return JSON.stringify(value || {}, null, 2);
}

export default function DTEDetailPage() {
  const params = useParams<{ id: string }>();
  const { can } = useAuth();
  const canManage = can("fiscal.manage");
  const [item, setItem] = useState<DTEDocumentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [reason, setReason] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<DTEDocumentDetail>(`/api/v1/dte/${params.id}`)
      .then(setItem)
      .catch((reasonValue) => setError(reasonValue instanceof Error ? reasonValue.message : "No fue posible cargar el DTE."))
      .finally(() => setLoading(false));
  }, [params.id]);

  useEffect(() => { load(); }, [load]);

  async function execute(name: "submit" | "retry" | "cancel") {
    if (!item || !canManage) return;
    setAction(name);
    setError("");
    setMessage("");
    try {
      const updated = await api<DTEDocumentDetail>(`/api/v1/dte/${item.id}/${name}`, { method: "POST", body: "{}" });
      setItem(updated);
      setMessage(updated.status === "ACCEPTED" ? "El proveedor aceptó el documento." : updated.status === "CANCELLED" ? "La preparación DTE fue cancelada y la factura volvió a quedar disponible." : "La operación terminó y el estado fue actualizado.");
    } catch (reasonValue) {
      setError(reasonValue instanceof Error ? reasonValue.message : "No fue posible completar la transmisión.");
    } finally {
      setAction("");
    }
  }

  async function invalidate(event: FormEvent) {
    event.preventDefault();
    if (!item || !canManage) return;
    setAction("invalidate");
    setError("");
    setMessage("");
    setFields({});
    try {
      const updated = await api<DTEDocumentDetail>(`/api/v1/dte/${item.id}/invalidate`, {
        method: "POST",
        body: JSON.stringify({ reason }),
      });
      setItem(updated);
      setMessage("El documento fue invalidado y la evidencia permanece disponible.");
      setReason("");
    } catch (reasonValue) {
      if (reasonValue instanceof ApiError) {
        setError(reasonValue.message);
        setFields(reasonValue.fields || {});
      } else {
        setError("No fue posible invalidar el documento.");
      }
    } finally {
      setAction("");
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error && !item) return <div className="panel inline-error">{error}<button type="button" onClick={load}>Reintentar</button></div>;
  if (!item) return null;

  return (
    <div className="page-stack dte-detail-page">
      <div className="breadcrumbs"><Link href="/dte">DTE</Link><span>/</span><span>{item.control_number}</span></div>

      <section className="panel dte-detail-hero">
        <div>
          <p className="eyebrow">{item.document_type_label.toUpperCase()} · {item.environment}</p>
          <h2>{item.control_number}</h2>
          <p>{item.invoice_display_number} · {item.customer_name}</p>
          <div className="dte-hero-meta"><span className={`dte-status-chip ${statusTone(item.status)}`}>{statusLabel(item.status)}</span><span>{item.provider_mode}</span><span>Esquema v{item.schema_version}</span></div>
        </div>
        <div className="dte-detail-actions">
          <Link className="button button-secondary" href={`/invoices/${item.invoice_id}`}>Ver factura</Link>
          {canManage && item.status === "READY_TO_SIGN" && <button className="button button-primary" disabled={Boolean(action)} onClick={() => void execute("submit")}>{action === "submit" ? "Transmitiendo…" : "Firmar y transmitir"}</button>}
          {canManage && item.status === "RETRY_REQUIRED" && <button className="button button-primary" disabled={Boolean(action)} onClick={() => void execute("retry")}>{action === "retry" ? "Reintentando…" : "Reintentar transmisión"}</button>}
          {canManage && (item.status === "READY_TO_SIGN" || item.status === "RETRY_REQUIRED") && <button className="button button-danger" disabled={Boolean(action)} onClick={() => { if (window.confirm("¿Cancelar esta preparación DTE? El número de control quedará en el historial y no se reutilizará.")) void execute("cancel"); }}>{action === "cancel" ? "Cancelando…" : "Cancelar preparación"}</button>}
        </div>
      </section>

      {error && <div className="form-alert">{error}</div>}
      {message && <div className="success-banner">{message}</div>}

      <section className="dte-detail-metrics">
        <article className="panel"><span>Código de generación</span><strong className="mono-copy">{item.generation_code}</strong></article>
        <article className="panel"><span>Sello de recepción</span><strong className="mono-copy">{item.receipt_seal || "Pendiente"}</strong></article>
        <article className="panel"><span>Intentos</span><strong>{item.attempt_count}</strong><small>{item.next_attempt_at ? `próximo ${formatDateTime(item.next_attempt_at)}` : "sin reintento programado"}</small></article>
        <article className="panel"><span>Proveedor</span><strong>{item.provider_status || item.provider_mode}</strong><small>{item.accepted_at ? formatDateTime(item.accepted_at) : formatDateTime(item.updated_at)}</small></article>
      </section>

      {(item.error_code || item.error_message) && <section className="panel dte-error-panel"><p className="eyebrow">PROVIDER RESPONSE</p><h3>{item.error_code || "Documento rechazado"}</h3><p>{item.error_message || "El proveedor devolvió un error sin descripción."}</p></section>}

      <div className="dte-detail-grid">
        <section className="panel dte-json-panel">
          <div className="panel-header"><div><p className="eyebrow">IMMUTABLE SNAPSHOT</p><h2>JSON preparado</h2><p>Contenido congelado al preparar el documento.</p></div></div>
          <pre>{pretty(item.payload)}</pre>
        </section>
        <section className="panel dte-provider-panel">
          <div className="panel-header"><div><p className="eyebrow">TRANSMISSION</p><h2>Intercambio con proveedor</h2></div></div>
          <dl className="profile-definition-list">
            <div><dt>Estado proveedor</dt><dd>{item.provider_status || "—"}</dd></div>
            <div><dt>Enviado</dt><dd>{formatDateTime(item.submitted_at)}</dd></div>
            <div><dt>Aceptado</dt><dd>{formatDateTime(item.accepted_at)}</dd></div>
            <div><dt>Rechazado</dt><dd>{formatDateTime(item.rejected_at)}</dd></div>
            <div><dt>Invalidado</dt><dd>{formatDateTime(item.invalidated_at)}</dd></div>
            <div><dt>Idempotencia</dt><dd className="mono-copy">{item.idempotency_key}</dd></div>
          </dl>
          <details className="dte-response-details"><summary>Solicitud registrada</summary><pre>{pretty(item.provider_request)}</pre></details>
          <details className="dte-response-details"><summary>Respuesta registrada</summary><pre>{pretty(item.provider_response)}</pre></details>{Object.keys(item.invalidation_request || {}).length > 0 && <details className="dte-response-details"><summary>Solicitud de invalidación</summary><pre>{pretty(item.invalidation_request)}</pre></details>}{Object.keys(item.invalidation_response || {}).length > 0 && <details className="dte-response-details"><summary>Respuesta de invalidación</summary><pre>{pretty(item.invalidation_response)}</pre></details>}
        </section>
      </div>

      {canManage && item.status === "ACCEPTED" && (
        <form className="panel dte-invalidation-panel" onSubmit={invalidate}>
          <div><p className="eyebrow">INVALIDATION</p><h2>Invalidar documento</h2><p>La invalidación es una operación fiscal independiente. El documento y sus eventos no se eliminan.</p></div>
          <label className="field"><span>Motivo</span><textarea rows={3} value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Describe el motivo de invalidación…" />{fields.reason && <small className="field-error">{fields.reason}</small>}</label>
          <button className="button button-danger" disabled={Boolean(action)}>{action === "invalidate" ? "Invalidando…" : "Invalidar DTE"}</button>
        </form>
      )}

      <section className="panel dte-event-panel">
        <div className="panel-header"><div><p className="eyebrow">AUDIT TRAIL</p><h2>Historial fiscal</h2></div></div>
        <div className="timeline-list">{item.events.map((event) => <article key={event.id}><span className="timeline-dot" /><div><strong>{event.event_type.replaceAll("_", " ")}</strong><small>{formatDateTime(event.created_at)}</small><pre>{JSON.stringify(event.metadata, null, 2)}</pre></div></article>)}</div>
      </section>

      <section className="architecture-note"><span>i</span><div><strong>{item.provider_mode === "MOCK" ? "Documento de certificación local" : "Adaptador técnico configurable"}</strong><p>{item.provider_mode === "MOCK" ? "Los sellos MOCK validan el ciclo de RentStage, pero no tienen validez fiscal ante Hacienda." : "La aceptación fiscal depende de las credenciales, esquemas y servicios oficiales habilitados para el contribuyente."}</p></div></section>
    </div>
  );
}
