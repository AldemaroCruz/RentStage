"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api } from "@/lib/api";
import { demoReadiness, type DemoStepID } from "@/lib/demo-readiness";
import { formatCurrency } from "@/lib/format";
import type { BillingDashboard, DashboardData, DTESettings, QuoteSummary, ReservationSummary } from "@/lib/types";

type DemoData = {
  dashboard: DashboardData;
  quotes: QuoteSummary[];
  reservations: ReservationSummary[];
  billing: BillingDashboard | null;
  dte: DTESettings | null;
};

type JourneyStep = {
  id: DemoStepID;
  kicker: string;
  title: string;
  description: string;
  href: string;
  action: string;
};

const journey: JourneyStep[] = [
  {
    id: "inventory",
    kicker: "1 · OFERTA",
    title: "Inventario disponible y publicable",
    description: "Modelos, unidades físicas, precios y estado operativo alimentan una oferta que el cliente puede consultar.",
    href: "/inventory",
    action: "Ver inventario",
  },
  {
    id: "quotes",
    kicker: "2 · CONVERSIÓN",
    title: "Cotización con precio y aceptación",
    description: "La propuesta conserva recursos, cantidades, descuentos y evidencia de la decisión comercial.",
    href: "/quotes",
    action: "Ver cotizaciones",
  },
  {
    id: "reservations",
    kicker: "3 · OPERACIÓN",
    title: "Reserva que bloquea disponibilidad",
    description: "Al aceptar, Booking Core vuelve a validar existencias y crea el compromiso operativo del evento.",
    href: "/reservations",
    action: "Ver reservas",
  },
  {
    id: "billing",
    kicker: "4 · CAJA",
    title: "Factura, saldo y pago conectados",
    description: "El equipo conoce cuánto se facturó, cuánto ingresó y qué saldo sigue pendiente sin perder trazabilidad.",
    href: "/billing",
    action: "Ver finanzas",
  },
  {
    id: "fiscal-boundary",
    kicker: "5 · CUMPLIMIENTO",
    title: "DTE separado y seguro para demostración",
    description: "El ciclo fiscal se prueba con proveedor local MOCK en TEST; no se presenta como transmisión productiva a Hacienda.",
    href: "/dte",
    action: "Ver DTE",
  },
];

async function optional<T>(allowed: boolean, path: string): Promise<T | null> {
  if (!allowed) return null;
  try {
    return await api<T>(path);
  } catch {
    return null;
  }
}

export default function CommercialDemoPage() {
  const { me, can } = useAuth();
  const [data, setData] = useState<DemoData | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    Promise.all([
      api<DashboardData>("/api/v1/dashboard"),
      optional<{ items: QuoteSummary[] }>(can("quote.read"), "/api/v1/quotes"),
      optional<{ items: ReservationSummary[] }>(can("reservation.read"), "/api/v1/reservations"),
      optional<BillingDashboard>(can("billing.read"), "/api/v1/billing/dashboard"),
      optional<DTESettings>(can("fiscal.read"), "/api/v1/dte-settings"),
    ])
      .then(([dashboard, quotes, reservations, billing, dte]) => {
        if (!active) return;
        setData({
          dashboard,
          quotes: quotes?.items || [],
          reservations: reservations?.items || [],
          billing,
          dte,
        });
      })
      .catch((reason) => {
        if (active) setError(reason instanceof Error ? reason.message : "No fue posible preparar la demo comercial.");
      });

    return () => { active = false; };
  }, [can]);

  const readiness = useMemo(() => {
    if (!data) return null;
    return demoReadiness({
      activeResourceCount: data.dashboard.metrics.active_resources,
      quoteCount: data.quotes.length,
      acceptedQuoteCount: data.quotes.filter((quote) => quote.status === "ACCEPTED").length,
      activeReservationCount: data.dashboard.metrics.active_reservations,
      issuedTotal: data.billing?.metrics.issued_total || 0,
      collectedTotal: data.billing?.metrics.collected_total || 0,
      dteProviderMode: data.dte?.provider_mode,
      dteEnvironment: data.dte?.environment,
    });
  }, [data]);

  if (error) return <section className="panel inline-error">{error}</section>;
  if (!data || !readiness) return <div className="skeleton detail-skeleton" />;

  const acceptedQuote = data.quotes.find((quote) => quote.status === "ACCEPTED");
  const activeReservation = data.reservations.find((reservation) => ["PENDING", "CONFIRMED", "PREPARING", "READY", "CHECKED_OUT"].includes(reservation.status));
  const recentInvoice = data.billing?.recent_invoices[0];
  const publicCatalogHref = `/p/${me?.active_workspace?.slug || "audiopro-demo"}`;

  const detailByStep: Record<DemoStepID, string> = {
    inventory: `${data.dashboard.metrics.active_resources} recursos · ${data.dashboard.metrics.available_assets} unidades disponibles`,
    quotes: acceptedQuote ? `QT-${String(acceptedQuote.quote_number).padStart(6, "0")} aceptada por ${acceptedQuote.customer_name}` : `${data.quotes.length} cotizaciones visibles`,
    reservations: activeReservation ? `RS-${String(activeReservation.reservation_number).padStart(6, "0")} · ${formatCurrency(activeReservation.total)}` : `${data.dashboard.metrics.active_reservations} reservas activas`,
    billing: `${formatCurrency(data.billing?.metrics.issued_total || 0)} facturado · ${formatCurrency(data.billing?.metrics.collected_total || 0)} cobrado`,
    "fiscal-boundary": data.dte ? `${data.dte.provider_mode} · ${data.dte.environment}` : "Configuración fiscal no disponible para este rol",
  };

  return (
    <div className="page-stack commercial-demo-page">
      <section className="demo-hero">
        <div className="demo-hero-copy">
          <p className="eyebrow light-eyebrow">DEMO COMERCIAL · v0.15.0</p>
          <h2>De una consulta a un cobro, en siete minutos.</h2>
          <p>Presenta cómo una MYPE de alquiler convierte su inventario en ventas y coordina cada evento desde una sola operación.</p>
          <div className="demo-hero-actions">
            <a className="button button-light" href="#recorrido">Comenzar recorrido <span>↓</span></a>
            <Link className="button demo-ghost-button" href={publicCatalogHref} target="_blank" rel="noreferrer">Abrir catálogo público ↗</Link>
          </div>
        </div>
        <div className="demo-score" aria-label={`${readiness.percent}% de la demo lista`}>
          <strong>{readiness.percent}%</strong>
          <span>escenario listo</span>
          <div><i style={{ width: `${readiness.percent}%` }} /></div>
          <small>{readiness.readyCount} de {readiness.totalCount} hitos verificados</small>
        </div>
        <span className="demo-hero-orb" />
      </section>

      <section className="demo-value-grid" aria-label="Resumen del escenario comercial">
        <article className="panel"><span>Oferta lista</span><strong>{data.dashboard.metrics.available_assets}</strong><small>unidades disponibles</small></article>
        <article className="panel"><span>Cotizaciones</span><strong>{data.quotes.length}</strong><small>{data.quotes.filter((quote) => quote.status === "ACCEPTED").length} aceptada</small></article>
        <article className="panel"><span>Valor reservado</span><strong>{formatCurrency(data.dashboard.metrics.active_value)}</strong><small>{data.dashboard.metrics.active_reservations} compromiso activo</small></article>
        <article className="panel"><span>Cobrado</span><strong>{formatCurrency(data.billing?.metrics.collected_total || 0)}</strong><small>de {formatCurrency(data.billing?.metrics.issued_total || 0)} facturado</small></article>
      </section>

      <section className="panel demo-journey-panel" id="recorrido">
        <header className="panel-header demo-section-header">
          <div><p className="eyebrow">RECORRIDO GUIADO</p><h2>Una historia, cinco módulos reales</h2><p>Sigue el orden para mostrar valor comercial, control operativo y límites de cumplimiento.</p></div>
          <span className="demo-time-chip">≈ 7 minutos</span>
        </header>
        <div className="demo-journey-list">
          {journey.map((step, index) => {
            const ready = readiness.steps[step.id];
            const target = step.id === "quotes" && acceptedQuote
              ? `/quotes/${acceptedQuote.id}`
              : step.id === "reservations" && activeReservation
                ? `/reservations/${activeReservation.id}`
                : step.id === "billing" && recentInvoice
                  ? `/invoices/${recentInvoice.id}`
                  : step.href;
            return (
              <article className="demo-journey-step" key={step.id}>
                <span className={`demo-step-number ${ready ? "ready" : "pending"}`}>{ready ? "✓" : index + 1}</span>
                <div className="demo-step-copy">
                  <p className="eyebrow">{step.kicker}</p>
                  <h3>{step.title}</h3>
                  <p>{step.description}</p>
                  <small>{detailByStep[step.id]}</small>
                </div>
                <Link className="button button-secondary" href={target}>{step.action} →</Link>
              </article>
            );
          })}
        </div>
      </section>

      <section className="demo-opportunity-grid">
        <article className="panel demo-presenter-card">
          <p className="eyebrow">GUION PARA PRESENTAR</p>
          <h2>El problema que RentStage resuelve</h2>
          <ol>
            <li><span>01</span><div><strong>La información deja de vivir en chats y hojas separadas.</strong><p>Oferta, cliente y precio comparten el mismo origen.</p></div></li>
            <li><span>02</span><div><strong>Vender ya no significa perder el control del equipo.</strong><p>La reserva bloquea disponibilidad y conecta bodega con calendario.</p></div></li>
            <li><span>03</span><div><strong>El negocio conoce venta, cobro y saldo.</strong><p>La factura y el pago mantienen evidencia y separación fiscal.</p></div></li>
          </ol>
        </article>

        <article className="panel demo-ai-card">
          <span className="demo-roadmap-badge">NUEVO · V0.15</span>
          <div className="demo-ai-icon">✦</div>
          <h2>WhatsApp + AI, con control humano</h2>
          <p>El simulador integrado convierte una consulta en recomendación, disponibilidad y cotización DRAFT sobre datos reales. La respuesta y la cotización requieren aprobación humana.</p>
          <div className="demo-ai-boundary"><strong>v0.15.0</strong><span>Inbox, análisis, aprobación y auditoría</span></div>
          <div className="demo-ai-boundary future"><strong>Después</strong><span>Conectar Meta Business sin cambiar el flujo</span></div>
          <Link className="button button-primary" href="/assistant">Abrir WhatsApp AI →</Link>
        </article>
      </section>

      <section className="demo-evidence-strip">
        <div><p className="eyebrow">EVIDENCIA DE PRODUCTO</p><h3>Multiempresa, roles, auditoría, CI/CD y staging real</h3><p>El recorrido ocurre sobre los mismos módulos protegidos y desplegados que valida la suite de integración.</p></div>
        <Link href="/audit" className="text-link">Ver trazabilidad →</Link>
      </section>
    </div>
  );
}
