"use client";

import { FormEvent, useState } from "react";
import { ApiError, api } from "@/lib/api";
import type { Category } from "@/lib/types";

export function CategoryForm({
  onSaved,
  onCancel,
}: {
  onSaved: (category: Category) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [message, setMessage] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    setFieldError("");
    try {
      const category = await api<Category>("/api/v1/categories", {
        method: "POST",
        body: JSON.stringify({ name, description }),
      });
      onSaved(category);
    } catch (error) {
      if (error instanceof ApiError) {
        setMessage(error.message);
        setFieldError(error.fields?.name || "");
      } else {
        setMessage("No fue posible crear la categoría.");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={submit} className="form-stack">
      {message && <div className="form-alert">{message}</div>}
      <label className="field">
        <span>Nombre *</span>
        <input value={name} onChange={(event) => setName(event.target.value)} placeholder="Ej. Speakers" autoFocus />
        {fieldError && <small className="field-error">{fieldError}</small>}
      </label>
      <label className="field">
        <span>Descripción</span>
        <textarea
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          rows={3}
          placeholder="Qué tipo de recursos agrupa esta categoría."
        />
      </label>
      <footer className="form-actions">
        <button type="button" className="button button-secondary" onClick={onCancel}>
          Cancelar
        </button>
        <button type="submit" className="button button-primary" disabled={saving}>
          {saving ? "Guardando…" : "Crear categoría"}
        </button>
      </footer>
    </form>
  );
}
