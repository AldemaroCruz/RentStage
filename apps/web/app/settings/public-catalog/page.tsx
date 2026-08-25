"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type {
  AdminPublicCatalog,
  AdminPublicPackage,
  AdminPublicResource,
  PublicCatalogSettings,
} from "@/lib/types";

type SettingsDraft = {
  enabled: boolean;
  headline: string;
  description: string;
  cover_image_url: string;
  accent_color: string;
  show_prices: boolean;
  show_resources: boolean;
  quote_requests_enabled: boolean;
  web_chat_enabled: boolean;
  contact_email: string;
  contact_phone: string;
  contact_address: string;
  terms_text: string;
  terms_version: string;
};

function settingsDraft(item: PublicCatalogSettings): SettingsDraft {
  return {
    enabled: item.enabled,
    headline: item.headline,
    description: item.description,
    cover_image_url: item.cover_image_url || "",
    accent_color: item.accent_color || "#6558e8",
    show_prices: item.show_prices,
    show_resources: item.show_resources,
    quote_requests_enabled: item.quote_requests_enabled,
    web_chat_enabled: item.web_chat_enabled,
    contact_email: item.contact_email || "",
    contact_phone: item.contact_phone || "",
    contact_address: item.contact_address || "",
    terms_text: item.terms_text,
    terms_version: item.terms_version,
  };
}

export default function PublicCatalogSettingsPage() {
  const { can } = useAuth();
  const canManage = can("public_catalog.manage");
  const [catalog, setCatalog] = useState<AdminPublicCatalog | null>(null);
  const [form, setForm] = useState<SettingsDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState("");
  const [error, setError] = useState("");
  const [saved, setSaved] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [resourceFilter, setResourceFilter] = useState("");
  const [resourceMode, setResourceMode] = useState<"ALL" | "PUBLISHED" | "HIDDEN">("ALL");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<AdminPublicCatalog>("/api/v1/public-catalog")
      .then((result) => { setCatalog(result); setForm(settingsDraft(result.settings)); })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el catálogo público."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const publishedPackages = useMemo(() => catalog?.packages.filter((item) => item.public_visible).length || 0, [catalog]);
  const publishedResources = useMemo(() => catalog?.resources.filter((item) => item.public_visible).length || 0, [catalog]);
  const filteredResources = useMemo(() => {
    const query = resourceFilter.trim().toLowerCase();
    return (catalog?.resources || []).filter((item) => {
      if (resourceMode === "PUBLISHED" && !item.public_visible) return false;
      if (resourceMode === "HIDDEN" && item.public_visible) return false;
      return !query || item.name.toLowerCase().includes(query) || (item.category_name || "").toLowerCase().includes(query) || (item.public_slug || "").includes(query);
    });
  }, [catalog, resourceFilter, resourceMode]);

  function updateSettings<K extends keyof SettingsDraft>(name: K, value: SettingsDraft[K]) {
    setForm((current) => current ? { ...current, [name]: value } : current);
    setSaved("");
    setFields((current) => {
      if (!current[name]) return current;
      const next = { ...current };
      delete next[name];
      return next;
    });
  }

  function updatePackage(id: string, patch: Partial<AdminPublicPackage>) {
    setCatalog((current) => current ? { ...current, packages: current.packages.map((item) => item.id === id ? { ...item, ...patch } : item) } : current);
    setSaved("");
  }

  function updateResource(id: string, patch: Partial<AdminPublicResource>) {
    setCatalog((current) => current ? { ...current, resources: current.resources.map((item) => item.id === id ? { ...item, ...patch } : item) } : current);
    setSaved("");
  }

  async function saveSettings(event: FormEvent) {
    event.preventDefault();
    if (!form || !catalog) return;
    setSaving("settings");
    setError("");
    setFields({});
    try {
      const result = await api<PublicCatalogSettings>("/api/v1/public-catalog", {
        method: "PATCH",
        body: JSON.stringify({
          ...form,
          cover_image_url: form.cover_image_url.trim() || null,
          contact_email: form.contact_email.trim() || null,
          contact_phone: form.contact_phone.trim() || null,
          contact_address: form.contact_address.trim() || null,
        }),
      });
      setCatalog({ ...catalog, settings: result });
      setForm(settingsDraft(result));
      setSaved("Configuración pública guardada.");
    } catch (reason) {
      if (reason instanceof ApiError) { setError(reason.message); setFields(reason.fields || {}); }
      else setError("No fue posible guardar la configuración.");
    } finally {
      setSaving("");
    }
  }

  async function savePackage(item: AdminPublicPackage) {
    setSaving(`package:${item.id}`);
    setError("");
    setFields({});
    try {
      const result = await api<AdminPublicCatalog>(`/api/v1/public-catalog/packages/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify({ visible: item.public_visible, featured: item.public_featured, sort_order: item.public_sort_order }),
      });
      setCatalog(result);
      setSaved(`Publicación de “${item.name}” actualizada.`);
    } catch (reason) {
      if (reason instanceof ApiError) { setError(reason.message); setFields(reason.fields || {}); }
      else setError("No fue posible actualizar el paquete.");
    } finally {
      setSaving("");
    }
  }

  async function saveResource(item: AdminPublicResource) {
    setSaving(`resource:${item.id}`);
    setError("");
    setFields({});
    try {
      const result = await api<AdminPublicCatalog>(`/api/v1/public-catalog/resources/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          public_slug: item.public_slug || "",
          public_description: item.public_description,
          public_image_url: item.public_image_url || null,
          visible: item.public_visible,
          featured: item.public_featured,
          sort_order: item.public_sort_order,
        }),
      });
      setCatalog(result);
      setSaved(`Publicación de “${item.name}” actualizada.`);
    } catch (reason) {
      if (reason instanceof ApiError) { setError(reason.message); setFields(reason.fields || {}); }
      else setError("No fue posible actualizar el recurso.");
    } finally {
      setSaving("");
    }
  }

  async function copyPublicURL() {
    if (!catalog) return;
    try {
      await navigator.clipboard.writeText(catalog.public_url);
      setSaved("URL pública copiada al portapapeles.");
    } catch {
      setError("No fue posible copiar la URL. Puedes seleccionarla manualmente.");
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error && !catalog) return <section className="panel connection-panel"><span className="connection-icon">!</span><div><h2>Catálogo no disponible</h2><p>{error}</p><button className="text-link" type="button" onClick={load}>Reintentar</button></div></section>;
  if (!catalog || !form) return null;

  return (
    <div className="page-stack public-admin-page">
      <div className="page-heading public-admin-heading">
        <div><p className="eyebrow">PUBLIC CATALOG</p><h2>Catálogo público</h2><p>Decide qué verá el cliente, publica paquetes y recibe solicitudes desde una página propia para este workspace.</p></div>
        <div className="public-admin-header-actions">
          <button className="button button-secondary" type="button" onClick={() => void copyPublicURL()}>Copiar URL</button>
          <a className="button button-primary" href={catalog.public_url} target="_blank" rel="noreferrer">Abrir catálogo ↗</a>
        </div>
      </div>

      {error && <div className="form-alert">{error}</div>}
      {saved && <div className="success-banner">{saved}</div>}
      {!canManage && <div className="info-callout"><strong>Vista de solo lectura</strong><span>Tu rol permite revisar la configuración, pero no modificar publicaciones.</span></div>}

      <section className={`panel public-admin-status ${catalog.settings.enabled ? "online" : "offline"}`}>
        <div className="public-admin-status-indicator"><span /><div><p>{catalog.settings.enabled ? "CATÁLOGO ACTIVO" : "CATÁLOGO DESACTIVADO"}</p><strong>{catalog.tenant.name}</strong></div></div>
        <div className="public-admin-url"><small>Página pública</small><code>{catalog.public_url}</code></div>
        <div className="public-admin-status-stats"><div><strong>{publishedPackages}</strong><span>paquetes</span></div><div><strong>{publishedResources}</strong><span>recursos</span></div><div><strong>{catalog.settings.quote_requests_enabled ? "Sí" : "No"}</strong><span>solicitudes web</span></div></div>
      </section>

      <form className="panel public-admin-settings" onSubmit={saveSettings}>
        <div className="panel-title-row"><div><p className="eyebrow">PRESENTACIÓN</p><h3>Configuración del escaparate</h3><p>Controla el contenido, contacto y comportamiento general de la página pública.</p></div><label className="switch-row prominent"><input type="checkbox" checked={form.enabled} disabled={!canManage} onChange={(event) => updateSettings("enabled", event.target.checked)} /><span /><strong>{form.enabled ? "Publicado" : "Desactivado"}</strong></label></div>
        <div className="form-grid two public-admin-form-grid">
          <label className="field"><span>Titular principal</span><input value={form.headline} disabled={!canManage} onChange={(event) => updateSettings("headline", event.target.value)} />{fields.headline && <small className="field-error">{fields.headline}</small>}</label>
          <label className="field"><span>Color de acento</span><div className="color-input-row"><input type="color" value={form.accent_color} disabled={!canManage} onChange={(event) => updateSettings("accent_color", event.target.value)} /><input value={form.accent_color} disabled={!canManage} onChange={(event) => updateSettings("accent_color", event.target.value)} /></div>{fields.accent_color && <small className="field-error">{fields.accent_color}</small>}</label>
          <label className="field form-span-two"><span>Descripción</span><textarea rows={4} value={form.description} disabled={!canManage} onChange={(event) => updateSettings("description", event.target.value)} />{fields.description && <small className="field-error">{fields.description}</small>}</label>
          <label className="field form-span-two"><span>URL de imagen de portada</span><input value={form.cover_image_url} disabled={!canManage} onChange={(event) => updateSettings("cover_image_url", event.target.value)} placeholder="https://…" />{fields.cover_image_url && <small className="field-error">{fields.cover_image_url}</small>}</label>
          <label className="field"><span>Correo público</span><input type="email" value={form.contact_email} disabled={!canManage} onChange={(event) => updateSettings("contact_email", event.target.value)} /></label>
          <label className="field"><span>Teléfono público</span><input value={form.contact_phone} disabled={!canManage} onChange={(event) => updateSettings("contact_phone", event.target.value)} /></label>
          <label className="field form-span-two"><span>Dirección pública</span><input value={form.contact_address} disabled={!canManage} onChange={(event) => updateSettings("contact_address", event.target.value)} /></label>
          <label className="field"><span>Versión de términos</span><input value={form.terms_version} disabled={!canManage} onChange={(event) => updateSettings("terms_version", event.target.value)} />{fields.terms_version && <small className="field-error">{fields.terms_version}</small>}</label>
          <label className="field form-span-two"><span>Aviso de contacto y privacidad</span><textarea rows={4} value={form.terms_text} disabled={!canManage} onChange={(event) => updateSettings("terms_text", event.target.value)} />{fields.terms_text && <small className="field-error">{fields.terms_text}</small>}</label>
        </div>
        <div className="public-admin-toggle-grid">
          <label className="switch-card"><input type="checkbox" checked={form.show_prices} disabled={!canManage} onChange={(event) => updateSettings("show_prices", event.target.checked)} /><span /><div><strong>Mostrar precios</strong><small>Incluye precios base y estimados en el catálogo.</small></div></label>
          <label className="switch-card"><input type="checkbox" checked={form.show_resources} disabled={!canManage} onChange={(event) => updateSettings("show_resources", event.target.checked)} /><span /><div><strong>Mostrar recursos</strong><small>Publica una sección adicional de equipo y servicios.</small></div></label>
          <label className="switch-card"><input type="checkbox" checked={form.quote_requests_enabled} disabled={!canManage} onChange={(event) => updateSettings("quote_requests_enabled", event.target.checked)} /><span /><div><strong>Recibir solicitudes</strong><small>Habilita el formulario público de cotización.</small></div></label>
          <label className="switch-card">
            <input
              type="checkbox"
              checked={form.web_chat_enabled}
              disabled={!canManage || !form.enabled}
              onChange={(event) =>
                updateSettings(
                  "web_chat_enabled",
                  event.target.checked,
                )
              }
            />
            <span />
            <div>
              <strong>Activar chat web</strong>
              <small>
                Muestra el canal de conversación en el catálogo público.
              </small>
            </div>
          </label>
       </div>
        {canManage && <div className="form-actions"><button className="button button-primary" disabled={saving === "settings"}>{saving === "settings" ? "Guardando…" : "Guardar configuración"}</button></div>}
      </form>

      <section className="panel public-admin-publications">
        <div className="panel-title-row"><div><p className="eyebrow">PAQUETES</p><h3>Paquetes publicados</h3><p>Solo los paquetes activos y listos pueden mostrarse al público.</p></div><Link className="button button-secondary" href="/packages">Administrar paquetes</Link></div>
        <div className="public-admin-package-list">
          {catalog.packages.map((item) => (
            <article className={item.public_visible ? "published" : ""} key={item.id}>
              <div className="public-admin-publication-media">{item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <strong>{item.name.slice(0, 2).toUpperCase()}</strong>}</div>
              <div className="public-admin-publication-copy"><small>{item.ready && item.active ? "LISTO PARA PUBLICAR" : "REQUIERE REVISIÓN"}</small><h4>{item.name}</h4><p>{item.description}</p><div><span>{formatCurrency(item.effective_price, catalog.tenant.currency)}</span><span>{item.item_count} recursos</span></div></div>
              <div className="public-admin-publication-controls">
                <label className="switch-row"><input type="checkbox" checked={item.public_visible} disabled={!canManage || ((!item.ready || !item.active) && !item.public_visible)} onChange={(event) => updatePackage(item.id, { public_visible: event.target.checked, public_featured: event.target.checked ? item.public_featured : false })} /><span /><strong>Visible</strong></label>
                <label className="switch-row"><input type="checkbox" checked={item.public_featured} disabled={!canManage || !item.public_visible} onChange={(event) => updatePackage(item.id, { public_featured: event.target.checked })} /><span /><strong>Destacado</strong></label>
                <label className="compact-number"><span>Orden</span><input type="number" min={0} max={1000000} value={item.public_sort_order} disabled={!canManage} onChange={(event) => updatePackage(item.id, { public_sort_order: Number(event.target.value) || 0 })} /></label>
                {canManage && <button type="button" className="button button-secondary" disabled={saving === `package:${item.id}`} onClick={() => void savePackage(item)}>{saving === `package:${item.id}` ? "Guardando…" : "Guardar"}</button>}
              </div>
            </article>
          ))}
          {catalog.packages.length === 0 && <div className="empty-state"><span className="empty-icon">◇</span><h3>No hay paquetes</h3><p>Crea un paquete antes de configurar su publicación.</p></div>}
        </div>
      </section>

      <section className="panel public-admin-publications">
        <div className="panel-title-row public-resource-heading"><div><p className="eyebrow">RECURSOS</p><h3>Equipo y servicios públicos</h3><p>Configura URL, descripción e imagen específicas para el escaparate.</p></div><div className="public-resource-filters"><input value={resourceFilter} onChange={(event) => setResourceFilter(event.target.value)} placeholder="Buscar recurso…" /><select value={resourceMode} onChange={(event) => setResourceMode(event.target.value as typeof resourceMode)}><option value="ALL">Todos</option><option value="PUBLISHED">Publicados</option><option value="HIDDEN">Ocultos</option></select></div></div>
        <div className="public-admin-resource-list">
          {filteredResources.map((item) => (
            <article className={item.public_visible ? "published" : ""} key={item.id}>
              <header><div><small>{item.category_name || item.resource_type}</small><h4>{item.name}</h4><p>{formatCurrency(item.base_price, catalog.tenant.currency)} / {pricingUnitLabel(item.pricing_unit)} · {item.active ? "Activo" : "Archivado"}</p></div><div className="public-admin-resource-switches"><label className="switch-row"><input type="checkbox" checked={item.public_visible} disabled={!canManage || (!item.active && !item.public_visible)} onChange={(event) => updateResource(item.id, { public_visible: event.target.checked, public_featured: event.target.checked ? item.public_featured : false })} /><span /><strong>Visible</strong></label><label className="switch-row"><input type="checkbox" checked={item.public_featured} disabled={!canManage || !item.public_visible} onChange={(event) => updateResource(item.id, { public_featured: event.target.checked })} /><span /><strong>Destacado</strong></label></div></header>
              <div className="public-admin-resource-fields">
                <label><span>Slug público</span><input value={item.public_slug || ""} disabled={!canManage} onChange={(event) => updateResource(item.id, { public_slug: event.target.value })} placeholder="nombre-del-recurso" />{saving === `resource:${item.id}` && fields.public_slug && <small>{fields.public_slug}</small>}</label>
                <label><span>Orden</span><input type="number" min={0} max={1000000} value={item.public_sort_order} disabled={!canManage} onChange={(event) => updateResource(item.id, { public_sort_order: Number(event.target.value) || 0 })} /></label>
                <label className="span-two"><span>URL de imagen</span><input value={item.public_image_url || ""} disabled={!canManage} onChange={(event) => updateResource(item.id, { public_image_url: event.target.value })} placeholder="https://…" /></label>
                <label className="span-two"><span>Descripción pública</span><textarea rows={3} value={item.public_description} disabled={!canManage} onChange={(event) => updateResource(item.id, { public_description: event.target.value })} placeholder={item.description || "Describe este recurso para tus clientes."} /></label>
              </div>
              {canManage && <footer><button type="button" className="button button-secondary" disabled={saving === `resource:${item.id}`} onClick={() => void saveResource(item)}>{saving === `resource:${item.id}` ? "Guardando…" : "Guardar recurso"}</button></footer>}
            </article>
          ))}
          {filteredResources.length === 0 && <div className="empty-state"><span className="empty-icon">⌕</span><h3>Sin coincidencias</h3><p>Cambia los filtros para encontrar otros recursos.</p></div>}
        </div>
      </section>
    </div>
  );
}
