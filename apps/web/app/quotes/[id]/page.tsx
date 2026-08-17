"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { QuoteStatusBadge } from "@/components/QuoteStatusBadge";
import { useAuth } from "@/components/AuthProvider";
import { ApiError, api } from "@/lib/api";
import {
  formatCurrency,
  formatDateTime,
  formatQuoteNumber,
  formatReservationNumber,
} from "@/lib/format";
import type { AvailabilityResult, QuoteDetail, ReservationDetail } from "@/lib/types";

function portalLabel(status: NonNullable<QuoteDetail["portal"]>["status"]): string {
  return {
    ACTIVE: "Activo",
    ACCEPTED: "Aceptado",
    REJECTED: "Rechazado",
    REVOKED: "Revocado",
    EXPIRED: "Vencido",
  }[status];
}

export default function QuoteDetailPage() {
  const { can } = useAuth();
  const canManageQuote = can("quote.manage");
  const canManageReservation = can("reservation.manage");
  const canManageBilling = can("billing.manage");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [quote, setQuote] = useState<QuoteDetail | null>(null);
  const [availability, setAvailability] = useState<AvailabilityResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [acting, setActing] = useState("");
  const [freshPortalURL, setFreshPortalURL] = useState("");
  const [copied, setCopied] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api<QuoteDetail>(`/api/v1/quotes/${params.id}`)
      .then((response) => {
        setQuote(response);
        setError("");
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la cotización."))
      .finally(() => setLoading(false));
  }, [params.id]);

  useEffect(() => load(), [load]);

  async function transition(action: "send" | "accept" | "reject" | "cancel", confirmation: string) {
    if (!window.confirm(confirmation)) return;
    setActing(action);
    setError("");
    setCopied(false);
    try {
      const updated = await api<QuoteDetail>(`/api/v1/quotes/${params.id}/${action}`, { method: "POST" });
      setQuote(updated);
      setAvailability(null);
      if (updated.portal?.public_url) setFreshPortalURL(updated.portal.public_url);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible cambiar el estado de la cotización.");
    } finally {
      setActing("");
    }
  }

  async function portalAction(action: "reissue" | "revoke") {
    if (!quote) return;
    const confirmation = action === "reissue"
      ? "¿Rotar el enlace? El enlace anterior dejará de funcionar inmediatamente."
      : "¿Revocar el enlace público? La cotización seguirá enviada y podrás generar otro enlace después.";
    if (!window.confirm(confirmation)) return;
    setActing(`portal-${action}`);
    setError("");
    setCopied(false);
    try {
      const updated = await api<QuoteDetail>(
        action === "reissue" ? `/api/v1/quotes/${quote.id}/portal/reissue` : `/api/v1/quotes/${quote.id}/portal`,
        { method: action === "reissue" ? "POST" : "DELETE" },
      );
      setQuote(updated);
      setFreshPortalURL(updated.portal?.public_url || "");
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible actualizar el enlace público.");
    } finally {
      setActing("");
    }
  }

  async function copyPortalURL() {
    if (!freshPortalURL) return;
    try {
      await navigator.clipboard.writeText(freshPortalURL);
      setCopied(true);
    } catch {
      setError("No fue posible copiar automáticamente. Selecciona el enlace y cópialo manualmente.");
    }
  }

  async function checkAvailability() {
    if (!quote) return;
    setActing("availability");
    setError("");
    try {
      const result = await api<AvailabilityResult>("/api/v1/availability/check", {
        method: "POST",
        body: JSON.stringify({
          start_at: quote.start_at,
          end_at: quote.end_at,
          items: quote.items.map((item) => ({ resource_id: item.resource_id, quantity: item.quantity })),
        }),
      });
      setAvailability(result);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible validar disponibilidad.");
    } finally {
      setActing("");
    }
  }

  async function convertToReservation() {
    if (!quote || !window.confirm("¿Crear una reserva desde esta cotización aceptada? RentStage volverá a validar disponibilidad dentro de una transacción.")) return;
    setActing("convert");
    setError("");
    try {
      const reservation = await api<ReservationDetail>(`/api/v1/quotes/${quote.id}/convert-to-reservation`, { method: "POST" });
      router.push(`/reservations/${reservation.id}`);
    } catch (reason) {
      if (reason instanceof ApiError) {
        const conflict = reason.payload.availability as AvailabilityResult | undefined;
        if (conflict) setAvailability(conflict);
        const existingID = reason.payload.reservation_id;
        if (typeof existingID === "string") {
          router.push(`/reservations/${existingID}`);
          return;
        }
        setError(reason.message);
      } else {
        setError("No fue posible convertir la cotización en reserva.");
      }
    } finally {
      setActing("");
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error && !quote) return <div className="panel inline-error">{error}<button onClick={load}>Reintentar</button></div>;
  if (!quote) return <div className="panel inline-error">Cotización no encontrada.</div>;

  const portal = quote.portal;

  return (
    <div className="page-stack">
      <div className="breadcrumbs"><Link href="/quotes">Cotizaciones</Link><span>/</span><span>{formatQuoteNumber(quote.quote_number)}</span></div>

      {error && <div className="form-alert quote-action-alert">{error}</div>}
      {freshPortalURL && (
        <section className="quote-portal-one-time panel">
          <div><p className="eyebrow">ENLACE DISPONIBLE UNA SOLA VEZ</p><h3>Comparte este portal con el cliente</h3><p>RentStage guarda únicamente el hash SHA-256. Si pierdes el enlace tendrás que rotarlo.</p></div>
          <div className="quote-portal-copy-row"><input readOnly value={freshPortalURL} onFocus={(event) => event.currentTarget.select()} /><button className="button button-primary" onClick={() => void copyPortalURL()}>{copied ? "Copiado ✓" : "Copiar enlace"}</button><a className="button button-secondary" href={freshPortalURL} target="_blank" rel="noreferrer">Abrir</a></div>
        </section>
      )}

      <section className="quote-detail-hero panel">
        <div>
          <div className="quote-detail-title-row">
            <div><p className="eyebrow">QUOTE DOCUMENT</p><h2>{formatQuoteNumber(quote.quote_number)}</h2></div>
            <QuoteStatusBadge status={quote.status} />
          </div>
          <p className="quote-detail-event">{quote.event_type || "Cotización de alquiler"}</p>
          <div className="quote-detail-meta">
            <span><small>Cliente</small><Link href={`/customers/${quote.customer_id}`}>{quote.customer_name}</Link></span>
            <span><small>Ubicación</small><strong>{quote.event_location || "Sin ubicación"}</strong></span>
            <span><small>Creada</small><strong>{formatDateTime(quote.created_at)}</strong></span>
          </div>
        </div>
        <div className="quote-detail-actions">
          {canManageQuote && quote.status === "DRAFT" && (
            <>
              <Link href={`/quotes/${quote.id}/edit`} className="button button-secondary">Editar borrador</Link>
              <button className="button button-primary" disabled={Boolean(acting)} onClick={() => void transition("send", "¿Enviar esta cotización y generar un enlace seguro para el cliente? Después ya no podrá editarse.")}>{acting === "send" ? "Generando…" : "Enviar y generar enlace"}</button>
              <button className="button button-danger-ghost" disabled={Boolean(acting)} onClick={() => void transition("cancel", "¿Cancelar este borrador?")}>Cancelar</button>
            </>
          )}
          {canManageQuote && quote.status === "SENT" && (
            <>
              <button className="button button-secondary" disabled={Boolean(acting)} onClick={() => void transition("accept", "¿Registrar manualmente que el cliente aceptó esta cotización? La reserva no se creará automáticamente en este flujo administrativo.")}>{acting === "accept" ? "Procesando…" : "Aceptar manualmente"}</button>
              <button className="button button-secondary" disabled={Boolean(acting)} onClick={() => void transition("reject", "¿Registrar manualmente que el cliente rechazó esta cotización?")}>Rechazar manualmente</button>
              <button className="button button-danger-ghost" disabled={Boolean(acting)} onClick={() => void transition("cancel", "¿Cancelar esta cotización enviada?")}>Cancelar</button>
            </>
          )}
          {quote.status === "ACCEPTED" && !quote.reservation_id && (
            <>
              <button className="button button-secondary" disabled={Boolean(acting)} onClick={() => void checkAvailability()}>{acting === "availability" ? "Validando…" : "Validar disponibilidad"}</button>
              {canManageReservation && <button className="button button-primary" disabled={Boolean(acting)} onClick={() => void convertToReservation()}>{acting === "convert" ? "Creando…" : "Crear reserva"}</button>}
            </>
          )}
          {quote.reservation_id && quote.reservation_number && <Link className="button button-primary" href={`/reservations/${quote.reservation_id}`}>Ver {formatReservationNumber(quote.reservation_number)} →</Link>}
          {canManageBilling && quote.status === "ACCEPTED" && <Link className="button button-secondary" href={`/invoices/new?source_type=QUOTE&source_id=${quote.id}`}>Crear factura</Link>}
        </div>
      </section>

      <div className="quote-detail-grid">
        <section className="panel quote-document-panel">
          <div className="quote-document-header">
            <div><p className="eyebrow">PERÍODO DE BLOQUEO PROPUESTO</p><h3>{formatDateTime(quote.start_at)}</h3><span>hasta {formatDateTime(quote.end_at)}</span></div>
            {quote.expires_at && <div className="quote-expiration"><small>Válida hasta</small><strong>{formatDateTime(quote.expires_at)}</strong></div>}
          </div>

          <div className="data-table-wrap quote-items-table-wrap">
            <table className="data-table quote-items-table">
              <thead><tr><th>Descripción</th><th>Cantidad</th><th>Precio</th><th>Descuento</th><th>Total</th></tr></thead>
              <tbody>{quote.items.map((item) => <tr key={item.id}><td><strong className="table-primary-copy">{item.description}</strong><span className="table-subline">{item.resource_name}</span></td><td>{item.quantity}</td><td>{formatCurrency(item.unit_price)}</td><td>{formatCurrency(item.discount_amount)}</td><td><strong className="table-primary-copy">{formatCurrency(item.line_total)}</strong></td></tr>)}</tbody>
            </table>
          </div>

          <div className="quote-document-footer">
            <div className="quote-notes-block"><small>Notas internas</small><p>{quote.notes || "Sin notas adicionales."}</p></div>
            <dl className="quote-total-list"><div><dt>Subtotal</dt><dd>{formatCurrency(quote.subtotal)}</dd></div><div><dt>Descuento</dt><dd>− {formatCurrency(quote.discount_amount)}</dd></div><div><dt>Cargos adicionales</dt><dd>{formatCurrency(quote.extra_charges)}</dd></div><div className="quote-total-final"><dt>Total</dt><dd>{formatCurrency(quote.total)}</dd></div></dl>
          </div>
        </section>

        <aside className="page-stack quote-side-column">
          <section className="panel quote-side-card quote-portal-admin-card">
            <div className="quote-portal-admin-heading"><div><p className="eyebrow">CUSTOMER PORTAL</p><h3>Respuesta online</h3></div>{portal && <span className={`quote-portal-admin-status status-${portal.status.toLowerCase()}`}>{portalLabel(portal.status)}</span>}</div>
            {!portal ? (
              <>
                <p>{quote.status === "DRAFT" ? "El enlace se generará al enviar la cotización." : "Esta cotización enviada todavía no tiene un enlace público."}</p>
                {canManageQuote && quote.status === "SENT" && <button className="button button-primary button-full" disabled={Boolean(acting)} onClick={() => void portalAction("reissue")}>{acting === "portal-reissue" ? "Generando…" : "Generar enlace"}</button>}
              </>
            ) : (
              <>
                <div className="quote-portal-admin-metrics"><span>Vistas<strong>{portal.view_count}</strong></span><span>Revisión<strong>#{portal.revision}</strong></span></div>
                <dl className="profile-definition-list compact-definition-list">
                  <div><dt>Vence</dt><dd>{formatDateTime(portal.expires_at)}</dd></div>
                  <div><dt>Última vista</dt><dd>{portal.last_viewed_at ? formatDateTime(portal.last_viewed_at) : "Sin abrir"}</dd></div>
                  <div><dt>Términos</dt><dd>v{portal.terms_version}</dd></div>
                  {portal.decision_at && <div><dt>Respondida</dt><dd>{formatDateTime(portal.decision_at)}</dd></div>}
                  {portal.response_name && <div><dt>Persona</dt><dd>{portal.response_name}</dd></div>}
                </dl>
                {portal.rejection_reason && <div className="quote-portal-admin-reason"><small>Motivo</small><p>{portal.rejection_reason}</p></div>}
                {canManageQuote && quote.status === "SENT" && <div className="quote-portal-admin-actions"><button className="button button-secondary" disabled={Boolean(acting)} onClick={() => void portalAction("reissue")}>{acting === "portal-reissue" ? "Rotando…" : portal.status === "ACTIVE" ? "Rotar enlace" : "Generar nuevo enlace"}</button>{portal.status === "ACTIVE" && <button className="button button-danger-ghost" disabled={Boolean(acting)} onClick={() => void portalAction("revoke")}>{acting === "portal-revoke" ? "Revocando…" : "Revocar"}</button>}</div>}
              </>
            )}
            <Link className="text-link" href="/settings/quote-portal">Configurar portal →</Link>
          </section>

          <section className="panel quote-side-card">
            <p className="eyebrow">CLIENTE</p>
            <div className="quote-customer-mini"><span>{quote.customer_name.slice(0, 2).toUpperCase()}</span><div><strong>{quote.customer_name}</strong><small>{quote.customer_phone || "Sin teléfono"}</small></div></div>
            <Link href={`/customers/${quote.customer_id}`} className="text-link">Ver perfil del cliente →</Link>
          </section>

          <section className="panel quote-side-card availability-card">
            <p className="eyebrow">BOOKING CORE</p>
            {quote.reservation_id && quote.reservation_number ? (
              <><h3>Inventario reservado</h3><p>Esta cotización ya fue convertida y las cantidades se administran desde la reserva.</p><Link href={`/reservations/${quote.reservation_id}`} className="text-link">Abrir {formatReservationNumber(quote.reservation_number)} →</Link></>
            ) : quote.status === "ACCEPTED" ? (
              <><h3>{availability ? (availability.available ? "Disponible para reservar" : "Conflicto de disponibilidad") : "Lista para validar"}</h3><p>La aceptación online crea la reserva en la misma transacción. Para una aceptación manual, valida y conviértela desde aquí.</p>{availability && <div className="availability-list">{availability.items.map((item) => <article key={item.resource_id} className={item.can_fulfill ? "availability-ok" : "availability-conflict"}><div><strong>{item.resource_name}</strong><small>Solicitadas {item.requested_quantity}</small></div><span>{item.available_quantity} disponibles</span></article>)}</div>}</>
            ) : (
              <><h3>Conversión a reserva</h3><p>La cotización debe estar aceptada antes de bloquear inventario.</p><span className="future-status quote-future-status">Esperando aceptación</span></>
            )}
          </section>

          <section className="panel quote-side-card">
            <p className="eyebrow">TRAZABILIDAD</p>
            <dl className="profile-definition-list compact-definition-list"><div><dt>Última edición</dt><dd>{formatDateTime(quote.updated_at)}</dd></div><div><dt>Líneas</dt><dd>{quote.item_count}</dd></div><div><dt>Estado</dt><dd>{quote.status}</dd></div></dl>
          </section>
        </aside>
      </div>
    </div>
  );
}
