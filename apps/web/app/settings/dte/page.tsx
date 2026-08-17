"use client";

import { FormEvent, useEffect, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import type { DTESettings } from "@/lib/types";

const emptySettings: DTESettings = {
  tenant_id: "",
  enabled: true,
  provider_mode: "MOCK",
  environment: "TEST",
  default_document_type: "01",
  schema_version: 1,
  establishment_type: "01",
  establishment_code: "M001",
  point_of_sale_code: "P001",
  auth_url: "",
  signer_url: "",
  reception_url: "",
  invalidation_url: "",
  query_url: "",
  user_secret_ref: "",
  password_secret_ref: "",
  signing_password_secret_ref: "",
  auto_submit_on_issue: false,
  max_attempts: 5,
  retry_base_seconds: 60,
  next_control_number: 1,
  configuration_ready: true,
  production_safety_ready: false,
  missing_configuration: [],
  created_at: "",
  updated_at: "",
};

export default function DTESettingsPage() {
  const { can } = useAuth();
  const canManage = can("fiscal.manage");
  const [form, setForm] = useState<DTESettings>(emptySettings);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [fields, setFields] = useState<Record<string, string>>({});

  useEffect(() => {
    api<DTESettings>("/api/v1/dte-settings")
      .then(setForm)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la configuración DTE."))
      .finally(() => setLoading(false));
  }, []);

  function patch<K extends keyof DTESettings>(key: K, value: DTESettings[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setSaved(false);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!canManage) return;
    setSaving(true);
    setError("");
    setSaved(false);
    setFields({});
    try {
      const updated = await api<DTESettings>("/api/v1/dte-settings", {
        method: "PATCH",
        body: JSON.stringify({
          enabled: form.enabled,
          provider_mode: form.provider_mode,
          environment: form.environment,
          default_document_type: form.default_document_type,
          schema_version: Number(form.schema_version),
          establishment_type: form.establishment_type,
          establishment_code: form.establishment_code,
          point_of_sale_code: form.point_of_sale_code,
          auth_url: form.auth_url,
          signer_url: form.signer_url,
          reception_url: form.reception_url,
          invalidation_url: form.invalidation_url,
          query_url: form.query_url,
          user_secret_ref: form.user_secret_ref,
          password_secret_ref: form.password_secret_ref,
          signing_password_secret_ref: form.signing_password_secret_ref,
          auto_submit_on_issue: false,
          max_attempts: Number(form.max_attempts),
          retry_base_seconds: Number(form.retry_base_seconds),
        }),
      });
      setForm(updated);
      setSaved(true);
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.message);
        setFields(reason.fields || {});
      } else {
        setError("No fue posible guardar la configuración DTE.");
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;

  return (
    <div className="page-stack dte-settings-page">
      <section className="page-heading">
        <div><p className="eyebrow">FISCAL PROVIDER</p><h2>Integración DTE</h2><p>Configura el ambiente, numeración, firma y transmisión por cada empresa.</p></div>
        <span className={`dte-readiness-chip ${form.configuration_ready ? "ready" : "incomplete"}`}>{form.configuration_ready ? "Configuración completa" : "Configuración incompleta"}</span>
      </section>

      {error && <div className="form-alert">{error}</div>}
      {saved && <div className="success-banner">Configuración DTE guardada correctamente.</div>}

      <section className={`panel dte-provider-banner ${form.provider_mode.toLowerCase()}`}>
        <div><span className={`dte-live-dot ${form.enabled ? "enabled" : ""}`} /><div><strong>{form.provider_mode === "MOCK" ? "Simulador local seguro" : "Adaptador MH_HTTP"}</strong><p>{form.provider_mode === "MOCK" ? "No hace llamadas externas ni produce validez fiscal." : "Requiere onboarding oficial, credenciales y endpoints entregados al contribuyente."}</p></div></div>
        <span className={`dte-environment-chip ${form.environment.toLowerCase()}`}>{form.environment}</span>
      </section>

      <form className="page-stack" onSubmit={submit}>
        <section className="panel dte-settings-panel">
          <div className="panel-header"><div><p className="eyebrow">MODE</p><h2>Proveedor y ambiente</h2><p>Comienza con MOCK. Cambia a MH_HTTP únicamente durante la homologación oficial.</p></div><label className="toggle-row"><input type="checkbox" checked={form.enabled} disabled={!canManage} onChange={(event) => patch("enabled", event.target.checked)} /><span>DTE habilitado</span></label></div>
          <div className="form-grid three-columns">
            <label className="field"><span>Proveedor</span><select value={form.provider_mode} disabled={!canManage} onChange={(event) => patch("provider_mode", event.target.value as DTESettings["provider_mode"])}><option value="MOCK">MOCK local</option><option value="MH_HTTP">MH_HTTP configurable</option></select>{fields.provider_mode && <small className="field-error">{fields.provider_mode}</small>}</label>
            <label className="field"><span>Ambiente</span><select value={form.environment} disabled={!canManage} onChange={(event) => patch("environment", event.target.value as DTESettings["environment"])}><option value="TEST">Pruebas</option><option value="PRODUCTION">Producción</option></select>{fields.environment && <small className="field-error">{fields.environment}</small>}</label>
            <label className="field"><span>Documento predeterminado</span><select value={form.default_document_type} disabled={!canManage} onChange={(event) => patch("default_document_type", event.target.value as "01" | "03")}><option value="01">01 · Factura</option><option value="03">03 · Crédito fiscal</option></select></label>
          </div>
          <div className="form-grid four-columns">
            <label className="field"><span>Versión de esquema</span><input type="number" min={1} max={99} value={form.schema_version} disabled={!canManage} onChange={(event) => patch("schema_version", Number(event.target.value))} /></label>
            <label className="field"><span>Tipo establecimiento</span><input value={form.establishment_type} disabled={!canManage} onChange={(event) => patch("establishment_type", event.target.value.toUpperCase())} /></label>
            <label className="field"><span>Código establecimiento</span><input value={form.establishment_code} disabled={!canManage} onChange={(event) => patch("establishment_code", event.target.value.toUpperCase())} />{fields.establishment_code && <small className="field-error">{fields.establishment_code}</small>}</label>
            <label className="field"><span>Punto de venta</span><input value={form.point_of_sale_code} disabled={!canManage} onChange={(event) => patch("point_of_sale_code", event.target.value.toUpperCase())} />{fields.point_of_sale_code && <small className="field-error">{fields.point_of_sale_code}</small>}</label>
          </div>
          <div className="dte-sequence-preview"><span>Siguiente secuencia interna</span><strong>{String(form.next_control_number).padStart(15, "0")}</strong><small>Se bloquea transaccionalmente al preparar cada documento.</small></div>
        </section>

        {form.provider_mode === "MH_HTTP" && (
          <>
            <section className="panel dte-settings-panel">
              <div className="panel-header"><div><p className="eyebrow">OFFICIAL SERVICES</p><h2>Endpoints entregados al emisor</h2><p>RentStage no incluye URLs hardcodeadas. Usa exactamente las del proceso oficial de pruebas o producción.</p></div></div>
              <div className="form-grid two-columns">
                <label className="field"><span>Autenticación</span><input value={form.auth_url} disabled={!canManage} onChange={(event) => patch("auth_url", event.target.value)} placeholder="https://…" />{fields.auth_url && <small className="field-error">{fields.auth_url}</small>}</label>
                <label className="field"><span>Firma</span><input value={form.signer_url} disabled={!canManage} onChange={(event) => patch("signer_url", event.target.value)} placeholder="https://…" />{fields.signer_url && <small className="field-error">{fields.signer_url}</small>}</label>
                <label className="field"><span>Recepción</span><input value={form.reception_url} disabled={!canManage} onChange={(event) => patch("reception_url", event.target.value)} placeholder="https://…" />{fields.reception_url && <small className="field-error">{fields.reception_url}</small>}</label>
                <label className="field"><span>Invalidación</span><input value={form.invalidation_url} disabled={!canManage} onChange={(event) => patch("invalidation_url", event.target.value)} placeholder="https://… (opcional)" /></label>
                <label className="field"><span>Consulta</span><input value={form.query_url} disabled={!canManage} onChange={(event) => patch("query_url", event.target.value)} placeholder="https://… (opcional)" /></label>
              </div>
            </section>

            <section className="panel dte-settings-panel">
              <div className="panel-header"><div><p className="eyebrow">SECRET BOUNDARY</p><h2>Referencias a secretos</h2><p>Guarda solo referencias. El valor real se resuelve en tiempo de ejecución desde variables seguras.</p></div></div>
              <div className="form-grid three-columns">
                <label className="field"><span>Usuario</span><input value={form.user_secret_ref} disabled={!canManage} onChange={(event) => patch("user_secret_ref", event.target.value)} placeholder="env://DTE_MH_USER" />{fields.user_secret_ref && <small className="field-error">{fields.user_secret_ref}</small>}</label>
                <label className="field"><span>Contraseña</span><input value={form.password_secret_ref} disabled={!canManage} onChange={(event) => patch("password_secret_ref", event.target.value)} placeholder="env://DTE_MH_PASSWORD" />{fields.password_secret_ref && <small className="field-error">{fields.password_secret_ref}</small>}</label>
                <label className="field"><span>Clave de firma</span><input value={form.signing_password_secret_ref} disabled={!canManage} onChange={(event) => patch("signing_password_secret_ref", event.target.value)} placeholder="env://DTE_MH_SIGNING_PASSWORD" />{fields.signing_password_secret_ref && <small className="field-error">{fields.signing_password_secret_ref}</small>}</label>
              </div>
              <div className="architecture-note compact"><span>!</span><div><strong>Nunca pegues credenciales reales</strong><p>La base de datos conserva las referencias `env://…`; el secreto se inyecta mediante el entorno o, en GCP, desde Secret Manager.</p></div></div>
            </section>
          </>
        )}

        <section className="panel dte-settings-panel">
          <div className="panel-header"><div><p className="eyebrow">RESILIENCE</p><h2>Reintentos y operación</h2></div></div>
          <div className="form-grid three-columns">
            <label className="field"><span>Máximo de intentos</span><input type="number" min={1} max={20} value={form.max_attempts} disabled={!canManage} onChange={(event) => patch("max_attempts", Number(event.target.value))} /></label>
            <label className="field"><span>Intervalo base (segundos)</span><input type="number" min={5} max={86400} value={form.retry_base_seconds} disabled={!canManage} onChange={(event) => patch("retry_base_seconds", Number(event.target.value))} /></label>
            <label className="toggle-row dte-auto-submit-disabled"><input type="checkbox" checked={false} disabled /><span>Envío automático al emitir <small>reservado; v0.12 exige confirmación manual</small></span></label>
          </div>
        </section>

        {canManage && <footer className="form-actions"><button className="button button-primary" disabled={saving}>{saving ? "Guardando…" : "Guardar configuración DTE"}</button></footer>}
      </form>

      <section className="architecture-note"><span>i</span><div><strong>Homologación obligatoria para producción</strong><p>El modo MOCK prueba el dominio local. MH_HTTP es un adaptador configurable y debe validarse con los esquemas, credenciales, certificados y casos oficiales habilitados para cada contribuyente antes de transmitir documentos reales.</p></div></section>
    </div>
  );
}
