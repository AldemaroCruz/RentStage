"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { formatCurrency, formatDate, invoiceStatusLabel, invoiceStatusTone, monthLabel, paymentMethodLabel } from "@/lib/format";
import type { BillingDashboard } from "@/lib/types";

export default function BillingDashboardPage() {
  const [data, setData] = useState<BillingDashboard | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<BillingDashboard>("/api/v1/billing/dashboard")
      .then(setData)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el dashboard financiero."));
  }, []);

  const chartMax = useMemo(() => {
    if (!data) return 1;
    return Math.max(1, ...data.monthly_billing.map((item) => item.amount), ...data.monthly_payments.map((item) => item.amount));
  }, [data]);

  if (error) return <div className="panel inline-error">{error}</div>;
  if (!data) return <div className="skeleton detail-skeleton" />;

  const { metrics, currency, settings } = data;
  return (
    <div className="page-stack billing-dashboard-page">
      <section className="page-heading">
        <div><p className="eyebrow">BILLING & PAYMENTS</p><h2>Finanzas</h2><p>Facturación interna, cuentas por cobrar, pagos, depósitos e IVA estimado en una sola vista.</p></div>
        <div className="page-heading-actions"><Link className="button button-secondary" href="/settings/billing">Configuración</Link><Link className="button button-primary" href="/invoices/new">Nueva factura</Link></div>
      </section>

      {!settings.fiscal_profile_complete && <section className="panel billing-warning-panel compact"><strong>Perfil fiscal incompleto</strong><p>Completa el perfil fiscal antes de preparar documentos DTE.</p><Link href="/settings/billing">Completar perfil →</Link></section>}

      <section className="billing-metric-grid">
        <article className="panel"><span>Facturado</span><strong>{formatCurrency(metrics.issued_total, currency)}</strong><small>Documentos emitidos</small></article>
        <article className="panel"><span>Cobrado</span><strong>{formatCurrency(metrics.collected_total, currency)}</strong><small>{metrics.paid_count} facturas pagadas</small></article>
        <article className="panel"><span>Por cobrar</span><strong>{formatCurrency(metrics.outstanding_total, currency)}</strong><small>{metrics.open_invoice_count} documentos abiertos</small></article>
        <article className="panel billing-danger-metric"><span>Vencido</span><strong>{formatCurrency(metrics.overdue_total, currency)}</strong><small>{metrics.overdue_count} facturas vencidas</small></article>
        <article className="panel"><span>IVA estimado</span><strong>{formatCurrency(metrics.tax_output_total, currency)}</strong><small>Débito fiscal de ventas emitidas</small></article>
        <article className="panel"><span>Depósitos retenidos</span><strong>{formatCurrency(metrics.deposits_held_total, currency)}</strong><small>Fuera del ingreso de venta</small></article>
      </section>

      <section className="panel billing-chart-panel">
        <div className="panel-header"><div><p className="eyebrow">6 MESES</p><h2>Facturación y cobros</h2></div><span className="billing-chart-legend"><i /> Facturado <i /> Cobrado</span></div>
        <div className="billing-month-chart">
          {data.monthly_billing.map((billed, index) => {
            const paid = data.monthly_payments[index]?.amount || 0;
            return <div className="billing-month-column" key={billed.month}><div className="billing-bars"><span title={`Facturado ${formatCurrency(billed.amount, currency)}`} style={{ height: `${Math.max(3, billed.amount / chartMax * 100)}%` }} /><span title={`Cobrado ${formatCurrency(paid, currency)}`} style={{ height: `${Math.max(3, paid / chartMax * 100)}%` }} /></div><small>{monthLabel(billed.month)}</small></div>;
          })}
        </div>
      </section>

      <div className="billing-dashboard-grid">
        <section className="panel">
          <div className="panel-header"><div><p className="eyebrow">ACCOUNTS RECEIVABLE</p><h2>Facturas recientes</h2></div><Link href="/invoices">Ver todas</Link></div>
          {data.recent_invoices.length === 0 ? <div className="billing-empty-small">Aún no hay facturas.</div> : <div className="billing-list">{data.recent_invoices.map((item) => <Link key={item.id} href={`/invoices/${item.id}`}><span className="billing-list-icon">▤</span><div><strong>{item.display_number}</strong><small>{item.customer_name} · vence {formatDate(item.due_date)}</small></div><div className="billing-list-value"><strong>{formatCurrency(item.balance_due, item.currency)}</strong><span className={`billing-status-chip ${invoiceStatusTone(item.display_status)}`}>{invoiceStatusLabel(item.display_status)}</span></div></Link>)}</div>}
        </section>
        <section className="panel">
          <div className="panel-header"><div><p className="eyebrow">CASH IN</p><h2>Pagos recientes</h2></div><Link href="/payments">Ver todos</Link></div>
          {data.recent_payments.length === 0 ? <div className="billing-empty-small">Aún no hay pagos.</div> : <div className="billing-list">{data.recent_payments.map((item) => <Link key={item.id} href={`/payments/${item.id}`}><span className="billing-list-icon">$</span><div><strong>{item.display_number}</strong><small>{item.customer_name} · {paymentMethodLabel(item.method)}</small></div><div className="billing-list-value"><strong>{formatCurrency(item.amount, item.currency)}</strong><small>{formatDate(item.received_at)}</small></div></Link>)}</div>}
        </section>
      </div>
    </div>
  );
}
