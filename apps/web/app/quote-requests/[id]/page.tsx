"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import {
  formatCurrency,
  formatDateTime,
  formatQuoteNumber,
  quoteRequestStatusLabel,
  quoteRequestStatusTone,
} from "@/lib/format";
import type { QuoteRequestConversion, QuoteRequestDetail, QuoteRequestStatus } from "@/lib/types";

const editableStatuses: Array<{ value: Exclude<QuoteRequestStatus, "CONVERTED">; label: string }> = [
  { value: "NEW", label: "Nueva" },
  { value: "IN_REVIEW", label: "En revisión" },
  { value: "CLOSED", label: "Cerrada" },
  { value: "SPAM", label: "Spam" },
];

export default function QuoteRequestDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const { can, me } = useAuth();
  const canManage = can("quote_request.manage");
  const fallbackCurrency = me?.active_workspace?.currency || "USD";
  const [item, setItem] = useState<QuoteRequestDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<QuoteRequestDetail>(`/api/v1/quote-requests/${params.id}`)
      .then(setItem)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la solicitud."))
      .finally(() => setLoading(false));
  }, [params.id]);

  useEffect(() => { load(); }, [load]);

  const currency = item?.currency || fallbackCurrency;
  const discount = item?.estimated_discount_amount || 0;
  const unavailableItems = useMemo(() => item?.availability.items.filter((entry) => !entry.can_fulfill) || [], [item]);

  async function updateStatus(status: Exclude<QuoteRequestStatus, "CONVERTED">) {
    if (!item) return;
    setSaving("status");
    setError("");
    setNotice("");
    try {
      const result = await api<QuoteRequestDetail>(`/api/v1/quote-requests/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      setItem(result);
      setNotice(`Estado actualizado a ${quoteRequestStatusLabel(result.status).toLowerCase()}.`);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible actualizar el estado.");
    } finally {
      setSaving("");
    }
  }

  async function convert() {
    if (!item || !window.confirm("Se creará o reutilizará el cliente y se generará una cotización en borrador. ¿Continuar?")) return;
    setSaving("convert");
    setError("");
    setNotice("");
    try {
      const result = await api<QuoteRequestConversion>(`/api/v1/quote-requests/${item.id}/convert`, { method: "POST" });
      setNotice(`${result.reference_code} se convirtió en ${formatQuoteNumber(result.quote_number)}.`);
      router.push(`/quotes/${result.quote_id}/edit`);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible convertir la solicitud.");
    } finally {
      setSaving("");
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error && !item) return <section className="panel connection-panel"><span className="connection-icon">!</span><div><h2>Solicitud no disponible</h2><p>{error}</p><Link href="/quote-requests" className="text-link">← Volver a solicitudes</Link></div></section>;
  if (!item) return null;

  return (
    <div className="page-stack quote-request-detail-page">
      <div className="breadcrumbs"><Link href="/quote-requests">Solicitudes web</Link><span>/</span><span>{item.reference_code}</span></div>
      {error && <div className="form-alert">{error}</div>}
      {notice && <div className="success-banner">{notice}</div>}

      <section className="panel quote-request-detail-hero">
        <div>
          <div className="quote-request-detail-title"><span className={`quote-request-status ${quoteRequestStatusTone(item.status)}`}>{quoteRequestStatusLabel(item.status)}</span><small>RECIBIDA {formatDateTime(item.created_at).toUpperCase()}</small></div>
          <h2>{item.reference_code}</h2>
          <p>{item.customer_name} · {item.event_type || "Evento por definir"}</p>
        </div>
        <div className="quote-request-detail-actions">
          {item.converted_quote_id ? <Link className="button button-primary" href={`/quotes/${item.converted_quote_id}/edit`}>Abrir cotización</Link> : canManage && (item.status === "NEW" || item.status === "IN_REVIEW") ? <button className="button button-primary" type="button" disabled={saving === "convert"} onClick={() => void convert()}>{saving === "convert" ? "Convirtiendo…" : "Convertir a cotización"}</button> : null}
          {!canManage && <span className="read-only-pill">Solo lectura</span>}
        </div>
      </section>

      <div className="quote-request-detail-layout">
        <div className="quote-request-detail-main">
          <section className="panel quote-request-detail-section">
            <div className="panel-title-row"><div><p className="eyebrow">CLIENTE</p><h3>Contacto</h3></div></div>
            <div className="quote-request-contact-grid">
              <div><small>Nombre</small><strong>{item.customer_name}</strong></div>
              <div><small>Empresa</small><strong>{item.company_name || "—"}</strong></div>
              <div><small>Correo</small>{item.email ? <a href={`mailto:${item.email}`}>{item.email}</a> : <strong>—</strong>}</div>
              <div><small>Teléfono</small>{item.phone ? <a href={`tel:${item.phone}`}>{item.phone}</a> : <strong>—</strong>}</div>
              <div><small>Idioma</small><strong>{item.preferred_language === "en" ? "Inglés" : "Español"}</strong></div>
              <div><small>Consentimiento</small><strong>{item.consent_accepted ? `Aceptado · ${item.terms_version}` : "No registrado"}</strong></div>
            </div>
            <div className="quote-request-consent-copy"><small>Aviso aceptado · versión {item.terms_version}</small><p>{item.terms_text}</p></div>
          </section>

          <section className="panel quote-request-detail-section">
            <div className="panel-title-row"><div><p className="eyebrow">EVENTO</p><h3>Período y ubicación</h3></div></div>
            <div className="quote-request-event-grid">
              <div><small>Inicio</small><strong>{formatDateTime(item.start_at)}</strong></div>
              <div><small>Fin</small><strong>{formatDateTime(item.end_at)}</strong></div>
              <div><small>Tipo</small><strong>{item.event_type || "Por definir"}</strong></div>
              <div><small>Ubicación</small><strong>{item.event_location || "Por definir"}</strong></div>
            </div>
            {item.notes && <div className="quote-request-notes"><small>Notas del cliente</small><p>{item.notes}</p></div>}
          </section>

          <section className="panel quote-request-detail-section">
            <div className="panel-title-row"><div><p className="eyebrow">PAQUETES</p><h3>Selección enviada</h3></div></div>
            <div className="quote-request-package-detail-list">
              {item.packages.map((entry) => (
                <article key={entry.id}>
                  <span>{entry.quantity}×</span>
                  <div><strong>{entry.package_name}</strong><small>{entry.template.items.length} líneas expandidas</small></div>
                  <strong>{formatCurrency(entry.line_total, currency)}</strong>
                </article>
              ))}
            </div>
          </section>

          <section className="panel quote-request-detail-section">
            <div className="panel-title-row"><div><p className="eyebrow">LÍNEAS</p><h3>Snapshot comercial</h3><p>Estos valores se conservarán al convertir la solicitud en cotización.</p></div></div>
            <div className="quote-request-items-table-wrap">
              <table className="data-table quote-request-items-table">
                <thead><tr><th>Recurso</th><th>Cantidad</th><th>Precio unitario</th><th>Descuento</th><th>Total</th></tr></thead>
                <tbody>{item.items.map((entry) => <tr key={entry.id}><td data-label="Recurso"><strong>{entry.resource_name}</strong><small>{entry.description}</small></td><td data-label="Cantidad">{entry.quantity}</td><td data-label="Precio unitario">{formatCurrency(entry.unit_price, currency)}</td><td data-label="Descuento">{formatCurrency(entry.discount_amount, currency)}</td><td data-label="Total"><strong>{formatCurrency(entry.line_total, currency)}</strong></td></tr>)}</tbody>
              </table>
            </div>
          </section>

          <section className={`panel quote-request-detail-section quote-request-availability ${item.availability_available ? "available" : "attention"}`}>
            <div className="panel-title-row"><div><p className="eyebrow">DISPONIBILIDAD</p><h3>{item.availability_available ? "Disponible al momento de la solicitud" : "Requiere revisión comercial"}</h3><p>Es una fotografía preliminar; debe validarse nuevamente antes de crear una reserva.</p></div><span className="availability-icon">{item.availability_available ? "✓" : "!"}</span></div>
            <div className="quote-request-availability-grid">
              {item.availability.items.map((entry) => <div className={entry.can_fulfill ? "" : "missing"} key={entry.resource_id}><strong>{entry.resource_name}</strong><span>{entry.requested_quantity} solicitadas · {entry.available_quantity} disponibles</span><small>{entry.can_fulfill ? "Puede cubrirse" : `Faltan ${Math.max(0, entry.requested_quantity - entry.available_quantity)}`}</small></div>)}
            </div>
            {unavailableItems.length > 0 && <div className="info-callout warning"><strong>No bloquea la conversión</strong><span>El comercial puede convertirla en borrador y ajustar cantidades o recursos antes de enviarla.</span></div>}
          </section>
        </div>

        <aside className="quote-request-detail-sidebar">
          <section className="panel quote-request-total-card">
            <p className="eyebrow">ESTIMADO</p>
            <div><span>Subtotal</span><strong>{formatCurrency(item.estimated_subtotal, currency)}</strong></div>
            <div><span>Descuento de paquetes</span><strong>−{formatCurrency(discount, currency)}</strong></div>
            <div><span>Cargos extra</span><strong>{formatCurrency(item.estimated_extra_charges, currency)}</strong></div>
            <footer><span>Total estimado</span><strong>{formatCurrency(item.estimated_total, currency)}</strong></footer>
          </section>

          <section className="panel quote-request-status-card">
            <p className="eyebrow">SEGUIMIENTO</p>
            <h3>Estado de la solicitud</h3>
            {item.status === "CONVERTED" ? <div className="converted-message"><strong>Convertida</strong><span>Esta solicitud ya generó una cotización.</span></div> : canManage ? <>
              <select value={item.status} disabled={saving === "status"} onChange={(event) => void updateStatus(event.target.value as Exclude<QuoteRequestStatus, "CONVERTED">)}>{editableStatuses.map((status) => <option value={status.value} key={status.value}>{status.label}</option>)}</select>
              <small>Los estados cerrada y spam dejan la solicitud fuera del flujo activo. Puedes reabrirla como nueva.</small>
            </> : <div className="read-only-box">Tu rol no permite cambiar el estado.</div>}
            {item.handled_at && <p className="quote-request-handled">Última gestión: {formatDateTime(item.handled_at)}</p>}
          </section>

          <section className="panel quote-request-origin-card">
            <p className="eyebrow">ORIGEN</p><h3>Catálogo público</h3><p>Cliente y cotización conservarán atribución web después de la conversión.</p>
          </section>
        </aside>
      </div>
    </div>
  );
}
