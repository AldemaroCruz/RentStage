"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { formatCurrency, formatDateTime, paymentMethodLabel } from "@/lib/format";
import type { PaymentDetail } from "@/lib/types";

export default function PaymentDetailPage() {
  const params = useParams<{ id: string }>();
  const { can } = useAuth();
  const [item, setItem] = useState<PaymentDetail | null>(null);
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);

  useEffect(() => {
    api<PaymentDetail>(`/api/v1/payments/${params.id}`).then(setItem).catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el pago."));
  }, [params.id]);

  async function voidPayment() {
    const reason = window.prompt("Motivo de reversión del pago:");
    if (!reason?.trim()) return;
    setWorking(true);
    try {
      const updated = await api<PaymentDetail>(`/api/v1/payments/${params.id}/void`, { method: "POST", body: JSON.stringify({ reason }) });
      setItem(updated);
    } catch (reasonValue) {
      setError(reasonValue instanceof Error ? reasonValue.message : "No fue posible revertir el pago.");
    } finally {
      setWorking(false);
    }
  }

  if (error && !item) return <div className="panel inline-error">{error}</div>;
  if (!item) return <div className="skeleton detail-skeleton" />;

  return <div className="page-stack payment-detail-page">
    <div className="breadcrumbs"><Link href="/payments">Pagos</Link><span>/</span><span>{item.display_number}</span></div>
    {error && <div className="form-alert">{error}</div>}
    <section className="panel payment-hero"><div><p className="eyebrow">PAYMENT RECEIPT</p><h2>{item.display_number}</h2><p>{item.customer_name} · {formatDateTime(item.received_at)}</p><span className={`billing-status-chip ${item.status === "CONFIRMED" ? "paid" : "void"}`}>{item.status === "CONFIRMED" ? "Confirmado" : "Revertido"}</span></div><div><span>Monto recibido</span><strong>{formatCurrency(item.amount, item.currency)}</strong><small>{paymentMethodLabel(item.method)}</small></div>{can("payment.manage") && item.status === "CONFIRMED" && <button className="button button-danger" disabled={working} onClick={() => void voidPayment()}>{working ? "Revirtiendo…" : "Revertir pago"}</button>}</section>
    <div className="payment-detail-grid"><section className="panel"><div className="panel-header"><div><p className="eyebrow">ALLOCATIONS</p><h2>Facturas aplicadas</h2></div></div><div className="invoice-allocation-list large">{item.allocations.map((allocation) => <Link key={allocation.id} href={`/invoices/${allocation.invoice_id}`}><span>{allocation.display_number}</span><strong>{formatCurrency(allocation.amount, item.currency)}</strong></Link>)}</div></section><section className="panel invoice-side-panel"><p className="eyebrow">DETAIL</p><h2>Información</h2><dl className="profile-definition-list"><div><dt>Método</dt><dd>{paymentMethodLabel(item.method)}</dd></div><div><dt>Referencia</dt><dd>{item.reference || "—"}</dd></div><div><dt>Notas</dt><dd>{item.notes || "—"}</dd></div>{item.voided_at && <div><dt>Revertido</dt><dd>{formatDateTime(item.voided_at)}</dd></div>}{item.void_reason && <div><dt>Motivo</dt><dd>{item.void_reason}</dd></div>}</dl></section></div>
  </div>;
}
