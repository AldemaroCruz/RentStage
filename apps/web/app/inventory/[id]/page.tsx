"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AssetForm } from "@/components/AssetForm";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { Modal } from "@/components/Modal";
import { ResourceForm } from "@/components/ResourceForm";
import { StatusBadge } from "@/components/StatusBadge";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, formatDate, pricingUnitLabel } from "@/lib/format";
import type { Asset, AssetStatus, Category, Resource } from "@/lib/types";

export default function InventoryDetailPage() {
  const { can } = useAuth();
  const canManageCatalog = can("catalog.manage");
  const canManageInventory = can("inventory.manage");
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const resourceId = params.id;
  const [resource, setResource] = useState<Resource | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [assetModalOpen, setAssetModalOpen] = useState(false);
  const [resourceModalOpen, setResourceModalOpen] = useState(false);
  const [editingAsset, setEditingAsset] = useState<Asset | undefined>();

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [resourceResponse, assetsResponse, categoryResponse] = await Promise.all([
        api<Resource>(`/api/v1/resources/${resourceId}`),
        api<{ items: Asset[] }>(`/api/v1/resources/${resourceId}/assets`),
        api<{ items: Category[] }>("/api/v1/categories"),
      ]);
      setResource(resourceResponse);
      setAssets(assetsResponse.items);
      setCategories(categoryResponse.items);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible cargar el recurso.");
    } finally {
      setLoading(false);
    }
  }, [resourceId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function quickStatus(asset: Asset, status: AssetStatus) {
    try {
      const updated = await api<Asset>(`/api/v1/assets/${asset.id}`, {
        method: "PATCH",
        body: JSON.stringify({ physical_status: status }),
      });
      setAssets((items) => items.map((item) => (item.id === updated.id ? updated : item)));
      await refreshResourceMetrics();
    } catch (reason) {
      window.alert(reason instanceof ApiError ? reason.message : "No fue posible cambiar el estado.");
    }
  }

  async function refreshResourceMetrics() {
    const refreshed = await api<Resource>(`/api/v1/resources/${resourceId}`);
    setResource(refreshed);
  }

  async function archiveResource() {
    if (!resource || !window.confirm(`¿Archivar ${resource.name}?`)) return;
    try {
      await api<Resource>(`/api/v1/resources/${resource.id}`, { method: "DELETE" });
      router.push("/inventory");
    } catch (reason) {
      window.alert(reason instanceof ApiError ? reason.message : "No fue posible archivar el recurso.");
    }
  }

  if (loading) {
    return <div className="skeleton skeleton-panel detail-skeleton" />;
  }

  if (error || !resource) {
    return (
      <section className="panel connection-panel">
        <span className="connection-icon">!</span>
        <div><h2>Recurso no disponible</h2><p>{error || "No encontramos este recurso."}</p><Link href="/inventory" className="text-link">← Volver al inventario</Link></div>
      </section>
    );
  }

  const brand = typeof resource.metadata.brand === "string" ? resource.metadata.brand : "";
  const model = typeof resource.metadata.model === "string" ? resource.metadata.model : "";

  return (
    <div className="page-stack">
      <div className="breadcrumbs"><Link href="/inventory">Inventario</Link><span>/</span><span>{resource.name}</span></div>

      <section className="resource-hero panel">
        <span className="resource-hero-mark">{resource.name.slice(0, 2).toUpperCase()}</span>
        <div className="resource-hero-copy">
          <div className="resource-title-row">
            <h2>{resource.name}</h2>
            <StatusBadge status={resource.active ? "ACTIVE" : "INACTIVE"} />
          </div>
          <p>{resource.description || "Sin descripción operativa."}</p>
          <div className="resource-meta-line">
            <span><small>SKU</small><strong>{resource.sku || "—"}</strong></span>
            <span><small>Categoría</small><strong>{resource.category_name || "Sin categoría"}</strong></span>
            <span><small>Marca / modelo</small><strong>{[brand, model].filter(Boolean).join(" ") || "—"}</strong></span>
          </div>
        </div>
        <div className="resource-hero-actions">
          {canManageCatalog && <button className="button button-secondary" onClick={() => setResourceModalOpen(true)}>Editar recurso</button>}
          {canManageCatalog && resource.active && <button className="button button-danger-ghost" onClick={() => void archiveResource()}>Archivar</button>}
        </div>
      </section>

      <section className="resource-metric-grid">
        <div><span>Precio base</span><strong>{formatCurrency(resource.base_price)}</strong><small>por {pricingUnitLabel(resource.pricing_unit)}</small></div>
        <div><span>Depósito</span><strong>{formatCurrency(resource.deposit_amount)}</strong><small>configurado por alquiler</small></div>
        <div><span>Unidades</span><strong>{resource.asset_count}</strong><small>activos no retirados</small></div>
        <div><span>Disponibles</span><strong>{resource.available_asset_count}</strong><small>estado físico disponible</small></div>
        <div className={resource.attention_asset_count ? "metric-warning" : ""}><span>Atención</span><strong>{resource.attention_asset_count}</strong><small>mantenimiento, daño o pérdida</small></div>
      </section>

      <section className="panel inventory-panel">
        <header className="panel-header assets-header">
          <div><p className="eyebrow">PHYSICAL ASSETS</p><h2>Unidades físicas</h2><p>Cada código representa un equipo identificable dentro de la operación.</p></div>
          {canManageInventory && <button className="button button-primary" onClick={() => { setEditingAsset(undefined); setAssetModalOpen(true); }}><span className="button-plus">+</span> Agregar unidad</button>}
        </header>

        {assets.length === 0 ? (
          <EmptyState
            icon="▣"
            title="Este recurso aún no tiene unidades"
            description="Agrega las unidades físicas para que el motor pueda calcular disponibilidad real."
            action={canManageInventory ? <button className="button button-primary" onClick={() => setAssetModalOpen(true)}>Agregar primera unidad</button> : undefined}
          />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table asset-table">
              <thead><tr><th>Código</th><th>Serie</th><th>Estado físico</th><th>Compra</th><th>Valor</th><th>Notas</th><th aria-label="Acciones" /></tr></thead>
              <tbody>
                {assets.map((asset) => (
                  <tr key={asset.id} className={asset.physical_status === "RETIRED" ? "retired-row" : ""}>
                    <td><strong>{asset.asset_code}</strong></td>
                    <td><span className="mono-copy">{asset.serial_number || "—"}</span></td>
                    <td>
                      <div className="asset-status-control">
                        <StatusBadge status={asset.physical_status} />
                        <select value={asset.physical_status} disabled={!canManageInventory} onChange={(event) => void quickStatus(asset, event.target.value as AssetStatus)} aria-label={`Cambiar estado de ${asset.asset_code}`} title={!canManageInventory ? "Tu rol no puede modificar inventario físico." : undefined}>
                          <option value="AVAILABLE">Disponible</option><option value="MAINTENANCE">Mantenimiento</option><option value="DAMAGED">Dañado</option><option value="LOST">Perdido</option><option value="RETIRED">Retirado</option>
                        </select>
                      </div>
                    </td>
                    <td>{formatDate(asset.purchase_date)}</td>
                    <td>{asset.purchase_price == null ? "—" : formatCurrency(asset.purchase_price)}</td>
                    <td><span className="notes-cell" title={asset.notes}>{asset.notes || "—"}</span></td>
                    <td>{canManageInventory && <button className="icon-action" onClick={() => { setEditingAsset(asset); setAssetModalOpen(true); }} title="Editar unidad">✎</button>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="architecture-note">
        <span>i</span>
        <div><strong>Separación Resource ↔ Asset</strong><p>Las cotizaciones reservarán cantidades del recurso. Las operaciones asignarán unidades físicas específicas antes del check-out.</p></div>
      </section>

      <Modal open={resourceModalOpen} title="Editar recurso" eyebrow="CATÁLOGO RENTSTAGE" onClose={() => setResourceModalOpen(false)}>
        <ResourceForm
          categories={categories}
          initial={resource}
          onCancel={() => setResourceModalOpen(false)}
          onSaved={(updated) => { setResource(updated); setResourceModalOpen(false); }}
        />
      </Modal>

      <Modal
        open={assetModalOpen}
        title={editingAsset ? "Editar unidad física" : "Agregar unidad física"}
        eyebrow={resource.name.toUpperCase()}
        onClose={() => { setAssetModalOpen(false); setEditingAsset(undefined); }}
        width="720px"
      >
        <AssetForm
          resourceId={resource.id}
          initial={editingAsset}
          onCancel={() => { setAssetModalOpen(false); setEditingAsset(undefined); }}
          onSaved={(saved) => {
            setAssets((items) => {
              const exists = items.some((item) => item.id === saved.id);
              return exists ? items.map((item) => (item.id === saved.id ? saved : item)) : [...items, saved];
            });
            setAssetModalOpen(false);
            setEditingAsset(undefined);
            void refreshResourceMetrics();
          }}
        />
      </Modal>
    </div>
  );
}
