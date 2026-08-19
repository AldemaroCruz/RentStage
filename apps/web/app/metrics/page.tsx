"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { customerSourceLabel, metricBarPercent, responseTimeLabel } from "@/lib/commercial-metrics";
import { formatCurrency, formatDate, formatDateTime, monthLabel } from "@/lib/format";
import type { CommercialMetricsReport } from "@/lib/types";

type WindowDays = 7 | 30 | 90;

export default function CommercialMetricsPage() {
  const [days, setDays] = useState<WindowDays>(30);
  const [data, setData] = useState<CommercialMetricsReport | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setError("");
    api<CommercialMetricsReport>(`/api/v1/metrics/commercial?days=${days}`)
      .then((result) => { if (active) setData(result); })
      .catch((reason) => {
        if (active) setError(reason instanceof Error ? reason.message : "No fue posible cargar las métricas comerciales.");
      });
    return () => { active = false; };
  }, [days]);

  const maxima = useMemo(() => {
    if (!data) return { funnel: 1, monthly: 1, sources: 1 };
    return {
      funnel: Math.max(1, ...data.funnel.map((item) => item.count)),
      monthly: Math.max(1, ...data.monthly_activity.flatMap((item) => [item.quote_value, item.reservation_value, item.collected_value])),
      sources: Math.max(1, ...data.customer_sources.map((item) => item.count)),
    };
  }, [data]);

  if (error) return <section className="panel inline-error">{error}</section>;
  if (!data) return <div className="skeleton detail-skeleton" />;

  const { overview, reservation_outcomes: outcomes } = data;
  return (
    <div className="page-stack commercial-metrics-page">
      <section className="page-heading commercial-metrics-heading">
        <div>
          <p className="eyebrow">COMMERCIAL INTELLIGENCE</p>
          <h2>Métricas operativas</h2>
          <p>Convierte la actividad real de RentStage en evidencia del embudo, velocidad comercial, reservas y cobros.</p>
        </div>
        <label className="metrics-window-control">
          <span>Ventana</span>
          <select value={days} onChange={(event) => setDays(Number(event.target.value) as WindowDays)}>
            <option value={7}>Últimos 7 días</option>
            <option value={30}>Últimos 30 días</option>
            <option value={90}>Últimos 90 días</option>
          </select>
        </label>
      </section>

      <section className="metrics-context-strip">
        <div><strong>{formatDate(data.window.start_at)}</strong><span>Inicio del período</span></div>
        <div><strong>{formatDate(data.window.end_at)}</strong><span>Cierre del período</span></div>
        <div><strong>{formatDateTime(data.generated_at)}</strong><span>Actualizado</span></div>
        <p>Actividad por fecha de creación o cobro. Pipeline y saldo pendiente son fotografías actuales.</p>
      </section>

      <section className="commercial-kpi-grid">
        <article className="panel"><span>Consultas</span><strong>{overview.inquiries}</strong><small>{overview.public_requests} web · {overview.assistant_conversations} chat</small></article>
        <article className="panel"><span>Aceptación</span><strong>{overview.quote_acceptance_rate.toFixed(1)}%</strong><small>{overview.quotes_accepted} aceptadas de {overview.quotes_accepted + overview.quotes_rejected} decididas</small></article>
        <article className="panel"><span>Respuesta promedio</span><strong>{responseTimeLabel(overview.average_response_minutes, overview.response_samples)}</strong><small>{overview.response_samples} interacciones con respuesta enviada</small></article>
        <article className="panel"><span>Valor aceptado</span><strong>{formatCurrency(overview.accepted_quote_value, data.currency)}</strong><small>Cotizaciones aceptadas en el período</small></article>
        <article className="panel"><span>Cobrado</span><strong>{formatCurrency(overview.collected_value, data.currency)}</strong><small>Pagos confirmados en el período</small></article>
        <article className="panel snapshot"><span>Por cobrar · hoy</span><strong>{formatCurrency(overview.outstanding_value, data.currency)}</strong><small>Saldo abierto al momento de consulta</small></article>
      </section>

      <section className="commercial-main-grid">
        <article className="panel commercial-funnel-card">
          <header className="panel-header"><div><p className="eyebrow">EMBUDO OPERATIVO</p><h2>Actividad por etapa</h2></div><span>{days} días</span></header>
          <p className="metrics-method-note">Las etapas resumen actividad del período; no representan una cohorte cerrada y pueden incluir ventas originadas antes de la ventana.</p>
          <div className="commercial-funnel">
            {data.funnel.map((stage) => (
              <div key={stage.key} className="commercial-funnel-row">
                <div><strong>{stage.label}</strong><small>{stage.description}</small></div>
                <span className="commercial-funnel-track"><i style={{ width: `${metricBarPercent(stage.count, maxima.funnel)}%` }} /></span>
                <b>{stage.count}</b>
              </div>
            ))}
          </div>
          <div className="commercial-rate-pair">
            <div><span>Aceptada → reserva</span><strong>{overview.quote_to_reservation_rate.toFixed(1)}%</strong></div>
            <div><span>Nuevos clientes</span><strong>{overview.new_customers}</strong></div>
          </div>
        </article>

        <article className="panel commercial-value-card">
          <header className="panel-header"><div><p className="eyebrow">VALOR COMERCIAL</p><h2>Del interés al efectivo</h2></div></header>
          <dl className="commercial-value-list">
            <div><dt>Pipeline actual</dt><dd>{formatCurrency(overview.quote_pipeline_value, data.currency)}</dd><small>Borradores y enviadas</small></div>
            <div><dt>Reservado en período</dt><dd>{formatCurrency(overview.reservation_value, data.currency)}</dd><small>Reservas no canceladas</small></div>
            <div><dt>Facturado en período</dt><dd>{formatCurrency(overview.issued_value, data.currency)}</dd><small>{overview.invoices_issued} documentos emitidos</small></div>
            <div className="positive"><dt>Cobrado en período</dt><dd>{formatCurrency(overview.collected_value, data.currency)}</dd><small>Pagos confirmados</small></div>
          </dl>
          <Link href="/billing" className="full-width-link">Abrir finanzas <span>→</span></Link>
        </article>
      </section>

      <section className="panel commercial-trend-card">
        <header className="panel-header">
          <div><p className="eyebrow">ÚLTIMOS 6 MESES</p><h2>Valor generado y cobrado</h2></div>
          <span className="commercial-chart-legend"><i /> Cotizado <i /> Reservado <i /> Cobrado</span>
        </header>
        <div className="commercial-month-chart">
          {data.monthly_activity.map((month) => (
            <div className="commercial-month-column" key={month.month}>
              <div className="commercial-month-bars">
                <span title={`Cotizado ${formatCurrency(month.quote_value, data.currency)}`} style={{ height: `${metricBarPercent(month.quote_value, maxima.monthly)}%` }} />
                <span title={`Reservado ${formatCurrency(month.reservation_value, data.currency)}`} style={{ height: `${metricBarPercent(month.reservation_value, maxima.monthly)}%` }} />
                <span title={`Cobrado ${formatCurrency(month.collected_value, data.currency)}`} style={{ height: `${metricBarPercent(month.collected_value, maxima.monthly)}%` }} />
              </div>
              <small>{monthLabel(month.month)}</small>
            </div>
          ))}
        </div>
      </section>

      <section className="commercial-detail-grid">
        <article className="panel metrics-outcomes-card">
          <header className="panel-header"><div><p className="eyebrow">RESULTADO OPERATIVO</p><h2>Reservas</h2></div></header>
          <div className="metrics-outcome-list">
            <div><span>Activas</span><strong>{outcomes.active}</strong></div>
            <div><span>Completadas</span><strong>{outcomes.completed}</strong></div>
            <div><span>Canceladas</span><strong>{outcomes.cancelled}</strong></div>
          </div>
          <p>Tasa de cancelación sobre reservas con resultado: <strong>{outcomes.cancellation_rate.toFixed(1)}%</strong></p>
        </article>

        <article className="panel metrics-sources-card">
          <header className="panel-header"><div><p className="eyebrow">ADQUISICIÓN</p><h2>Nuevos clientes por origen</h2></div></header>
          <div className="metrics-source-list">
            {data.customer_sources.map((source) => <div key={source.source}><span>{customerSourceLabel(source.source)}</span><i><b style={{ width: `${metricBarPercent(source.count, maxima.sources)}%` }} /></i><strong>{source.count}</strong></div>)}
          </div>
        </article>

        <article className="panel metrics-evidence-card">
          <header className="panel-header"><div><p className="eyebrow">EVIDENCIA DEMOSTRABLE</p><h2>Control y trazabilidad</h2></div></header>
          <div className="metrics-evidence-list">
            <div><span>Mensajes con aprobación humana</span><strong>{overview.human_approved_messages}</strong></div>
            <div><span>Decisiones del cliente en portal</span><strong>{overview.customer_portal_decisions}</strong></div>
            <div><span>Eventos de auditoría</span><strong>{overview.audit_events}</strong></div>
          </div>
          <p>Los conteos provienen de registros tenant-scoped. El canal DEMO no equivale a mensajes enviados por Meta.</p>
        </article>
      </section>
    </div>
  );
}
