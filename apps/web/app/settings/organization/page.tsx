"use client";

import { FormEvent, useEffect, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import type { Tenant, Workspace } from "@/lib/types";

export default function OrganizationPage() {
  const { refresh } = useAuth();
  const [form, setForm] = useState({ name: "", slug: "", legal_name: "", email: "", phone: "", country_code: "SV", timezone: "America/El_Salvador", currency: "USD", address: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const [fields, setFields] = useState<Record<string, string>>({});

  useEffect(() => { api<Tenant>("/api/v1/tenant").then((tenant) => setForm({ name: tenant.name, slug: tenant.slug, legal_name: tenant.legal_name || "", email: tenant.email || "", phone: tenant.phone || "", country_code: tenant.country_code, timezone: tenant.timezone, currency: tenant.currency, address: tenant.address || "" })).catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la organización.")).finally(() => setLoading(false)); }, []);
  function update(name: string, value: string) { setForm((current) => ({ ...current, [name]: value })); setSaved(false); }
  async function submit(event: FormEvent) { event.preventDefault(); setSaving(true); setError(""); setFields({}); try { await api<Workspace>("/api/v1/tenant", { method: "PATCH", body: JSON.stringify(form) }); await refresh(); setSaved(true); } catch (reason) { if (reason instanceof ApiError) { setError(reason.message); setFields(reason.fields || {}); } else setError("No fue posible guardar la organización."); } finally { setSaving(false); } }

  if (loading) return <div className="skeleton detail-skeleton" />;
  return <div className="page-stack"><div className="page-heading"><div><p className="eyebrow">WORKSPACE</p><h2>Configuración de organización</h2><p>Estos datos identifican la empresa activa y se mantienen separados de otros tenants.</p></div></div>{error && <div className="form-alert">{error}</div>}{saved && <div className="success-banner">Cambios guardados correctamente.</div>}<form className="panel settings-form" onSubmit={submit}><div className="form-grid two"><label className="field"><span>Nombre comercial</span><input value={form.name} onChange={(event) => update("name", event.target.value)} required />{fields.name && <small className="field-error">{fields.name}</small>}</label><label className="field"><span>Slug</span><input value={form.slug} onChange={(event) => update("slug", event.target.value)} required />{fields.slug && <small className="field-error">{fields.slug}</small>}</label><label className="field"><span>Razón social</span><input value={form.legal_name} onChange={(event) => update("legal_name", event.target.value)} /></label><label className="field"><span>Correo</span><input type="email" value={form.email} onChange={(event) => update("email", event.target.value)} /></label><label className="field"><span>Teléfono</span><input value={form.phone} onChange={(event) => update("phone", event.target.value)} /></label><label className="field"><span>Dirección</span><input value={form.address} onChange={(event) => update("address", event.target.value)} /></label><label className="field"><span>País</span><input maxLength={2} value={form.country_code} onChange={(event) => update("country_code", event.target.value)} /></label><label className="field"><span>Moneda</span><input maxLength={3} value={form.currency} onChange={(event) => update("currency", event.target.value)} /></label><label className="field form-span-two"><span>Zona horaria</span><input value={form.timezone} onChange={(event) => update("timezone", event.target.value)} /></label></div><div className="form-actions"><button className="button button-primary" disabled={saving}>{saving ? "Guardando…" : "Guardar cambios"}</button></div></form></div>;
}
