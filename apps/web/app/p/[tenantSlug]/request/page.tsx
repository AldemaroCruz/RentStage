"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { PublicCatalogShell } from "@/components/PublicCatalogShell";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, formatDateTime } from "@/lib/format";
import type {
  PublicAvailabilityResult,
  PublicCatalog,
  PublicQuoteRequestInput,
  QuoteRequestReceipt,
  QuoteRequestSelection,
} from "@/lib/types";

type ContactForm = {
  first_name: string;
  last_name: string;
  phone: string;
  email: string;
  company_name: string;
  event_type: string;
  event_location: string;
  start_at: string;
  end_at: string;
  notes: string;
  consent_accepted: boolean;
  website: string;
};

const initialForm: ContactForm = {
  first_name: "",
  last_name: "",
  phone: "",
  email: "",
  company_name: "",
  event_type: "",
  event_location: "",
  start_at: "",
  end_at: "",
  notes: "",
  consent_accepted: false,
  website: "",
};

function toRFC3339(value: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toISOString();
}

export default function PublicQuoteRequestPage() {
  const params = useParams<{ tenantSlug: string }>();
  const [catalog, setCatalog] = useState<PublicCatalog | null>(null);
  const [form, setForm] = useState<ContactForm>(initialForm);
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [availability, setAvailability] = useState<PublicAvailabilityResult | null>(null);
  const [receipt, setReceipt] = useState<QuoteRequestReceipt | null>(null);
  const [presetApplied, setPresetApplied] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<PublicCatalog>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}`)
      .then(setCatalog)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible abrir el formulario."))
      .finally(() => setLoading(false));
  }, [params.tenantSlug]);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    if (!catalog || presetApplied || typeof window === "undefined") return;
    const preset = new URLSearchParams(window.location.search).get("package");
    if (preset && catalog.packages.some((item) => item.slug === preset)) {
      setQuantities({ [preset]: 1 });
    }
    setPresetApplied(true);
  }, [catalog, presetApplied]);

  const selections = useMemo<QuoteRequestSelection[]>(() => Object.entries(quantities)
    .filter(([, quantity]) => quantity > 0)
    .map(([package_slug, quantity]) => ({ package_slug, quantity })), [quantities]);

  const selectedPackages = useMemo(() => catalog?.packages.filter((item) => (quantities[item.slug] || 0) > 0) || [], [catalog, quantities]);
  const estimatedTotal = useMemo(() => selectedPackages.reduce((total, item) => total + (item.effective_price || 0) * (quantities[item.slug] || 0), 0), [selectedPackages, quantities]);

  function updateField<K extends keyof ContactForm>(name: K, value: ContactForm[K]) {
    setForm((current) => ({ ...current, [name]: value }));
    setFields((current) => {
      if (!current[name]) return current;
      const next = { ...current };
      delete next[name];
      return next;
    });
    setAvailability(null);
  }

  function updateQuantity(slug: string, quantity: number) {
    const safe = Math.max(0, Math.min(100, Number.isFinite(quantity) ? quantity : 0));
    setQuantities((current) => ({ ...current, [slug]: safe }));
    setAvailability(null);
    setFields((current) => {
      const next = { ...current };
      delete next.selections;
      return next;
    });
  }

  function validateBasics(): boolean {
    const next: Record<string, string> = {};
    if (selections.length === 0) next.selections = "Selecciona al menos un paquete.";
    if (!form.start_at) next.start_at = "Indica la fecha y hora inicial.";
    if (!form.end_at) next.end_at = "Indica la fecha y hora final.";
    if (form.start_at && form.end_at && new Date(form.end_at) <= new Date(form.start_at)) next.end_at = "La fecha final debe ser posterior a la inicial.";
    setFields(next);
    return Object.keys(next).length === 0;
  }

  async function checkAvailability() {
    if (!validateBasics()) return;
    setChecking(true);
    setError("");
    try {
      const result = await api<PublicAvailabilityResult>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}/availability`, {
        method: "POST",
        body: JSON.stringify({ start_at: toRFC3339(form.start_at), end_at: toRFC3339(form.end_at), selections }),
      });
      setAvailability(result);
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.message);
        setFields(reason.fields || {});
      } else {
        setError("No fue posible consultar la disponibilidad.");
      }
    } finally {
      setChecking(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    const nextFields: Record<string, string> = {};
    if (!validateBasics()) return;
    if (!form.first_name.trim()) nextFields.first_name = "Escribe tu nombre.";
    if (!form.email.trim() && !form.phone.trim()) nextFields.contact = "Proporciona correo o teléfono.";
    if (!form.consent_accepted) nextFields.consent_accepted = "Debes aceptar el aviso para continuar.";
    if (Object.keys(nextFields).length > 0) {
      setFields((current) => ({ ...current, ...nextFields }));
      return;
    }

    setSubmitting(true);
    setError("");
    setFields({});
    const input: PublicQuoteRequestInput = {
      first_name: form.first_name.trim(),
      last_name: form.last_name.trim(),
      phone: form.phone.trim() || undefined,
      email: form.email.trim() || undefined,
      company_name: form.company_name.trim() || undefined,
      preferred_language: "es",
      event_type: form.event_type.trim() || undefined,
      event_location: form.event_location.trim() || undefined,
      start_at: toRFC3339(form.start_at),
      end_at: toRFC3339(form.end_at),
      notes: form.notes.trim(),
      consent_accepted: form.consent_accepted,
      website: form.website,
      selections,
    };
    try {
      const result = await api<QuoteRequestReceipt>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}/quote-requests`, {
        method: "POST",
        body: JSON.stringify(input),
      });
      setReceipt(result);
      setAvailability(result.availability);
      if (typeof window !== "undefined") window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.message);
        setFields(reason.fields || {});
      } else {
        setError("No fue posible enviar la solicitud.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) return <div className="public-page-state"><span className="public-loader" /><p>Preparando solicitud…</p></div>;
  if (error && !catalog) return <div className="public-page-state public-page-error"><strong>Formulario no disponible</strong><p>{error}</p><button type="button" onClick={load}>Reintentar</button></div>;
  if (!catalog) return null;

  if (receipt) {
    return (
      <PublicCatalogShell tenant={catalog.tenant} settings={catalog.settings}>
        <section className="public-request-success">
          <span className="public-success-icon">✓</span>
          <p className="public-kicker">SOLICITUD RECIBIDA</p>
          <h1>Gracias, {form.first_name || "hemos recibido tu información"}</h1>
          <p>El equipo de {catalog.tenant.name} podrá revisar los detalles y preparar una cotización formal.</p>
          <div className="public-receipt-card">
            <div><small>Referencia</small><strong>{receipt.reference_code}</strong></div>
            <div><small>Fecha de envío</small><strong>{formatDateTime(receipt.created_at)}</strong></div>
            {receipt.estimated_total !== undefined && <div><small>Estimado actual</small><strong>{formatCurrency(receipt.estimated_total, catalog.tenant.currency)}</strong></div>}
            <div><small>Disponibilidad preliminar</small><strong className={receipt.availability_available ? "available" : "attention"}>{receipt.availability_available ? "Disponible" : "Requiere revisión"}</strong></div>
          </div>
          {!receipt.availability_available && <p className="public-success-note">La solicitud fue enviada aunque una o más cantidades requieren revisión. La empresa puede proponerte una alternativa.</p>}
          <div className="public-success-actions"><Link className="public-button primary" href={`/p/${catalog.tenant.slug}`}>Volver al catálogo</Link></div>
        </section>
      </PublicCatalogShell>
    );
  }

  return (
    <PublicCatalogShell tenant={catalog.tenant} settings={catalog.settings}>
      <div className="public-detail-breadcrumb"><Link href={`/p/${catalog.tenant.slug}`}>Catálogo</Link><span>/</span><span>Solicitar cotización</span></div>
      <section className="public-request-heading">
        <div><p className="public-kicker">SOLICITUD DE COTIZACIÓN</p><h1>Cuéntanos sobre tu evento</h1><p>Selecciona uno o más paquetes, indica la fecha y deja tus datos de contacto.</p></div>
        <div><strong>1</strong><span>Paquetes</span><strong>2</strong><span>Fecha</span><strong>3</strong><span>Contacto</span></div>
      </section>

      {!catalog.settings.quote_requests_enabled ? (
        <section className="public-empty large"><strong>Las solicitudes web están deshabilitadas</strong><p>Utiliza los datos de contacto del catálogo para comunicarte con la empresa.</p><Link href={`/p/${catalog.tenant.slug}`}>Volver al catálogo</Link></section>
      ) : (
        <form className="public-request-layout" onSubmit={submit}>
          <div className="public-request-main">
            {error && <div className="public-form-alert">{error}</div>}
            <section className="public-request-panel">
              <div className="public-request-panel-heading"><span>1</span><div><h2>Selecciona paquetes</h2><p>Puedes solicitar varias unidades del mismo paquete.</p></div></div>
              {catalog.packages.length === 0 ? (
                <div className="public-empty"><strong>No hay paquetes disponibles</strong><p>La empresa aún no publicó opciones para solicitudes web.</p></div>
              ) : (
                <div className="public-request-package-list">
                  {catalog.packages.map((item) => {
                    const quantity = quantities[item.slug] || 0;
                    return (
                      <article className={quantity > 0 ? "selected" : ""} key={item.slug}>
                        <button type="button" className="public-request-package-toggle" onClick={() => updateQuantity(item.slug, quantity > 0 ? 0 : 1)} aria-pressed={quantity > 0}>
                          <span className="public-check">{quantity > 0 ? "✓" : ""}</span>
                          <div><strong>{item.name}</strong><p>{item.description}</p><small>{item.guest_capacity ? `Hasta ${item.guest_capacity} personas · ` : ""}{item.item_count} tipos de recurso</small></div>
                          {item.effective_price !== undefined && <em>{formatCurrency(item.effective_price, catalog.tenant.currency)}</em>}
                        </button>
                        {quantity > 0 && <label><span>Cantidad</span><input type="number" min={1} max={100} value={quantity} onChange={(event) => updateQuantity(item.slug, Number(event.target.value))} /></label>}
                      </article>
                    );
                  })}
                </div>
              )}
              {fields.selections && <small className="public-field-error">{fields.selections}</small>}
            </section>

            <section className="public-request-panel">
              <div className="public-request-panel-heading"><span>2</span><div><h2>Fecha y evento</h2><p>Usaremos este período para la consulta preliminar de disponibilidad.</p></div></div>
              <div className="public-form-grid two">
                <label><span>Inicio *</span><input type="datetime-local" value={form.start_at} onChange={(event) => updateField("start_at", event.target.value)} />{fields.start_at && <small>{fields.start_at}</small>}</label>
                <label><span>Fin *</span><input type="datetime-local" value={form.end_at} onChange={(event) => updateField("end_at", event.target.value)} />{fields.end_at && <small>{fields.end_at}</small>}</label>
                <label><span>Tipo de evento</span><input value={form.event_type} onChange={(event) => updateField("event_type", event.target.value)} placeholder="Cumpleaños, boda, conferencia…" />{fields.event_type && <small>{fields.event_type}</small>}</label>
                <label><span>Ubicación</span><input value={form.event_location} onChange={(event) => updateField("event_location", event.target.value)} placeholder="Ciudad o dirección" />{fields.event_location && <small>{fields.event_location}</small>}</label>
              </div>
              <button type="button" className="public-button secondary dark" disabled={checking || selections.length === 0} onClick={() => void checkAvailability()}>{checking ? "Consultando…" : "Consultar disponibilidad"}</button>
              {availability && (
                <div className={`public-availability-result ${availability.available ? "available" : "attention"}`}>
                  <strong>{availability.available ? "Disponibilidad preliminar confirmada" : "Una o más cantidades requieren revisión"}</strong>
                  <p>{availability.available ? "Los componentes aparecen disponibles para el período indicado." : "Puedes enviar la solicitud; la empresa revisará alternativas antes de cotizar."}</p>
                  <div>{availability.items.map((item) => <span key={item.resource_name} className={item.can_fulfill ? "" : "missing"}>{item.can_fulfill ? "✓" : "!"} {item.requested_quantity} × {item.resource_name}</span>)}</div>
                </div>
              )}
            </section>

            <section className="public-request-panel">
              <div className="public-request-panel-heading"><span>3</span><div><h2>Datos de contacto</h2><p>La empresa utilizará estos datos para responder a tu solicitud.</p></div></div>
              <div className="public-form-grid two">
                <label><span>Nombre *</span><input value={form.first_name} onChange={(event) => updateField("first_name", event.target.value)} />{fields.first_name && <small>{fields.first_name}</small>}</label>
                <label><span>Apellido</span><input value={form.last_name} onChange={(event) => updateField("last_name", event.target.value)} />{fields.last_name && <small>{fields.last_name}</small>}</label>
                <label><span>Correo</span><input type="email" value={form.email} onChange={(event) => updateField("email", event.target.value)} />{fields.email && <small>{fields.email}</small>}</label>
                <label><span>Teléfono</span><input value={form.phone} onChange={(event) => updateField("phone", event.target.value)} />{fields.phone && <small>{fields.phone}</small>}</label>
                <label className="span-two"><span>Empresa u organización</span><input value={form.company_name} onChange={(event) => updateField("company_name", event.target.value)} />{fields.company_name && <small>{fields.company_name}</small>}</label>
                <label className="span-two"><span>Notas</span><textarea rows={5} value={form.notes} onChange={(event) => updateField("notes", event.target.value)} placeholder="Agrega requerimientos, horarios de montaje, transporte u otros detalles." />{fields.notes && <small>{fields.notes}</small>}</label>
              </div>
              {fields.contact && <small className="public-field-error">{fields.contact}</small>}
              <div className="public-consent-copy"><small>Aviso de contacto · versión {catalog.settings.terms_version || "vigente"}</small><p>{catalog.settings.terms_text}</p></div>
              <label className="public-consent"><input type="checkbox" checked={form.consent_accepted} onChange={(event) => updateField("consent_accepted", event.target.checked)} /><span>Acepto que {catalog.tenant.name} utilice estos datos para responder a mi solicitud.</span></label>
              {fields.consent_accepted && <small className="public-field-error">{fields.consent_accepted}</small>}
              <label className="public-honeypot" aria-hidden="true"><span>Sitio web</span><input tabIndex={-1} autoComplete="off" value={form.website} onChange={(event) => updateField("website", event.target.value)} /></label>
            </section>
          </div>

          <aside className="public-request-summary">
            <p className="public-kicker">RESUMEN</p>
            <h2>Tu solicitud</h2>
            {selectedPackages.length === 0 ? <p className="public-request-summary-empty">Selecciona al menos un paquete para ver el resumen.</p> : <div className="public-request-summary-lines">{selectedPackages.map((item) => <div key={item.slug}><span>{quantities[item.slug]} × {item.name}</span>{item.effective_price !== undefined && <strong>{formatCurrency(item.effective_price * quantities[item.slug], catalog.tenant.currency)}</strong>}</div>)}</div>}
            {catalog.settings.show_prices && <div className="public-request-total"><span>Estimado actual</span><strong>{formatCurrency(estimatedTotal, catalog.tenant.currency)}</strong></div>}
            <div className="public-request-period"><small>Período</small><span>{form.start_at ? formatDateTime(toRFC3339(form.start_at)) : "Sin definir"}</span><span>{form.end_at ? `a ${formatDateTime(toRFC3339(form.end_at))}` : ""}</span></div>
            <button className="public-button primary wide" type="submit" disabled={submitting || selections.length === 0}>{submitting ? "Enviando…" : "Enviar solicitud"}</button>
            <small className="public-request-disclaimer">Esta solicitud no bloquea inventario ni constituye una reserva. La empresa enviará una cotización formal.</small>
          </aside>
        </form>
      )}
    </PublicCatalogShell>
  );
}
