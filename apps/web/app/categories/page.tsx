"use client";

import { useEffect, useState } from "react";
import { CategoryForm } from "@/components/CategoryForm";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { Modal } from "@/components/Modal";
import { ApiError, api } from "@/lib/api";
import type { Category } from "@/lib/types";

export default function CategoriesPage() {
  const { can } = useAuth();
  const canManage = can("catalog.manage");
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);

  useEffect(() => {
    api<{ items: Category[] }>("/api/v1/categories")
      .then((response) => setCategories(response.items))
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar las categorías."))
      .finally(() => setLoading(false));
  }, []);

  async function remove(category: Category) {
    if (!window.confirm(`¿Eliminar la categoría ${category.name}?`)) return;
    try {
      await api<void>(`/api/v1/categories/${category.id}`, { method: "DELETE" });
      setCategories((items) => items.filter((item) => item.id !== category.id));
    } catch (reason) {
      window.alert(reason instanceof ApiError ? reason.message : "No fue posible eliminar la categoría.");
    }
  }

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div><p className="eyebrow">CATALOG STRUCTURE</p><h2>Categorías</h2><p>Organiza equipos, espacios y servicios sin amarrar el core únicamente al alquiler de audio.</p></div>
        {canManage && <button className="button button-primary" onClick={() => setOpen(true)}><span className="button-plus">+</span> Nueva categoría</button>}
      </section>

      {loading ? (
        <div className="skeleton skeleton-panel" />
      ) : error ? (
        <div className="inline-error">{error}</div>
      ) : categories.length === 0 ? (
        <section className="panel"><EmptyState icon="▦" title="No hay categorías" description="Crea la primera categoría para clasificar tus recursos." action={canManage ? <button className="button button-primary" onClick={() => setOpen(true)}>Crear categoría</button> : undefined} /></section>
      ) : (
        <section className="category-card-grid">
          {categories.map((category, index) => (
            <article className="category-card" key={category.id}>
              <div className={`category-card-icon category-tone-${(index % 4) + 1}`}><span>{category.name.slice(0, 2).toUpperCase()}</span></div>
              <div className="category-card-copy"><h3>{category.name}</h3><p>{category.description || "Sin descripción."}</p></div>
              <div className="category-card-footer"><span><strong>{category.resource_count}</strong> recursos activos</span>{canManage && <button onClick={() => void remove(category)} disabled={category.resource_count > 0} title={category.resource_count > 0 ? "La categoría contiene recursos" : "Eliminar categoría"}>Eliminar</button>}</div>
            </article>
          ))}
        </section>
      )}

      <section className="architecture-note"><span>i</span><div><strong>Categorías por tenant</strong><p>Cada empresa administra su propio catálogo. Dos clientes pueden usar el mismo nombre sin compartir información.</p></div></section>

      <Modal open={open} title="Nueva categoría" eyebrow="ORGANIZACIÓN DEL CATÁLOGO" onClose={() => setOpen(false)} width="560px">
        <CategoryForm onCancel={() => setOpen(false)} onSaved={(category) => { setCategories((items) => [...items, category].sort((a, b) => a.name.localeCompare(b.name))); setOpen(false); }} />
      </Modal>
    </div>
  );
}
