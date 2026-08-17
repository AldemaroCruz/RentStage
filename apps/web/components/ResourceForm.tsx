"use client";

import { FormEvent, useState } from "react";
import { ApiError, api } from "@/lib/api";
import type { Category, Resource } from "@/lib/types";

type Props = {
  categories: Category[];
  initial?: Resource;
  onSaved: (resource: Resource) => void;
  onCancel: () => void;
};

export function ResourceForm({ categories, initial, onSaved, onCancel }: Props) {
  const [form, setForm] = useState({
    category_id: initial?.category_id || "",
    resource_type: initial?.resource_type || "EQUIPMENT",
    name: initial?.name || "",
    description: initial?.description || "",
    sku: initial?.sku || "",
    base_price: initial?.base_price?.toString() || "0",
    pricing_unit: initial?.pricing_unit || "DAY",
    deposit_amount: initial?.deposit_amount?.toString() || "0",
    brand: typeof initial?.metadata?.brand === "string" ? initial.metadata.brand : "",
    model: typeof initial?.metadata?.model === "string" ? initial.metadata.model : "",
    track_individual_assets: initial?.track_individual_assets ?? true,
    active: initial?.active ?? true,
  });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  function update<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setFields((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    setFields({});
    const payload = {
      category_id: form.category_id || null,
      resource_type: form.resource_type,
      name: form.name,
      description: form.description,
      sku: form.sku || null,
      base_price: Number(form.base_price || 0),
      pricing_unit: form.pricing_unit,
      deposit_amount: Number(form.deposit_amount || 0),
      track_individual_assets: form.track_individual_assets,
      active: form.active,
      metadata: {
        ...(initial?.metadata || {}),
        brand: form.brand || undefined,
        model: form.model || undefined,
      },
    };

    try {
      const resource = await api<Resource>(
        initial ? `/api/v1/resources/${initial.id}` : "/api/v1/resources",
        {
          method: initial ? "PATCH" : "POST",
          body: JSON.stringify(payload),
        },
      );
      onSaved(resource);
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible guardar el recurso.");
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
          <span>Nombre del recurso *</span>
          <input
            value={form.name}
            onChange={(event) => update("name", event.target.value)}
            placeholder="Ej. JBL PRX815W"
            autoFocus
          />
          {fields.name && <small className="field-error">{fields.name}</small>}
        </label>
        <label className="field">
          <span>SKU interno</span>
          <input
            value={form.sku}
            onChange={(event) => update("sku", event.target.value)}
            placeholder="JBL-PRX815W"
          />
          {fields.sku && <small className="field-error">{fields.sku}</small>}
        </label>
      </div>

      <div className="form-grid two-columns">
        <label className="field">
          <span>Categoría</span>
          <select value={form.category_id} onChange={(event) => update("category_id", event.target.value)}>
            <option value="">Sin categoría</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span>Tipo de recurso</span>
          <select
            value={form.resource_type}
            onChange={(event) => update("resource_type", event.target.value as typeof form.resource_type)}
          >
            <option value="EQUIPMENT">Equipo</option>
            <option value="SPACE">Espacio</option>
            <option value="SERVICE">Servicio</option>
          </select>
        </label>
      </div>

      <label className="field">
        <span>Descripción</span>
        <textarea
          rows={3}
          value={form.description}
          onChange={(event) => update("description", event.target.value)}
          placeholder="Descripción breve para operaciones y catálogo."
        />
      </label>

      <div className="form-grid two-columns">
        <label className="field">
          <span>Marca</span>
          <input value={form.brand} onChange={(event) => update("brand", event.target.value)} placeholder="JBL" />
        </label>
        <label className="field">
          <span>Modelo</span>
          <input value={form.model} onChange={(event) => update("model", event.target.value)} placeholder="PRX815W" />
        </label>
      </div>

      <div className="form-grid three-columns">
        <label className="field">
          <span>Precio base *</span>
          <div className="input-prefix">
            <span>$</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={form.base_price}
              onChange={(event) => update("base_price", event.target.value)}
            />
          </div>
          {fields.base_price && <small className="field-error">{fields.base_price}</small>}
        </label>
        <label className="field">
          <span>Unidad de precio</span>
          <select
            value={form.pricing_unit}
            onChange={(event) => update("pricing_unit", event.target.value as typeof form.pricing_unit)}
          >
            <option value="HOUR">Por hora</option>
            <option value="DAY">Por día</option>
            <option value="EVENT">Por evento</option>
            <option value="FIXED">Precio fijo</option>
          </select>
        </label>
        <label className="field">
          <span>Depósito</span>
          <div className="input-prefix">
            <span>$</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={form.deposit_amount}
              onChange={(event) => update("deposit_amount", event.target.value)}
            />
          </div>
        </label>
      </div>

      <div className="check-grid">
        <label className="check-row">
          <input
            type="checkbox"
            checked={form.track_individual_assets}
            onChange={(event) => update("track_individual_assets", event.target.checked)}
          />
          <span>
            <strong>Rastrear unidades físicas</strong>
            <small>Cada unidad tendrá código, serie y estado independiente.</small>
          </span>
        </label>
        <label className="check-row">
          <input type="checkbox" checked={form.active} onChange={(event) => update("active", event.target.checked)} />
          <span>
            <strong>Recurso activo</strong>
            <small>Disponible para futuras cotizaciones y reservas.</small>
          </span>
        </label>
      </div>

      <footer className="form-actions">
        <button type="button" className="button button-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button type="submit" className="button button-primary" disabled={saving}>
          {saving ? "Guardando…" : initial ? "Guardar cambios" : "Agregar recurso"}
        </button>
      </footer>
    </form>
  );
}
