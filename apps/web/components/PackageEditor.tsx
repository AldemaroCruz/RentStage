"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { PackagePricingMode, RentalPackage, Resource } from "@/lib/types";

type DraftItem = {
  key: string;
  resource_id: string;
  description: string;
  quantity: string;
  unit_price_override: string;
};

type Props = {
  initial?: RentalPackage;
  readOnly?: boolean;
  onSaved?: (rentalPackage: RentalPackage) => void;
};

function draftKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

function blankItem(): DraftItem {
  return { key: draftKey(), resource_id: "", description: "", quantity: "1", unit_price_override: "" };
}

function slugify(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 140);
}

function money(value: string): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? Math.max(0, Math.round(parsed * 100) / 100) : 0;
}

export function PackageEditor({ initial, readOnly = false, onSaved }: Props) {
  const router = useRouter();
  const [resources, setResources] = useState<Resource[]>([]);
  const [loadingResources, setLoadingResources] = useState(true);
  const [resourceError, setResourceError] = useState("");
  const [saving, setSaving] = useState(false);
  const [archiving, setArchiving] = useState(false);
  const [message, setMessage] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [slugTouched, setSlugTouched] = useState(Boolean(initial?.slug));
  const [form, setForm] = useState({
    name: initial?.name || "",
    slug: initial?.slug || "",
    description: initial?.description || "",
    guest_capacity: initial?.guest_capacity ? String(initial.guest_capacity) : "",
    pricing_mode: (initial?.pricing_mode || "SUM_ITEMS") as PackagePricingMode,
    fixed_price: initial?.fixed_price == null ? "" : String(initial.fixed_price),
    image_url: initial?.image_url || "",
    active: initial?.active ?? true,
  });
  const [items, setItems] = useState<DraftItem[]>(
    initial?.items.length
      ? initial.items.map((item) => ({
          key: item.id,
          resource_id: item.resource_id,
          description: item.description,
          quantity: String(item.quantity),
          unit_price_override: item.unit_price_override == null ? "" : String(item.unit_price_override),
        }))
      : [blankItem()],
  );

  useEffect(() => {
    api<{ items: Resource[] }>("/api/v1/resources")
      .then((response) => {
        const byID = new Map(response.items.map((resource) => [resource.id, resource]));
        for (const packageItem of initial?.items || []) {
          if (!byID.has(packageItem.resource_id)) {
            byID.set(packageItem.resource_id, {
              id: packageItem.resource_id,
              resource_type: packageItem.resource_type,
              name: packageItem.resource_name,
              description: "",
              base_price: packageItem.base_price,
              pricing_unit: packageItem.pricing_unit,
              deposit_amount: 0,
              track_individual_assets: true,
              active: packageItem.resource_active,
              metadata: {},
              asset_count: packageItem.asset_count,
              available_asset_count: packageItem.available_asset_count,
              attention_asset_count: packageItem.attention_asset_count,
              created_at: packageItem.created_at,
              updated_at: packageItem.updated_at,
            });
          }
        }
        setResources([...byID.values()].sort((left, right) => left.name.localeCompare(right.name)));
      })
      .catch((reason) => setResourceError(reason instanceof Error ? reason.message : "No fue posible cargar el catálogo."))
      .finally(() => setLoadingResources(false));
  }, [initial?.items]);

  const calculations = useMemo(() => {
    const lines = items.map((item) => {
      const resource = resources.find((candidate) => candidate.id === item.resource_id);
      const unitPrice = item.unit_price_override === "" ? resource?.base_price || 0 : money(item.unit_price_override);
      const quantity = Math.max(0, Number(item.quantity) || 0);
      return Math.round(quantity * unitPrice * 100) / 100;
    });
    const componentTotal = Math.round(lines.reduce((sum, value) => sum + value, 0) * 100) / 100;
    const sellingPrice = form.pricing_mode === "FIXED" ? money(form.fixed_price) : componentTotal;
    const adjustment = Math.round((componentTotal - sellingPrice) * 100) / 100;
    const totalUnits = items.reduce((sum, item) => sum + Math.max(0, Number(item.quantity) || 0), 0);
    return { lines, componentTotal, sellingPrice, adjustment, totalUnits };
  }, [form.fixed_price, form.pricing_mode, items, resources]);

  function updateItem(index: number, patch: Partial<DraftItem>) {
    setItems((current) => current.map((item, itemIndex) => (itemIndex === index ? { ...item, ...patch } : item)));
  }

  function selectPackageResource(index: number, resourceID: string) {
    const resource = resources.find((candidate) => candidate.id === resourceID);
    updateItem(index, {
      resource_id: resourceID,
      description: resource?.name || "",
      unit_price_override: "",
    });
  }

  function changeName(value: string) {
    setForm((current) => ({
      ...current,
      name: value,
      slug: slugTouched ? current.slug : slugify(value),
    }));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (readOnly) return;
    setSaving(true);
    setMessage("");
    setFields({});
    try {
      const payload = {
        name: form.name,
        slug: form.slug,
        description: form.description,
        guest_capacity: form.guest_capacity ? Number(form.guest_capacity) : null,
        pricing_mode: form.pricing_mode,
        fixed_price: form.pricing_mode === "FIXED" ? money(form.fixed_price) : null,
        image_url: form.image_url || null,
        active: form.active,
        items: items.map((item, index) => ({
          resource_id: item.resource_id,
          description: item.description,
          quantity: Number(item.quantity),
          unit_price_override: item.unit_price_override === "" ? null : money(item.unit_price_override),
          sort_order: index,
        })),
      };
      const saved = await api<RentalPackage>(initial ? `/api/v1/packages/${initial.id}` : "/api/v1/packages", {
        method: initial ? "PATCH" : "POST",
        body: JSON.stringify(payload),
      });
      if (onSaved) {
        onSaved(saved);
        setMessage("Paquete guardado correctamente.");
      } else {
        router.push(`/packages/${saved.id}`);
        router.refresh();
      }
    } catch (reason) {
      if (reason instanceof ApiError) {
        setMessage(reason.message);
        setFields(reason.fields || {});
      } else {
        setMessage("No fue posible guardar el paquete.");
      }
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setSaving(false);
    }
  }

  async function archive() {
    if (!initial || !window.confirm(`¿Archivar “${initial.name}”? Ya no podrá agregarse a nuevas cotizaciones.`)) return;
    setArchiving(true);
    setMessage("");
    try {
      await api<RentalPackage>(`/api/v1/packages/${initial.id}`, { method: "DELETE" });
      router.push("/packages");
      router.refresh();
    } catch (reason) {
      setMessage(reason instanceof ApiError ? reason.message : "No fue posible archivar el paquete.");
    } finally {
      setArchiving(false);
    }
  }

  if (loadingResources) return <div className="skeleton detail-skeleton" />;
  if (resourceError) return <div className="panel inline-error">{resourceError}</div>;

  return (
    <form className="package-editor-layout" onSubmit={submit}>
      <div className="package-editor-main page-stack">
        {message && <div className="form-alert">{message}</div>}

        <section className="panel package-form-section">
          <div className="panel-header">
            <div>
              <p className="eyebrow">PACKAGE IDENTITY</p>
              <h2>Información comercial</h2>
              <p>Esta definición pertenece únicamente al workspace actual.</p>
            </div>
            <label className="package-active-toggle">
              <input
                type="checkbox"
                checked={form.active}
                disabled={readOnly}
                onChange={(event) => setForm({ ...form, active: event.target.checked })}
              />
              <span>{form.active ? "Activo" : "Archivado"}</span>
            </label>
          </div>
          <div className="package-form-body">
            <div className="form-grid two-columns">
              <label className="field">
                <span>Nombre *</span>
                <input value={form.name} disabled={readOnly} onChange={(event) => changeName(event.target.value)} placeholder="Paquete Fiesta 100 personas" />
                {fields.name && <small className="field-error">{fields.name}</small>}
              </label>
              <label className="field">
                <span>Slug *</span>
                <input
                  value={form.slug}
                  disabled={readOnly}
                  onChange={(event) => { setSlugTouched(true); setForm({ ...form, slug: slugify(event.target.value) }); }}
                  placeholder="paquete-fiesta-100-personas"
                />
                {fields.slug && <small className="field-error">{fields.slug}</small>}
                <small className="field-hint">Preparado para el catálogo público de una versión posterior.</small>
              </label>
            </div>
            <div className="form-grid two-columns">
              <label className="field">
                <span>Capacidad sugerida</span>
                <input type="number" min="1" step="1" value={form.guest_capacity} disabled={readOnly} onChange={(event) => setForm({ ...form, guest_capacity: event.target.value })} placeholder="100" />
                {fields.guest_capacity && <small className="field-error">{fields.guest_capacity}</small>}
              </label>
              <label className="field">
                <span>Imagen (URL opcional)</span>
                <input type="url" value={form.image_url} disabled={readOnly} onChange={(event) => setForm({ ...form, image_url: event.target.value })} placeholder="https://…" />
                {fields.image_url && <small className="field-error">{fields.image_url}</small>}
              </label>
            </div>
            <label className="field">
              <span>Descripción</span>
              <textarea rows={4} value={form.description} disabled={readOnly} onChange={(event) => setForm({ ...form, description: event.target.value })} placeholder="Describe qué resuelve el paquete y para qué tipo de evento está pensado." />
              {fields.description && <small className="field-error">{fields.description}</small>}
            </label>
          </div>
        </section>

        <section className="panel package-form-section">
          <div className="panel-header package-lines-header">
            <div>
              <p className="eyebrow">PACKAGE COMPOSITION</p>
              <h2>Recursos incluidos</h2>
              <p>Los paquetes agrupan modelos y cantidades; las unidades físicas se asignan al preparar la reserva.</p>
            </div>
            {!readOnly && (
              <button type="button" className="button button-secondary" onClick={() => setItems((current) => [...current, blankItem()])}>
                <span className="button-plus">+</span> Agregar recurso
              </button>
            )}
          </div>
          <div className="package-lines">
            {items.map((item, index) => {
              const resource = resources.find((candidate) => candidate.id === item.resource_id);
              const selectedElsewhere = new Set(items.filter((_, itemIndex) => itemIndex !== index).map((candidate) => candidate.resource_id));
              return (
                <article className="package-line-card" key={item.key}>
                  <div className="package-line-number">{index + 1}</div>
                  <div className="package-line-content">
                    <div className="form-grid package-line-grid">
                      <label className="field package-resource-field">
                        <span>Recurso *</span>
                        <select value={item.resource_id} disabled={readOnly} onChange={(event) => selectPackageResource(index, event.target.value)}>
                          <option value="">Seleccionar recurso</option>
                          {resources.map((option) => (
                            <option key={option.id} value={option.id} disabled={!option.active || selectedElsewhere.has(option.id)}>
                              {option.name}{!option.active ? " · archivado" : ""} · {formatCurrency(option.base_price)}/{pricingUnitLabel(option.pricing_unit)}
                            </option>
                          ))}
                        </select>
                        {fields[`items[${index}].resource_id`] && <small className="field-error">{fields[`items[${index}].resource_id`]}</small>}
                        {resource && (
                          <small className={`field-hint ${!resource.active ? "field-hint-warning" : ""}`}>
                            {resource.available_asset_count} de {resource.asset_count} unidades físicas disponibles ahora
                          </small>
                        )}
                      </label>
                      <label className="field">
                        <span>Cantidad *</span>
                        <input type="number" min="1" step="1" value={item.quantity} disabled={readOnly} onChange={(event) => updateItem(index, { quantity: event.target.value })} />
                        {fields[`items[${index}].quantity`] && <small className="field-error">{fields[`items[${index}].quantity`]}</small>}
                      </label>
                      <label className="field">
                        <span>Precio unitario opcional</span>
                        <div className="input-prefix">
                          <span>$</span>
                          <input type="number" min="0" step="0.01" value={item.unit_price_override} disabled={readOnly} onChange={(event) => updateItem(index, { unit_price_override: event.target.value })} placeholder={resource ? String(resource.base_price) : "0.00"} />
                        </div>
                        {fields[`items[${index}].unit_price_override`] && <small className="field-error">{fields[`items[${index}].unit_price_override`]}</small>}
                        <small className="field-hint">Vacío usa el precio base vigente del recurso.</small>
                      </label>
                    </div>
                    <div className="package-line-description-row">
                      <label className="field">
                        <span>Descripción para la cotización</span>
                        <input
                          value={item.description}
                          disabled={readOnly}
                          onChange={(event) => updateItem(index, { description: event.target.value })}
                          placeholder={resource?.name || "Se usará el nombre del recurso"}
                        />
                        {fields[`items[${index}].description`] && <small className="field-error">{fields[`items[${index}].description`]}</small>}
                      </label>
                    </div>
                    <div className="package-line-footer">
                      <span>{resource?.name || "Recurso sin seleccionar"}</span>
                      <strong>{formatCurrency(calculations.lines[index] || 0)}</strong>
                      {!readOnly && (
                        <button type="button" className="icon-action package-remove-line" onClick={() => setItems((current) => current.filter((_, itemIndex) => itemIndex !== index))} disabled={items.length === 1} title="Eliminar recurso">×</button>
                      )}
                    </div>
                  </div>
                </article>
              );
            })}
            {fields.items && <div className="quote-items-error">{fields.items}</div>}
          </div>
        </section>
      </div>

      <aside className="package-summary-panel panel">
        <div className="package-summary-header">
          <p className="eyebrow">PRICING</p>
          <h2>Precio del paquete</h2>
          <p>RentStage convertirá el paquete en líneas normales de cotización con snapshots de precio.</p>
        </div>
        <div className="package-pricing-mode">
          <label className={form.pricing_mode === "SUM_ITEMS" ? "selected" : ""}>
            <input type="radio" name="pricing_mode" value="SUM_ITEMS" checked={form.pricing_mode === "SUM_ITEMS"} disabled={readOnly} onChange={() => setForm({ ...form, pricing_mode: "SUM_ITEMS", fixed_price: "" })} />
            <span><strong>Suma de recursos</strong><small>El total cambia con la composición.</small></span>
          </label>
          <label className={form.pricing_mode === "FIXED" ? "selected" : ""}>
            <input type="radio" name="pricing_mode" value="FIXED" checked={form.pricing_mode === "FIXED"} disabled={readOnly} onChange={() => setForm({ ...form, pricing_mode: "FIXED", fixed_price: form.fixed_price || String(calculations.componentTotal) })} />
            <span><strong>Precio fijo</strong><small>Permite descuento o margen comercial.</small></span>
          </label>
        </div>
        {form.pricing_mode === "FIXED" && (
          <label className="field package-fixed-price-field">
            <span>Precio fijo *</span>
            <div className="input-prefix"><span>$</span><input type="number" min="0" step="0.01" value={form.fixed_price} disabled={readOnly} onChange={(event) => setForm({ ...form, fixed_price: event.target.value })} /></div>
            {fields.fixed_price && <small className="field-error">{fields.fixed_price}</small>}
          </label>
        )}
        {fields.pricing_mode && <small className="field-error">{fields.pricing_mode}</small>}

        <dl className="package-total-list">
          <div><dt>Recursos</dt><dd>{items.filter((item) => item.resource_id).length}</dd></div>
          <div><dt>Unidades por paquete</dt><dd>{calculations.totalUnits}</dd></div>
          <div><dt>Suma de componentes</dt><dd>{formatCurrency(calculations.componentTotal)}</dd></div>
          {form.pricing_mode === "FIXED" && calculations.adjustment !== 0 && (
            <div className={calculations.adjustment > 0 ? "package-saving" : "package-markup"}>
              <dt>{calculations.adjustment > 0 ? "Ahorro comercial" : "Margen adicional"}</dt>
              <dd>{formatCurrency(Math.abs(calculations.adjustment))}</dd>
            </div>
          )}
          <div className="package-total-final"><dt>Precio de venta</dt><dd>{formatCurrency(calculations.sellingPrice)}</dd></div>
        </dl>

        <div className="package-summary-actions">
          <Link href="/packages" className="button button-secondary">Volver</Link>
          {!readOnly && <button type="submit" className="button button-primary" disabled={saving || resources.length === 0}>{saving ? "Guardando…" : initial ? "Guardar cambios" : "Crear paquete"}</button>}
          {!readOnly && initial?.active && <button type="button" className="button button-danger-ghost" disabled={archiving} onClick={() => void archive()}>{archiving ? "Archivando…" : "Archivar paquete"}</button>}
        </div>
      </aside>
    </form>
  );
}
