"use client";

import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import type { Workspace } from "@/lib/types";

export default function OnboardingPage() {
  const router = useRouter();
  const { me, selectWorkspace } = useAuth();
  const [form, setForm] = useState({ name: "", slug: "", legal_name: "", email: me?.user.email || "", phone: "", country_code: "SV", timezone: "America/El_Salvador", currency: "USD", address: "" });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function update(name: string, value: string) { setForm((current) => ({ ...current, [name]: value })); }
  async function submit(event: FormEvent) {
    event.preventDefault(); setSubmitting(true); setError(""); setFields({});
    try {
      const workspace = await api<Workspace>("/api/v1/organizations", { method: "POST", body: JSON.stringify(form) });
      await selectWorkspace(workspace.tenant_id);
      router.replace("/");
    } catch (reason) {
      if (reason instanceof ApiError) { setError(reason.message); setFields(reason.fields || {}); }
      else setError("No fue posible crear la organización.");
    } finally { setSubmitting(false); }
  }

  return <main className="standalone-page"><header className="standalone-header"><div className="auth-brand dark"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Onboarding</small></span></div></header><section className="onboarding-card panel"><div className="page-heading"><div><p className="eyebrow">PRIMER WORKSPACE</p><h1>Configura tu empresa</h1><p>Tu usuario será Owner y los datos quedarán aislados dentro de este workspace.</p></div></div>{error && <div className="form-alert">{error}</div>}<form className="settings-form" onSubmit={submit}><div className="form-grid two"><label className="field"><span>Nombre comercial</span><input value={form.name} onChange={(event) => update("name", event.target.value)} required />{fields.name && <small className="field-error">{fields.name}</small>}</label><label className="field"><span>Slug</span><input value={form.slug} onChange={(event) => update("slug", event.target.value)} placeholder="Se genera desde el nombre" />{fields.slug && <small className="field-error">{fields.slug}</small>}</label><label className="field"><span>Razón social</span><input value={form.legal_name} onChange={(event) => update("legal_name", event.target.value)} /></label><label className="field"><span>Correo comercial</span><input type="email" value={form.email} onChange={(event) => update("email", event.target.value)} /></label><label className="field"><span>Teléfono</span><input value={form.phone} onChange={(event) => update("phone", event.target.value)} /></label><label className="field"><span>Dirección</span><input value={form.address} onChange={(event) => update("address", event.target.value)} /></label><label className="field"><span>País</span><input value={form.country_code} maxLength={2} onChange={(event) => update("country_code", event.target.value)} /></label><label className="field"><span>Moneda</span><input value={form.currency} maxLength={3} onChange={(event) => update("currency", event.target.value)} /></label><label className="field form-span-two"><span>Zona horaria</span><input value={form.timezone} onChange={(event) => update("timezone", event.target.value)} /></label></div><div className="form-actions"><button className="button button-primary" disabled={submitting}>{submitting ? "Creando workspace…" : "Crear workspace"}</button></div></form></section></main>;
}
