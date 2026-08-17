"use client";

import { FormEvent, useEffect, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { ApiError, api } from "@/lib/api";
import type { QuotePortalSettings } from "@/lib/types";

type Draft = {
  enabled: boolean;
  headline: string;
  introduction: string;
  accent_color: string;
  default_validity_days: number;
  allow_rejection: boolean;
  require_response_name: boolean;
  acceptance_terms_text: string;
  acceptance_terms_version: string;
};

function draftOf(item: QuotePortalSettings): Draft {
  return {
    enabled: item.enabled,
    headline: item.headline,
    introduction: item.introduction,
    accent_color: item.accent_color,
    default_validity_days: item.default_validity_days,
    allow_rejection: item.allow_rejection,
    require_response_name: item.require_response_name,
    acceptance_terms_text: item.acceptance_terms_text,
    acceptance_terms_version: item.acceptance_terms_version,
  };
}

export default function QuotePortalSettingsPage() {
  const { can } = useAuth();
  const canManage = can("quote.manage");
  const [form, setForm] = useState<Draft | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});

  useEffect(() => {
    api<QuotePortalSettings>("/api/v1/quote-portal-settings")
      .then((item) => setForm(draftOf(item)))
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el portal de cotizaciones."))
      .finally(() => setLoading(false));
  }, []);

  function update<K extends keyof Draft>(key: K, value: Draft[K]) {
    setForm((current) => (current ? { ...current, [key]: value } : current));
    setSaved(false);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!form || !canManage) return;
    setSaving(true);
    setSaved(false);
    setError("");
    setFields({});
    try {
      const item = await api<QuotePortalSettings>("/api/v1/quote-portal-settings", {
        method: "PATCH",
        body: JSON.stringify(form),
      });
      setForm(draftOf(item));
      setSaved(true);
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.message);
        setFields(reason.fields || {});
      } else {
        setError("No fue posible guardar la configuración del portal.");
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (!form) return <div className="panel inline-error">{error || "No fue posible cargar la configuración."}</div>;

  return (
    <div className="page-stack quote-portal-settings-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">QUOTE PORTAL</p>
          <h2>Aceptación online de cotizaciones</h2>
          <p>Configura la experiencia, vigencia y evidencia que verá el cliente al abrir un enlace seguro.</p>
        </div>
      </div>

      {error && <div className="form-alert">{error}</div>}
      {saved && <div className="success-banner">Configuración del portal guardada correctamente.</div>}

      <section className="quote-portal-settings-summary panel">
        <div>
          <span className={`quote-portal-live-dot ${form.enabled ? "enabled" : ""}`} />
          <div>
            <strong>{form.enabled ? "Portal habilitado" : "Portal deshabilitado"}</strong>
            <p>Al enviar una cotización se genera un enlace secreto de un solo tenant y el valor original del token nunca se guarda en PostgreSQL.</p>
          </div>
        </div>
        <span className="quote-portal-version-pill">Términos v{form.acceptance_terms_version || "—"}</span>
      </section>

      <form className="panel settings-form quote-portal-settings-form" onSubmit={submit}>
        <div className="panel-title-row">
          <div>
            <p className="eyebrow">EXPERIENCIA DEL CLIENTE</p>
            <h3>Documento público</h3>
            <p>Estos textos se copian como snapshot al generar cada enlace.</p>
          </div>
          <label className="switch-row prominent">
            <input type="checkbox" checked={form.enabled} disabled={!canManage} onChange={(event) => update("enabled", event.target.checked)} />
            <span />
            <strong>Habilitado</strong>
          </label>
        </div>

        <div className="form-grid two quote-portal-settings-grid">
          <label className="field">
            <span>Título</span>
            <input value={form.headline} disabled={!canManage} onChange={(event) => update("headline", event.target.value)} />
            {fields.headline && <small className="field-error">{fields.headline}</small>}
          </label>
          <label className="field">
            <span>Color principal</span>
            <div className="color-input-row">
              <input type="color" value={form.accent_color} disabled={!canManage} onChange={(event) => update("accent_color", event.target.value)} />
              <input value={form.accent_color} disabled={!canManage} onChange={(event) => update("accent_color", event.target.value)} />
            </div>
            {fields.accent_color && <small className="field-error">{fields.accent_color}</small>}
          </label>
          <label className="field form-span-two">
            <span>Introducción</span>
            <textarea rows={4} value={form.introduction} disabled={!canManage} onChange={(event) => update("introduction", event.target.value)} />
            {fields.introduction && <small className="field-error">{fields.introduction}</small>}
          </label>
          <label className="field">
            <span>Vigencia predeterminada</span>
            <input type="number" min={1} max={60} value={form.default_validity_days} disabled={!canManage} onChange={(event) => update("default_validity_days", Number(event.target.value) || 1)} />
            <small className="field-hint">Se usa cuando la cotización no tiene una expiración futura válida.</small>
            {fields.default_validity_days && <small className="field-error">{fields.default_validity_days}</small>}
          </label>
          <label className="field">
            <span>Versión de términos</span>
            <input value={form.acceptance_terms_version} disabled={!canManage} onChange={(event) => update("acceptance_terms_version", event.target.value)} />
            {fields.acceptance_terms_version && <small className="field-error">{fields.acceptance_terms_version}</small>}
          </label>
          <label className="field form-span-two">
            <span>Términos de aceptación</span>
            <textarea rows={7} value={form.acceptance_terms_text} disabled={!canManage} onChange={(event) => update("acceptance_terms_text", event.target.value)} />
            <small className="field-hint">El cliente debe aceptarlos explícitamente. El texto y la versión quedan congelados en el enlace.</small>
            {fields.acceptance_terms_text && <small className="field-error">{fields.acceptance_terms_text}</small>}
          </label>
        </div>

        <div className="public-admin-toggle-grid quote-portal-toggle-grid">
          <label className="switch-card">
            <input type="checkbox" checked={form.allow_rejection} disabled={!canManage} onChange={(event) => update("allow_rejection", event.target.checked)} />
            <span />
            <div><strong>Permitir rechazo</strong><small>El cliente puede rechazar y dejar un motivo opcional desde el mismo enlace.</small></div>
          </label>
          <label className="switch-card">
            <input type="checkbox" checked={form.require_response_name} disabled={!canManage} onChange={(event) => update("require_response_name", event.target.checked)} />
            <span />
            <div><strong>Solicitar nombre</strong><small>Registra quién aceptó o rechazó la cotización como parte de la evidencia.</small></div>
          </label>
          <article className="quote-portal-security-card">
            <span>SHA-256</span>
            <div><strong>Token no recuperable</strong><small>Perder el enlace requiere rotarlo; la base de datos conserva únicamente su hash.</small></div>
          </article>
        </div>

        {canManage && <div className="form-actions"><button className="button button-primary" disabled={saving}>{saving ? "Guardando…" : "Guardar configuración"}</button></div>}
      </form>
    </div>
  );
}
