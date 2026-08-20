"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { EmptyState } from "@/components/EmptyState";
import { StatCard } from "@/components/StatCard";
import { api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { DashboardData } from "@/lib/types";

function MetricIcon({ children }: { children: React.ReactNode }) {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {children}
    </svg>
  );
}

export default function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<DashboardData>("/api/v1/dashboard")
      .then(setData)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el dashboard."));
  }, []);

  if (!data && !error) {
    return <DashboardSkeleton />;
  }

  if (error) {
    return (
      <section className="panel connection-panel">
        <span className="connection-icon">!</span>
        <div>
          <h2>No se pudo conectar con RentStage API</h2>
          <p>{error}</p>
          <code>docker compose logs -f api</code>
        </div>
      </section>
    );
  }

  const metrics = data!.metrics;
  const maxCategoryAssets = Math.max(...data!.categories.map((category) => category.asset_count), 1);

  return (
    <div className="page-stack">
      <section className="welcome-banner">
        <div>
          <p className="eyebrow light-eyebrow">OVERVIEW</p>
          <h2>Tu operación, en una sola vista.</h2>
          <p>Inventario, clientes, cotizaciones, reservas y cobros conectados en una sola operación demostrable.</p>
        </div>
        <Link href="/demo" className="button button-light">
          Iniciar demo <span>→</span>
        </Link>
        <span className="welcome-orb welcome-orb-one" />
        <span className="welcome-orb welcome-orb-two" />
      </section>

      <section className="operations-overview-grid">
        <Link href="/calendar?view=agenda" className="operation-metric-card">
          <span className="operation-metric-icon departures">↗</span>
          <div><small>Salidas de hoy</small><strong>{metrics.today_departures}</strong><p>Pedidos que comienzan su período bloqueado</p></div>
        </Link>
        <Link href="/calendar?view=agenda" className="operation-metric-card">
          <span className="operation-metric-icon returns">↙</span>
          <div><small>Retornos de hoy</small><strong>{metrics.today_returns}</strong><p>Reservas entregadas con retorno esperado</p></div>
        </Link>
        <Link href="/calendar#operations-alerts" className={`operation-metric-card ${metrics.overdue_returns > 0 ? "needs-attention" : ""}`}>
          <span className="operation-metric-icon overdue">!</span>
          <div><small>Retornos atrasados</small><strong>{metrics.overdue_returns}</strong><p>{metrics.overdue_returns > 0 ? "Requieren seguimiento inmediato" : "La operación está al día"}</p></div>
        </Link>
        <Link href="/reservations" className="operation-metric-card">
          <span className="operation-metric-icon value">$</span>
          <div><small>Valor comprometido</small><strong>{formatCurrency(metrics.active_value)}</strong><p>{metrics.active_reservations} reservas activas</p></div>
        </Link>
      </section>

      <section className="stats-grid">
        <StatCard
          label="Recursos activos"
          value={metrics.active_resources}
          hint="Productos reservables"
          icon={<MetricIcon><path d="M4 7.5 12 3l8 4.5v9L12 21l-8-4.5z" /><path d="m4.3 7.7 7.7 4.4 7.7-4.4M12 12v9" /></MetricIcon>}
          tone="purple"
        />
        <StatCard
          label="Unidades físicas"
          value={metrics.total_assets}
          hint={`${metrics.available_assets} listas para operar`}
          icon={<MetricIcon><rect x="4" y="4" width="16" height="16" rx="3" /><path d="M8 9h8M8 13h5M8 17h3" /></MetricIcon>}
          tone="positive"
        />
        <StatCard
          label="Requieren atención"
          value={metrics.attention_assets}
          hint="Mantenimiento, daño o pérdida"
          icon={<MetricIcon><path d="M12 3 2.5 20h19z" /><path d="M12 9v4M12 17h.01" /></MetricIcon>}
          tone="warning"
        />
        <StatCard
          label="Inversión registrada"
          value={formatCurrency(metrics.inventory_investment)}
          hint="Costo histórico de activos"
          icon={<MetricIcon><circle cx="12" cy="12" r="9" /><path d="M16 8.5c-.7-.7-1.8-1.1-3-1.1-1.7 0-3 .8-3 2s1 1.8 3 2.2 3 1 3 2.4-1.3 2.5-3 2.5c-1.3 0-2.6-.5-3.4-1.3M12 5.5v13" /></MetricIcon>}
        />
      </section>

      <section className="dashboard-grid">
        <article className="panel panel-wide">
          <header className="panel-header">
            <div>
              <p className="eyebrow">INVENTARIO RECIENTE</p>
              <h2>Recursos activos</h2>
            </div>
            <Link href="/inventory" className="text-link">Ver inventario →</Link>
          </header>

          {data!.recent_resources.length === 0 ? (
            <EmptyState
              icon="◫"
              title="Aún no hay recursos"
              description="Agrega tu primer modelo de equipo para comenzar a controlar unidades físicas."
              action={<Link href="/inventory" className="button button-primary">Agregar recurso</Link>}
            />
          ) : (
            <div className="resource-list compact-list">
              {data!.recent_resources.map((resource) => {
                const availability = resource.asset_count > 0
                  ? Math.round((resource.available_asset_count / resource.asset_count) * 100)
                  : 0;
                return (
                  <Link href={`/inventory/${resource.id}`} key={resource.id} className="resource-list-row">
                    <span className="resource-avatar">{resource.name.slice(0, 2).toUpperCase()}</span>
                    <span className="resource-primary">
                      <strong>{resource.name}</strong>
                      <small>{resource.category_name || "Sin categoría"}</small>
                    </span>
                    <span className="resource-price">
                      <strong>{formatCurrency(resource.base_price)}</strong>
                      <small>por {pricingUnitLabel(resource.pricing_unit)}</small>
                    </span>
                    <span className="resource-stock">
                      <span>
                        <strong>{resource.available_asset_count}</strong> / {resource.asset_count} disponibles
                      </span>
                      <i><b style={{ width: `${availability}%` }} /></i>
                    </span>
                    <span className="row-arrow">→</span>
                  </Link>
                );
              })}
            </div>
          )}
        </article>

        <article className="panel">
          <header className="panel-header">
            <div>
              <p className="eyebrow">DISTRIBUCIÓN</p>
              <h2>Por categoría</h2>
            </div>
          </header>
          <div className="category-bars">
            {data!.categories.map((category) => (
              <div className="category-bar-row" key={category.id}>
                <div>
                  <strong>{category.name}</strong>
                  <small>{category.resource_count} recursos</small>
                </div>
                <span className="category-track">
                  <i style={{ width: `${Math.max((category.asset_count / maxCategoryAssets) * 100, category.asset_count ? 8 : 0)}%` }} />
                </span>
                <b>{category.asset_count}</b>
              </div>
            ))}
            {data!.categories.length === 0 && <p className="muted-copy">No hay categorías registradas.</p>}
          </div>
          <Link href="/categories" className="full-width-link">Administrar categorías <span>→</span></Link>
        </article>
      </section>

      <section className="future-strip">
        <span className="future-icon">✦</span>
        <div>
          <p className="eyebrow">DEMO COMERCIAL · RECORRIDO GUIADO</p>
          <h3>Presenta la operación completa en siete minutos</h3>
          <p>Recorre catálogo, cotización, reserva, inventario, factura y cobro con un escenario coherente. DTE permanece claramente identificado como MOCK / TEST.</p>
        </div>
        <Link className="future-status" href="/assistant">v0.18.0 →</Link>
      </section>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="page-stack">
      <div className="skeleton skeleton-banner" />
      <div className="stats-grid">
        {[1, 2, 3, 4].map((item) => <div key={item} className="skeleton skeleton-card" />)}
      </div>
      <div className="dashboard-grid">
        <div className="skeleton skeleton-panel" />
        <div className="skeleton skeleton-panel" />
      </div>
    </div>
  );
}
