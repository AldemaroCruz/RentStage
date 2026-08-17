"use client";

import Link from "next/link";
import type { CSSProperties, FormEvent } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, formatDateTime, formatQuoteNumber, formatReservationNumber } from "@/lib/format";
import type {
  PublicQuotePortalView,
  QuotePortalAvailabilityConflict,
  QuotePortalDecisionResult,
} from "@/lib/types";

const tokenStorageKey = "rentstage_quote_portal_token";
const tokenHeader = "X-RentStage-Quote-Token";

function initials(value: string): string {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase() || "RS";
}

function portalStatusLabel(status: PublicQuotePortalView["portal"]["status"]): string {
  return {
    ACTIVE: "Pendiente de respuesta",
    ACCEPTED: "Aceptada",
    REJECTED: "Rechazada",
    REVOKED: "Revocada",
    EXPIRED: "Vencida",
  }[status];
}

function readToken(): string {
  const encodedHash = window.location.hash.replace(/^#/, "");
  let fromHash = "";
  try {
    fromHash = decodeURIComponent(encodedHash).trim();
  } catch {
    fromHash = "";
  }
  if (fromHash) {
    window.sessionStorage.setItem(tokenStorageKey, fromHash);
    window.history.replaceState(window.history.state, "", window.location.pathname);
    return fromHash;
  }
  return window.sessionStorage.getItem(tokenStorageKey)?.trim() || "";
}

export default function QuotePortalPage() {
  const [token, setToken] = useState("");
  const [tokenReady, setTokenReady] = useState(false);
  const [view, setView] = useState<PublicQuotePortalView | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<"accept" | "reject" | "">("");
  const [mode, setMode] = useState<"accept" | "reject">("accept");
  const [responseName, setResponseName] = useState("");
  const [responseEmail, setResponseEmail] = useState("");
  const [rejectionReason, setRejectionReason] = useState("");
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [error, setError] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [decision, setDecision] = useState<QuotePortalDecisionResult | null>(null);
  const [conflict, setConflict] = useState<QuotePortalAvailabilityConflict | null>(null);

  useEffect(() => {
    setToken(readToken());
    setTokenReady(true);
  }, []);

  const tokenHeaders = useMemo(() => ({ [tokenHeader]: token }), [token]);

  const load = useCallback(async () => {
    if (!token) {
      setView(null);
      setError("El enlace está incompleto. Solicita a la empresa un enlace de cotización nuevo.");
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const item = await api<PublicQuotePortalView>("/api/v1/public/quote-portal", {
        headers: tokenHeaders,
      });
      setView(item);
      setError("");
      setResponseName((current) => current || item.portal.response_name || "");
    } catch (reason) {
      setView(null);
      if (reason instanceof ApiError && ["quote_portal_not_found", "quote_portal_unavailable"].includes(reason.code || "")) {
        window.sessionStorage.removeItem(tokenStorageKey);
      }
      setError(reason instanceof ApiError ? reason.message : "No fue posible abrir la cotización.");
    } finally {
      setLoading(false);
    }
  }, [token, tokenHeaders]);

  useEffect(() => {
    if (tokenReady) void load();
  }, [load, tokenReady]);

  const style = useMemo(
    () => ({ "--quote-accent": view?.portal.accent_color || "#6558e8" }) as CSSProperties,
    [view?.portal.accent_color],
  );

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!view || !token) return;
    setActing(mode);
    setError("");
    setFields({});
    setConflict(null);
    try {
      const result = mode === "accept"
        ? await api<QuotePortalDecisionResult>("/api/v1/public/quote-portal/accept", {
            method: "POST",
            headers: tokenHeaders,
            body: JSON.stringify({
              response_name: responseName,
              response_email: responseEmail,
              terms_accepted: termsAccepted,
            }),
          })
        : await api<QuotePortalDecisionResult>("/api/v1/public/quote-portal/reject", {
            method: "POST",
            headers: tokenHeaders,
            body: JSON.stringify({
              response_name: responseName,
              response_email: responseEmail,
              rejection_reason: rejectionReason,
            }),
          });
      setDecision(result);
      await load();
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.message);
        setFields(reason.fields || {});
        const availability = reason.payload.availability as QuotePortalAvailabilityConflict | undefined;
        if (availability) setConflict(availability);
      } else {
        setError("No fue posible registrar tu respuesta.");
      }
    } finally {
      setActing("");
    }
  }

  if (!tokenReady || loading) {
    return (
      <div className="quote-portal-loading">
        <span className="brand-mark">
          <span className="brand-wave" />
          <span className="brand-wave brand-wave-two" />
          <span className="brand-wave brand-wave-three" />
        </span>
        <p>Preparando cotización…</p>
      </div>
    );
  }

  if (!view) {
    return (
      <main className="quote-portal-error-page">
        <section>
          <span>!</span>
          <h1>Enlace no disponible</h1>
          <p>{error || "La cotización no existe, fue reemplazada o dejó de estar disponible."}</p>
        </section>
      </main>
    );
  }

  const { tenant, portal, quote } = view;
  const decided = portal.status !== "ACTIVE";
  const accepted = portal.status === "ACCEPTED" || decision?.status === "ACCEPTED";
  const rejected = portal.status === "REJECTED" || decision?.status === "REJECTED";
  const unavailableItems = conflict?.items.filter((item) => !item.can_fulfill) || [];
  const reservationNumber = decision?.reservation_number || quote.reservation_number;
  const acceptedMessage = reservationNumber
    ? "La propuesta quedó confirmada y RentStage creó la reserva correspondiente."
    : "La propuesta quedó registrada como aceptada. La empresa continuará con la coordinación y creación de la reserva.";

  return (
    <div className="quote-portal-shell" style={style}>
      <header className="quote-portal-header">
        <Link href={`/p/${tenant.slug}`} className="quote-portal-brand">
          {tenant.logo_url ? (
            <span className="quote-portal-logo quote-portal-logo-image">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={tenant.logo_url} alt="" referrerPolicy="no-referrer" />
            </span>
          ) : (
            <span className="quote-portal-logo">{initials(tenant.name)}</span>
          )}
          <span><strong>{tenant.name}</strong><small>Portal de cotización</small></span>
        </Link>
        <span className={`quote-portal-state state-${portal.status.toLowerCase()}`}>{portalStatusLabel(portal.status)}</span>
      </header>

      <main className="quote-portal-main">
        {(decision || decided) && (
          <section className={`quote-portal-decision-banner ${accepted ? "accepted" : rejected ? "rejected" : "closed"}`}>
            <div className="quote-portal-decision-icon">{accepted ? "✓" : rejected ? "×" : "!"}</div>
            <div>
              <p className="eyebrow">RESPUESTA REGISTRADA</p>
              <h1>{accepted ? "Cotización aceptada" : rejected ? "Cotización rechazada" : portal.status === "EXPIRED" ? "Cotización vencida" : "Enlace cerrado"}</h1>
              <p>{accepted ? acceptedMessage : rejected ? "La empresa recibió tu respuesta. Puedes contactarla para solicitar una propuesta diferente." : "Contacta a la empresa para solicitar un nuevo enlace o una actualización."}</p>
              {reservationNumber && <strong>{formatReservationNumber(reservationNumber)}</strong>}
            </div>
          </section>
        )}

        <section className="quote-portal-intro">
          <div>
            <p className="eyebrow">{formatQuoteNumber(quote.quote_number)}</p>
            <h1>{portal.headline}</h1>
            <p>{portal.introduction}</p>
          </div>
          <div className="quote-portal-validity">
            <small>Válida hasta</small>
            <strong>{formatDateTime(portal.expires_at)}</strong>
            <span>Creada {formatDateTime(quote.created_at)}</span>
          </div>
        </section>

        <div className="quote-portal-layout">
          <div className="quote-portal-document">
            <section className="quote-portal-card quote-portal-event-card">
              <div><small>Preparada para</small><strong>{quote.customer_name}</strong></div>
              <div><small>Evento</small><strong>{quote.event_type || "Alquiler de equipo"}</strong></div>
              <div><small>Ubicación</small><strong>{quote.event_location || "Por confirmar"}</strong></div>
              <div><small>Período</small><strong>{formatDateTime(quote.start_at)}</strong><span>hasta {formatDateTime(quote.end_at)}</span></div>
            </section>

            <section className="quote-portal-card quote-portal-items-card">
              <header>
                <div><p className="eyebrow">DETALLE COMERCIAL</p><h2>Recursos incluidos</h2></div>
                <span>{quote.items.length} líneas</span>
              </header>
              <div className="quote-portal-item-list">
                {quote.items.map((item, index) => (
                  <article key={`${item.description}-${index}`}>
                    <span className="quote-portal-item-number">{String(index + 1).padStart(2, "0")}</span>
                    <div>
                      <strong>{item.description}</strong>
                      <small>{item.resource_name}{item.discount_amount > 0 ? ` · descuento ${formatCurrency(item.discount_amount, tenant.currency)}` : ""}</small>
                    </div>
                    <span>{item.quantity} × {formatCurrency(item.unit_price, tenant.currency)}</span>
                    <strong>{formatCurrency(item.line_total, tenant.currency)}</strong>
                  </article>
                ))}
              </div>
              <div className="quote-portal-totals">
                <span>Subtotal<strong>{formatCurrency(quote.subtotal, tenant.currency)}</strong></span>
                {quote.discount_amount > 0 && <span>Descuento<strong>− {formatCurrency(quote.discount_amount, tenant.currency)}</strong></span>}
                {quote.extra_charges > 0 && <span>Cargos adicionales<strong>{formatCurrency(quote.extra_charges, tenant.currency)}</strong></span>}
                <span className="total">Total<strong>{formatCurrency(quote.total, tenant.currency)}</strong></span>
              </div>
            </section>

            <section className="quote-portal-card quote-portal-terms-card">
              <p className="eyebrow">TÉRMINOS · VERSIÓN {portal.terms_version}</p>
              <h2>Condiciones de aceptación</h2>
              <p>{portal.terms_text}</p>
            </section>
          </div>

          <aside className="quote-portal-response-column">
            <section className="quote-portal-card quote-portal-response-card">
              <p className="eyebrow">TU RESPUESTA</p>
              <h2>{decided ? portalStatusLabel(portal.status) : mode === "accept" ? "Aceptar cotización" : "Rechazar cotización"}</h2>
              {!decided && <p>La respuesta queda registrada con fecha, versión de términos y evidencia técnica del enlace.</p>}

              {error && <div className="quote-portal-alert">{error}</div>}
              {conflict && (
                <div className="quote-portal-conflict">
                  <strong>Disponibilidad por revisar</strong>
                  <p>La empresa debe proponerte una alternativa para:</p>
                  {unavailableItems.map((item) => <span key={item.resource_name}>{item.requested_quantity} × {item.resource_name}</span>)}
                </div>
              )}

              {!decided && (
                <form onSubmit={submit}>
                  <div className="quote-portal-mode-tabs">
                    <button type="button" className={mode === "accept" ? "active" : ""} onClick={() => { setMode("accept"); setError(""); setFields({}); }}>Aceptar</button>
                    {portal.can_reject && <button type="button" className={mode === "reject" ? "active reject" : ""} onClick={() => { setMode("reject"); setError(""); setFields({}); }}>Rechazar</button>}
                  </div>
                  <label>
                    <span>Nombre de quien responde{portal.require_response_name ? " *" : ""}</span>
                    <input value={responseName} onChange={(event) => setResponseName(event.target.value)} autoComplete="name" />
                    {fields.response_name && <small>{fields.response_name}</small>}
                  </label>
                  <label>
                    <span>Correo para constancia</span>
                    <input type="email" value={responseEmail} onChange={(event) => setResponseEmail(event.target.value)} autoComplete="email" />
                    {fields.response_email && <small>{fields.response_email}</small>}
                  </label>
                  {mode === "reject" && (
                    <label>
                      <span>Motivo opcional</span>
                      <textarea rows={4} value={rejectionReason} onChange={(event) => setRejectionReason(event.target.value)} placeholder="Cuéntanos qué debería ajustarse…" />
                      {fields.rejection_reason && <small>{fields.rejection_reason}</small>}
                    </label>
                  )}
                  {mode === "accept" && (
                    <label className="quote-portal-consent">
                      <input type="checkbox" checked={termsAccepted} onChange={(event) => setTermsAccepted(event.target.checked)} />
                      <span>He revisado la cotización y acepto los términos versión {portal.terms_version}.</span>
                      {fields.terms_accepted && <small>{fields.terms_accepted}</small>}
                    </label>
                  )}
                  <button className={`quote-portal-submit ${mode === "reject" ? "reject" : ""}`} disabled={Boolean(acting) || (mode === "accept" ? !portal.can_accept : !portal.can_reject)}>
                    {acting ? "Registrando…" : mode === "accept" ? `Aceptar ${formatCurrency(quote.total, tenant.currency)}` : "Confirmar rechazo"}
                  </button>
                </form>
              )}

              {decided && (
                <div className="quote-portal-evidence">
                  <span>Estado<strong>{portalStatusLabel(portal.status)}</strong></span>
                  {portal.response_name && <span>Respondida por<strong>{portal.response_name}</strong></span>}
                  {portal.decision_at && <span>Fecha<strong>{formatDateTime(portal.decision_at)}</strong></span>}
                  {portal.rejection_reason && <span>Motivo<strong>{portal.rejection_reason}</strong></span>}
                </div>
              )}
            </section>

            <section className="quote-portal-contact-card">
              <strong>¿Necesitas un ajuste?</strong>
              <p>Contacta directamente a {tenant.name} antes de responder.</p>
              {tenant.phone && <a href={`tel:${tenant.phone}`}>{tenant.phone}</a>}
              {tenant.email && <a href={`mailto:${tenant.email}`}>{tenant.email}</a>}
            </section>
          </aside>
        </div>
      </main>

      <footer className="quote-portal-footer">
        <span>{tenant.name}</span>
        <small>Respuesta segura administrada con RentStage.</small>
      </footer>
    </div>
  );
}
