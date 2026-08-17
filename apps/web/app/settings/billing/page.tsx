"use client";

import { FormEvent, useEffect, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import type { BillingSettings, TaxRule } from "@/lib/types";

const emptySettings: BillingSettings = {
  tenant_id: "",
  enabled: true,
  legal_name: "",
  trade_name: "",
  tax_id: "",
  tax_registration_number: "",
  economic_activity: "",
  economic_activity_code: "",
  fiscal_address: "",
  department: "",
  municipality: "",
  district: "",
  department_code: "",
  municipality_code: "",
  district_code: "",
  email: "",
  phone: "",
  prices_include_tax: false,
  default_tax_rate: 13,
  default_payment_terms_days: 0,
  invoice_prefix: "INV",
  next_invoice_number: 1,
  fiscal_profile_complete: false,
  fiscal_profile_missing_fields: [],
  created_at: "",
  updated_at: "",
};

export default function BillingSettingsPage() {
  const { can } = useAuth();
  const canManage = can("billing.manage");
  const [form, setForm] = useState<BillingSettings>(emptySettings);
  const [rules, setRules] = useState<TaxRule[]>([]);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.all([
      api<BillingSettings>("/api/v1/billing/settings"),
      api<{ items: TaxRule[] }>("/api/v1/billing/tax-rules"),
    ])
      .then(([settings, taxRules]) => {
        setForm(settings);
        setRules(taxRules.items);
      })
      .catch((reason) => setMessage(reason instanceof Error ? reason.message : "No fue posible cargar la configuración."))
      .finally(() => setLoading(false));
  }, []);

  function patch<K extends keyof BillingSettings>(key: K, value: BillingSettings[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!canManage) return;
    setSaving(true);
    setFields({});
    setMessage("");
    try {
      const updated = await api<BillingSettings>("/api/v1/billing/settings", {
        method: "PATCH",
        body: JSON.stringify({
          enabled: form.enabled,
          legal_name: form.legal_name,
          trade_name: form.trade_name,
          tax_id: form.tax_id,
          tax_registration_number: form.tax_registration_number,
          economic_activity: form.economic_activity,
          economic_activity_code: form.economic_activity_code,
          fiscal_address: form.fiscal_address,
          department: form.department,
          municipality: form.municipality,
          district: form.district,
          department_code: form.department_code,
          municipality_code: form.municipality_code,
          district_code: form.district_code,
          email: form.email,
          phone: form.phone,
          prices_include_tax: form.prices_include_tax,
          default_tax_rate: Number(form.default_tax_rate),
          default_payment_terms_days: Number(form.default_payment_terms_days),
          invoice_prefix: form.invoice_prefix,
        }),
      });
      setForm(updated);
      const taxRules = await api<{ items: TaxRule[] }>("/api/v1/billing/tax-rules");
      setRules(taxRules.items);
      setMessage("Configuración guardada correctamente.");
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible guardar la configuración.");
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;

  return (
    <div className="page-stack billing-settings-page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">BILLING PROFILE</p>
          <h2>Configuración de facturación</h2>
          <p>Define el perfil fiscal, la forma de calcular IVA y la numeración de documentos internos por empresa.</p>
        </div>
        <span className={`billing-profile-chip ${form.fiscal_profile_complete ? "complete" : "incomplete"}`}>
          {form.fiscal_profile_complete ? "Perfil fiscal completo" : "Perfil fiscal incompleto"}
        </span>
      </section>

      {message && <div className={message.includes("correctamente") ? "success-banner" : "form-alert"}>{message}</div>}

      {!form.fiscal_profile_complete && (
        <section className="panel billing-warning-panel">
          <strong>Faltan datos para preparar documentos electrónicos</strong>
          <p>Completa: {form.fiscal_profile_missing_fields.join(", ") || "datos fiscales"}. Las facturas internas siguen disponibles, pero no podrán convertirse en DTE.</p>
        </section>
      )}

      <form className="page-stack" onSubmit={submit}>
        <section className="panel billing-settings-panel">
          <div className="panel-header">
            <div><p className="eyebrow">CONTRIBUYENTE</p><h2>Identidad fiscal</h2><p>Estos valores se guardan como snapshot al crear y emitir cada factura.</p></div>
            <label className="toggle-row"><input type="checkbox" checked={form.enabled} disabled={!canManage} onChange={(event) => patch("enabled", event.target.checked)} /><span>Billing habilitado</span></label>
          </div>
          <div className="form-grid two-columns">
            <label className="field"><span>Razón social</span><input value={form.legal_name} disabled={!canManage} onChange={(event) => patch("legal_name", event.target.value)} />{fields.legal_name && <small className="field-error">{fields.legal_name}</small>}</label>
            <label className="field"><span>Nombre comercial</span><input value={form.trade_name} disabled={!canManage} onChange={(event) => patch("trade_name", event.target.value)} /></label>
            <label className="field"><span>NIT</span><input value={form.tax_id} disabled={!canManage} onChange={(event) => patch("tax_id", event.target.value)} placeholder="Identificación tributaria" /></label>
            <label className="field"><span>NRC</span><input value={form.tax_registration_number} disabled={!canManage} onChange={(event) => patch("tax_registration_number", event.target.value)} placeholder="Número de registro" /></label>
            <label className="field"><span>Actividad económica</span><input value={form.economic_activity} disabled={!canManage} onChange={(event) => patch("economic_activity", event.target.value)} /></label>
            <label className="field"><span>Código de actividad</span><input value={form.economic_activity_code} disabled={!canManage} onChange={(event) => patch("economic_activity_code", event.target.value)} /></label>
          </div>
          <label className="field"><span>Dirección fiscal</span><textarea rows={3} value={form.fiscal_address} disabled={!canManage} onChange={(event) => patch("fiscal_address", event.target.value)} /></label>
          <div className="form-grid three-columns">
            <label className="field"><span>Departamento</span><input value={form.department} disabled={!canManage} onChange={(event) => patch("department", event.target.value)} /></label>
            <label className="field"><span>Municipio</span><input value={form.municipality} disabled={!canManage} onChange={(event) => patch("municipality", event.target.value)} /></label>
            <label className="field"><span>Distrito</span><input value={form.district} disabled={!canManage} onChange={(event) => patch("district", event.target.value)} /></label>
          </div>
          <div className="form-grid three-columns billing-location-code-grid">
            <label className="field"><span>Código departamento</span><input value={form.department_code} disabled={!canManage} onChange={(event) => patch("department_code", event.target.value)} placeholder="Ej. 06" />{fields.department_code && <small className="field-error">{fields.department_code}</small>}</label>
            <label className="field"><span>Código municipio</span><input value={form.municipality_code} disabled={!canManage} onChange={(event) => patch("municipality_code", event.target.value)} placeholder="Código vigente" />{fields.municipality_code && <small className="field-error">{fields.municipality_code}</small>}</label>
            <label className="field"><span>Código distrito</span><input value={form.district_code} disabled={!canManage} onChange={(event) => patch("district_code", event.target.value)} placeholder="Opcional" />{fields.district_code && <small className="field-error">{fields.district_code}</small>}</label>
          </div>
          <div className="form-grid two-columns">
            <label className="field"><span>Correo</span><input type="email" value={form.email} disabled={!canManage} onChange={(event) => patch("email", event.target.value)} />{fields.email && <small className="field-error">{fields.email}</small>}</label>
            <label className="field"><span>Teléfono</span><input value={form.phone} disabled={!canManage} onChange={(event) => patch("phone", event.target.value)} /></label>
          </div>
        </section>

        <section className="panel billing-settings-panel">
          <div className="panel-header"><div><p className="eyebrow">INVOICE ENGINE</p><h2>Impuestos y numeración</h2></div></div>
          <div className="form-grid three-columns">
            <label className="field"><span>Tasa IVA predeterminada (%)</span><input type="number" min="0" max="100" step="0.01" value={form.default_tax_rate} disabled={!canManage} onChange={(event) => patch("default_tax_rate", Number(event.target.value))} />{fields.default_tax_rate && <small className="field-error">{fields.default_tax_rate}</small>}</label>
            <label className="field"><span>Días de crédito</span><input type="number" min="0" max="365" value={form.default_payment_terms_days} disabled={!canManage} onChange={(event) => patch("default_payment_terms_days", Number(event.target.value))} /></label>
            <label className="field"><span>Prefijo</span><input value={form.invoice_prefix} disabled={!canManage} onChange={(event) => patch("invoice_prefix", event.target.value.toUpperCase())} />{fields.invoice_prefix && <small className="field-error">{fields.invoice_prefix}</small>}</label>
          </div>
          <div className="billing-number-preview"><span>Siguiente documento</span><strong>{form.invoice_prefix}-{String(form.next_invoice_number).padStart(6, "0")}</strong></div>
          <label className="toggle-row billing-tax-toggle"><input type="checkbox" checked={form.prices_include_tax} disabled={!canManage} onChange={(event) => patch("prices_include_tax", event.target.checked)} /><span>Los precios ingresados ya incluyen IVA</span></label>
          <div className="billing-tax-rule-grid">
            {rules.map((rule) => <article key={rule.id}><span>{rule.code}</span><strong>{rule.name}</strong><small>{rule.category === "TAXABLE" ? `${rule.rate}%` : "0%"}</small></article>)}
          </div>
        </section>

        {canManage && <footer className="form-actions billing-settings-actions"><button className="button button-primary" type="submit" disabled={saving}>{saving ? "Guardando…" : "Guardar configuración"}</button></footer>}
      </form>

      <section className="architecture-note">
        <span>i</span><div><strong>Billing y DTE permanecen separados</strong><p>Esta pantalla define el perfil tributario y los impuestos internos. La configuración del proveedor, ambientes, firma y transmisión se administra en Integración DTE.</p></div>
      </section>
    </div>
  );
}
