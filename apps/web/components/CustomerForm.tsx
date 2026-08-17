"use client";

import { FormEvent, useState } from "react";
import { ApiError, api } from "@/lib/api";
import type { Customer, CustomerSource } from "@/lib/types";

type Props = {
  initial?: Customer;
  onSaved: (customer: Customer) => void;
  onCancel: () => void;
};

export function CustomerForm({ initial, onSaved, onCancel }: Props) {
  const [form, setForm] = useState({
    first_name: initial?.first_name || "",
    last_name: initial?.last_name || "",
    phone: initial?.phone || "",
    email: initial?.email || "",
    company_name: initial?.company_name || "",
    tax_id: initial?.tax_id || "",
    tax_registration_number: initial?.tax_registration_number || "",
    billing_address: initial?.billing_address || "",
    document_type_code: initial?.document_type_code || "36",
    trade_name: initial?.trade_name || "",
    economic_activity: initial?.economic_activity || "",
    economic_activity_code: initial?.economic_activity_code || "",
    department_code: initial?.department_code || "",
    municipality_code: initial?.municipality_code || "",
    district_code: initial?.district_code || "",
    preferred_language: initial?.preferred_language || "es",
    source: initial?.source || ("MANUAL" as CustomerSource),
    notes: initial?.notes || "",
  });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setFields({});
    setMessage("");
    try {
      const customer = await api<Customer>(
        initial ? `/api/v1/customers/${initial.id}` : "/api/v1/customers",
        {
          method: initial ? "PATCH" : "POST",
          body: JSON.stringify({
            first_name: form.first_name,
            last_name: form.last_name,
            phone: form.phone || null,
            email: form.email || null,
            company_name: form.company_name || null,
            tax_id: form.tax_id,
            tax_registration_number: form.tax_registration_number,
            billing_address: form.billing_address,
            document_type_code: form.document_type_code,
            trade_name: form.trade_name,
            economic_activity: form.economic_activity,
            economic_activity_code: form.economic_activity_code,
            department_code: form.department_code,
            municipality_code: form.municipality_code,
            district_code: form.district_code,
            preferred_language: form.preferred_language,
            source: form.source,
            notes: form.notes,
          }),
        },
      );
      onSaved(customer);
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible guardar el cliente.");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={submit} className="form-stack">
      {message && <div className="form-alert">{message}</div>}

      <div className="form-grid two-columns">
        <label className="field">
          <span>Nombre *</span>
          <input
            value={form.first_name}
            onChange={(event) => setForm({ ...form, first_name: event.target.value })}
            placeholder="Carlos"
            autoFocus
          />
          {fields.first_name && <small className="field-error">{fields.first_name}</small>}
        </label>
        <label className="field">
          <span>Apellido</span>
          <input
            value={form.last_name}
            onChange={(event) => setForm({ ...form, last_name: event.target.value })}
            placeholder="Hernández"
          />
          {fields.last_name && <small className="field-error">{fields.last_name}</small>}
        </label>
      </div>

      <div className="form-grid two-columns">
        <label className="field">
          <span>WhatsApp / teléfono</span>
          <input
            value={form.phone}
            onChange={(event) => setForm({ ...form, phone: event.target.value })}
            placeholder="+50371234567"
            inputMode="tel"
          />
          {fields.phone && <small className="field-error">{fields.phone}</small>}
        </label>
        <label className="field">
          <span>Correo</span>
          <input
            type="email"
            value={form.email}
            onChange={(event) => setForm({ ...form, email: event.target.value })}
            placeholder="cliente@empresa.com"
          />
          {fields.email && <small className="field-error">{fields.email}</small>}
        </label>
      </div>

      <label className="field">
        <span>Empresa</span>
        <input
          value={form.company_name}
          onChange={(event) => setForm({ ...form, company_name: event.target.value })}
          placeholder="Opcional"
        />
        {fields.company_name && <small className="field-error">{fields.company_name}</small>}
      </label>

      <div className="form-grid two-columns">
        <label className="field">
          <span>NIT / identificación fiscal</span>
          <input
            value={form.tax_id}
            onChange={(event) => setForm({ ...form, tax_id: event.target.value })}
            placeholder="Opcional"
          />
          {fields.tax_id && <small className="field-error">{fields.tax_id}</small>}
        </label>
        <label className="field">
          <span>NRC / registro fiscal</span>
          <input
            value={form.tax_registration_number}
            onChange={(event) => setForm({ ...form, tax_registration_number: event.target.value })}
            placeholder="Opcional"
          />
          {fields.tax_registration_number && <small className="field-error">{fields.tax_registration_number}</small>}
        </label>
      </div>

      <label className="field">
        <span>Dirección de facturación</span>
        <textarea
          rows={2}
          value={form.billing_address}
          onChange={(event) => setForm({ ...form, billing_address: event.target.value })}
          placeholder="Dirección que aparecerá en facturas internas"
        />
        {fields.billing_address && <small className="field-error">{fields.billing_address}</small>}
      </label>

      <section className="customer-dte-fields">
        <div className="form-section-heading"><strong>Perfil DTE del receptor</strong><small>Completa estos campos cuando el cliente recibirá CCF.</small></div>
        <div className="form-grid two-columns">
          <label className="field"><span>Tipo de documento</span><select value={form.document_type_code} onChange={(event) => setForm({ ...form, document_type_code: event.target.value })}><option value="36">NIT</option><option value="13">DUI</option><option value="37">Otro</option><option value="03">Pasaporte</option></select></label>
          <label className="field"><span>Nombre comercial</span><input value={form.trade_name} onChange={(event) => setForm({ ...form, trade_name: event.target.value })} placeholder="Opcional" /></label>
          <label className="field"><span>Actividad económica</span><input value={form.economic_activity} onChange={(event) => setForm({ ...form, economic_activity: event.target.value })} placeholder="Descripción registrada" /></label>
          <label className="field"><span>Código de actividad</span><input value={form.economic_activity_code} onChange={(event) => setForm({ ...form, economic_activity_code: event.target.value })} placeholder="Código DGII" /></label>
        </div>
        <div className="form-grid three-columns">
          <label className="field"><span>Código departamento</span><input value={form.department_code} onChange={(event) => setForm({ ...form, department_code: event.target.value })} placeholder="06" /></label>
          <label className="field"><span>Código municipio</span><input value={form.municipality_code} onChange={(event) => setForm({ ...form, municipality_code: event.target.value })} placeholder="14" /></label>
          <label className="field"><span>Código distrito</span><input value={form.district_code} onChange={(event) => setForm({ ...form, district_code: event.target.value })} placeholder="Opcional" /></label>
        </div>
        {fields.location_codes && <small className="field-error">{fields.location_codes}</small>}
      </section>

      <div className="form-grid two-columns">
        <label className="field">
          <span>Origen</span>
          <select
            value={form.source}
            onChange={(event) => setForm({ ...form, source: event.target.value as CustomerSource })}
          >
            <option value="MANUAL">Carga manual</option>
            <option value="WHATSAPP">WhatsApp</option>
            <option value="WEB">Sitio web</option>
            <option value="IMPORT">Importación</option>
          </select>
          {fields.source && <small className="field-error">{fields.source}</small>}
        </label>
        <label className="field">
          <span>Idioma preferido</span>
          <select
            value={form.preferred_language}
            onChange={(event) => setForm({ ...form, preferred_language: event.target.value as "es" | "en" })}
          >
            <option value="es">Español</option>
            <option value="en">English</option>
          </select>
          {fields.preferred_language && <small className="field-error">{fields.preferred_language}</small>}
        </label>
      </div>

      <label className="field">
        <span>Notas</span>
        <textarea
          rows={4}
          value={form.notes}
          onChange={(event) => setForm({ ...form, notes: event.target.value })}
          placeholder="Preferencias, referencias del cliente o contexto comercial."
        />
        {fields.notes && <small className="field-error">{fields.notes}</small>}
      </label>

      <footer className="form-actions">
        <button type="button" className="button button-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button type="submit" className="button button-primary" disabled={saving}>
          {saving ? "Guardando…" : initial ? "Guardar cambios" : "Crear cliente"}
        </button>
      </footer>
    </form>
  );
}
