"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { useAuth } from "@/components/AuthProvider";
import { api } from "@/lib/api";
import { formatCurrency } from "@/lib/format";
import type { RentalPackageSummary } from "@/lib/types";

export default function PackagesPage() {
  const { can } = useAuth();
  const canManage = can("package.manage");
  const [items, setItems] = useState<RentalPackageSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("active");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const response = await api<{ items: RentalPackageSummary[] }>("/api/v1/packages");
      setItems(response.items);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible cargar los paquetes.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    return items.filter((item) => {
      const matchesText = !query || item.name.toLowerCase().includes(query) || item.slug.includes(query) || item.description.toLowerCase().includes(query);
      const matchesStatus = status === "all" || (status === "active" ? item.active : !item.active);
      return matchesText && matchesStatus;
    });
  }, [items, search, status]);

  const totals = useMemo(() => {
    const active = items.filter((item) => item.active);
    const ready = active.filter((item) => item.ready);
    const fixed = ready.filter((item) => item.pricing_mode === "FIXED");
    const savings = fixed.reduce((sum, item) => sum + Math.max(0, item.calculated_price - item.effective_price), 0);
    return {
      active: active.length,
      ready: ready.length,
      attention: active.length - ready.length,
      resources: ready.reduce((sum, item) => sum + item.item_count, 0),
      units: ready.reduce((sum, item) => sum + item.total_quantity, 0),
      savings,
    };
  }, [items]);

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div>
          <p className="eyebrow">PACKAGES CORE</p>
          <h2>Paquetes comerciales</h2>
          <p>Combina recursos, cantidades y precios reutilizables para cotizar eventos con mayor velocidad.</p>
        </div>
        {canManage && <Link href="/packages/new" className="button button-primary"><span className="button-plus">+</span> Nuevo paquete</Link>}
      </section>

      <section className="package-summary-grid">
        <div><span>Paquetes listos</span><strong>{totals.ready}</strong><small>{totals.attention > 0 ? `${totals.attention} activos requieren atención` : `${totals.active} activos disponibles`}</small></div>
        <div><span>Recursos configurados</span><strong>{totals.resources}</strong><small>líneas entre paquetes activos</small></div>
        <div><span>Unidades incluidas</span><strong>{totals.units}</strong><small>por una unidad de cada paquete</small></div>
        <div className={totals.savings > 0 ? "summary-positive" : ""}><span>Ahorro configurado</span><strong>{formatCurrency(totals.savings)}</strong><small>frente a precios por componente</small></div>
      </section>

      <section className="panel packages-panel">
        <div className="inventory-toolbar package-toolbar">
          <label className="search-box">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
            <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Buscar por nombre, slug o descripción…" />
          </label>
          <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="Filtrar paquetes por estado">
            <option value="active">Activos</option><option value="inactive">Archivados</option><option value="all">Todos</option>
          </select>
          <span className="toolbar-count">{filtered.length} resultados</span>
        </div>

        {loading ? (
          <div className="table-skeleton">Cargando paquetes…</div>
        ) : error ? (
          <div className="inline-error">{error}<button onClick={() => void load()}>Reintentar</button></div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon="◇"
            title={items.length ? "No encontramos coincidencias" : "Crea tu primer paquete"}
            description={items.length ? "Prueba cambiando los filtros." : "Agrupa el equipo que normalmente vendes junto y reutilízalo en nuevas cotizaciones."}
            action={!items.length && canManage ? <Link href="/packages/new" className="button button-primary">Crear paquete</Link> : undefined}
          />
        ) : (
          <div className="package-card-grid">
            {filtered.map((item) => {
              const saving = Math.max(0, item.calculated_price - item.effective_price);
              const markup = Math.max(0, item.effective_price - item.calculated_price);
              return (
                <Link href={`/packages/${item.id}`} className={`package-card ${!item.active ? "archived" : ""}`} key={item.id}>
                  <div className="package-card-visual">
                    {item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <strong>{item.name.slice(0, 2).toUpperCase()}</strong>}
                    <i className={!item.active ? "inactive" : item.ready ? "active" : "attention"}>{!item.active ? "Archivado" : item.ready ? "Listo" : "Revisar"}</i>
                  </div>
                  <div className="package-card-body">
                    <div className="package-card-title"><div><small>{item.pricing_mode === "FIXED" ? "PRECIO FIJO" : "SUMA DE RECURSOS"}</small><h3>{item.name}</h3></div><span>→</span></div>
                    <p>{item.description || "Sin descripción comercial."}</p>
                    <div className="package-card-tags">
                      {item.guest_capacity && <span>Hasta {item.guest_capacity} personas</span>}
                      <span>{item.item_count} recursos</span><span>{item.total_quantity} unidades</span>
                      {item.unavailable_item_count > 0 && <span className="warning">Requiere atención</span>}
                    </div>
                    <div className="package-card-pricing">
                      <div><small>Precio de venta</small><strong>{formatCurrency(item.effective_price)}</strong></div>
                      <div><small>Componentes</small><strong>{formatCurrency(item.calculated_price)}</strong></div>
                      {saving > 0 && <em>−{formatCurrency(saving)}</em>}
                      {markup > 0 && <em className="markup">+{formatCurrency(markup)}</em>}
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
