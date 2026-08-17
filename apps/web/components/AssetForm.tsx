"use client";

import { FormEvent, useState } from "react";
import { ApiError, api } from "@/lib/api";
import type { Asset, AssetStatus } from "@/lib/types";

type Props = {
  resourceId: string;
  initial?: Asset;
  onSaved: (asset: Asset) => void;
  onCancel: () => void;
};

export function AssetForm({ resourceId, initial, onSaved, onCancel }: Props) {
  const [form, setForm] = useState({
    asset_code: initial?.asset_code || "",
    serial_number: initial?.serial_number || "",
    physical_status: initial?.physical_status || ("AVAILABLE" as AssetStatus),
    purchase_date: initial?.purchase_date ? initial.purchase_date.slice(0, 10) : "",
    purchase_price: initial?.purchase_price?.toString() || "",
    notes: initial?.notes || "",
  });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    setFields({});
    try {
      const asset = await api<Asset>(
        initial ? `/api/v1/assets/${initial.id}` : `/api/v1/resources/${resourceId}/assets`,
        {
          method: initial ? "PATCH" : "POST",
          body: JSON.stringify({
            asset_code: form.asset_code,
            serial_number: form.serial_number || null,
            physical_status: form.physical_status,
            purchase_date: form.purchase_date || null,
            purchase_price: form.purchase_price === "" ? null : Number(form.purchase_price),
            notes: form.notes,
          }),
        },
      );
      onSaved(asset);
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible guardar la unidad física.");
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
          <span>Código de activo *</span>
          <input
            value={form.asset_code}
            onChange={(event) => setForm({ ...form, asset_code: event.target.value })}
            placeholder="SPK-JBL-001"
            autoFocus
          />
          {fields.asset_code && <small className="field-error">{fields.asset_code}</small>}
        </label>
        <label className="field">
          <span>Número de serie</span>
          <input
            value={form.serial_number}
            onChange={(event) => setForm({ ...form, serial_number: event.target.value })}
            placeholder="Serial del fabricante"
          />
        </label>
      </div>
      <div className="form-grid three-columns">
        <label className="field">
          <span>Estado físico</span>
          <select
            value={form.physical_status}
            onChange={(event) => setForm({ ...form, physical_status: event.target.value as AssetStatus })}
          >
            <option value="AVAILABLE">Disponible</option>
            <option value="MAINTENANCE">Mantenimiento</option>
            <option value="DAMAGED">Dañado</option>
            <option value="LOST">Perdido</option>
            <option value="RETIRED">Retirado</option>
          </select>
        </label>
        <label className="field">
          <span>Fecha de compra</span>
          <input
            type="date"
            value={form.purchase_date}
            onChange={(event) => setForm({ ...form, purchase_date: event.target.value })}
          />
          {fields.purchase_date && <small className="field-error">{fields.purchase_date}</small>}
        </label>
        <label className="field">
          <span>Precio de compra</span>
          <div className="input-prefix">
            <span>$</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={form.purchase_price}
              onChange={(event) => setForm({ ...form, purchase_price: event.target.value })}
            />
          </div>
        </label>
      </div>
      <label className="field">
        <span>Notas operativas</span>
        <textarea
          rows={3}
          value={form.notes}
          onChange={(event) => setForm({ ...form, notes: event.target.value })}
          placeholder="Condición, ubicación, reparación pendiente…"
        />
      </label>
      <footer className="form-actions">
        <button type="button" className="button button-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button type="submit" className="button button-primary" disabled={saving}>
          {saving ? "Guardando…" : initial ? "Guardar cambios" : "Agregar unidad"}
        </button>
      </footer>
    </form>
  );
}
