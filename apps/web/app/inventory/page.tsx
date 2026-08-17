"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { Modal } from "@/components/Modal";
import { ResourceForm } from "@/components/ResourceForm";
import { StatusBadge } from "@/components/StatusBadge";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { Category, Resource } from "@/lib/types";

export default function InventoryPage() {
  const { can } = useAuth();
  const canManageCatalog = can("catalog.manage");
  const [resources, setResources] = useState<Resource[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState("active");
  const [createOpen, setCreateOpen] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [resourceResponse, categoryResponse] = await Promise.all([
        api<{ items: Resource[] }>("/api/v1/resources"),
        api<{ items: Category[] }>("/api/v1/categories"),
      ]);
      setResources(resourceResponse.items);
      setCategories(categoryResponse.items);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible cargar el inventario.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    return resources.filter((resource) => {
      const matchesSearch = !query || resource.name.toLowerCase().includes(query) || resource.sku?.toLowerCase().includes(query);
      const matchesCategory = !category || resource.category_id === category;
      const matchesStatus = status === "all" || (status === "active" ? resource.active : !resource.active);
      return matchesSearch && matchesCategory && matchesStatus;
    });
  }, [resources, search, category, status]);

  async function archive(resource: Resource) {
    if (!window.confirm(`¿Archivar ${resource.name}? Sus unidades físicas conservarán su historial.`)) return;
    try {
      const updated = await api<Resource>(`/api/v1/resources/${resource.id}`, { method: "DELETE" });
      setResources((items) => items.map((item) => (item.id === updated.id ? updated : item)));
    } catch (reason) {
      window.alert(reason instanceof ApiError ? reason.message : "No fue posible archivar el recurso.");
    }
  }

  const totals = resources.reduce(
    (acc, resource) => ({
      models: acc.models + (resource.active ? 1 : 0),
      assets: acc.assets + resource.asset_count,
      available: acc.available + resource.available_asset_count,
      attention: acc.attention + resource.attention_asset_count,
    }),
    { models: 0, assets: 0, available: 0, attention: 0 },
  );

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div>
          <p className="eyebrow">CATALOG + PHYSICAL ASSETS</p>
          <h2>Inventario de recursos</h2>
          <p>Administra los modelos que alquilas y cada unidad física que realmente existe en bodega.</p>
        </div>
        {canManageCatalog && <button className="button button-primary" onClick={() => setCreateOpen(true)}>
          <span className="button-plus">+</span> Agregar recurso
        </button>}
      </section>

      <section className="inventory-summary-grid">
        <div><span>Modelos activos</span><strong>{totals.models}</strong></div>
        <div><span>Unidades físicas</span><strong>{totals.assets}</strong></div>
        <div><span>Disponibles</span><strong>{totals.available}</strong></div>
        <div className={totals.attention ? "summary-attention" : ""}><span>Requieren atención</span><strong>{totals.attention}</strong></div>
      </section>

      <section className="panel inventory-panel">
        <div className="inventory-toolbar">
          <label className="search-box">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
            <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Buscar por nombre o SKU…" />
          </label>
          <select value={category} onChange={(event) => setCategory(event.target.value)} aria-label="Filtrar por categoría">
            <option value="">Todas las categorías</option>
            {categories.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
          <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="Filtrar por estado">
            <option value="active">Activos</option>
            <option value="inactive">Archivados</option>
            <option value="all">Todos</option>
          </select>
          <span className="toolbar-count">{filtered.length} resultados</span>
        </div>

        {loading ? (
          <div className="table-skeleton">Cargando inventario…</div>
        ) : error ? (
          <div className="inline-error">{error}<button onClick={() => void load()}>Reintentar</button></div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon="◫"
            title={resources.length ? "No encontramos coincidencias" : "Agrega tu primer recurso"}
            description={resources.length ? "Prueba cambiando los filtros de búsqueda." : "Crea el modelo comercial y luego registra sus unidades físicas."}
            action={!resources.length && canManageCatalog ? <button className="button button-primary" onClick={() => setCreateOpen(true)}>Agregar recurso</button> : undefined}
          />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table inventory-table">
              <thead>
                <tr>
                  <th>Recurso</th>
                  <th>Categoría</th>
                  <th>Precio</th>
                  <th>Inventario físico</th>
                  <th>Estado</th>
                  <th aria-label="Acciones" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((resource) => {
                  const percentage = resource.asset_count ? Math.round((resource.available_asset_count / resource.asset_count) * 100) : 0;
                  return (
                    <tr key={resource.id}>
                      <td>
                        <Link href={`/inventory/${resource.id}`} className="table-resource-name">
                          <span>{resource.name.slice(0, 2).toUpperCase()}</span>
                          <div>
                            <strong>{resource.name}</strong>
                            <small>{resource.sku || "Sin SKU"}</small>
                          </div>
                        </Link>
                      </td>
                      <td><span className="category-pill">{resource.category_name || "Sin categoría"}</span></td>
                      <td><strong>{formatCurrency(resource.base_price)}</strong><small className="table-subline">por {pricingUnitLabel(resource.pricing_unit)}</small></td>
                      <td>
                        <div className="stock-cell">
                          <span><strong>{resource.available_asset_count}</strong> de {resource.asset_count} disponibles</span>
                          <i><b style={{ width: `${percentage}%` }} /></i>
                          {resource.attention_asset_count > 0 && <small>{resource.attention_asset_count} requieren atención</small>}
                        </div>
                      </td>
                      <td><StatusBadge status={resource.active ? "ACTIVE" : "INACTIVE"} /></td>
                      <td>
                        <div className="row-actions">
                          <Link href={`/inventory/${resource.id}`} className="icon-action" title="Abrir detalle">→</Link>
                          {resource.active && canManageCatalog && <button className="icon-action" onClick={() => void archive(resource)} title="Archivar">•••</button>}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Modal open={createOpen} title="Agregar recurso" eyebrow="CATÁLOGO RENTSTAGE" onClose={() => setCreateOpen(false)}>
        <ResourceForm
          categories={categories}
          onCancel={() => setCreateOpen(false)}
          onSaved={(resource) => {
            setResources((items) => [resource, ...items]);
            setCreateOpen(false);
          }}
        />
      </Modal>
    </div>
  );
}
