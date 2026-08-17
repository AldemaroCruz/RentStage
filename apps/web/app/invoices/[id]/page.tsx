"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { formatCurrency, formatDate, formatDateTime, invoiceStatusLabel, invoiceStatusTone, paymentMethodLabel } from "@/lib/format";
import type { DTEDocumentDetail, InvoiceDetail, PaymentDetail, PaymentMethod } from "@/lib/types";

export default function InvoiceDetailPage() {
  const params = useParams<{ id: string }>();
  const { can } = useAuth();
  const canManage = can("billing.manage");
  const canManagePayments = can("payment.manage");
  const canReadDTE = can("fiscal.read");
  const canManageDTE = can("fiscal.manage");
  const [invoice, setInvoice] = useState<InvoiceDetail | null>(null);
  const [error, setError] = useState("");
  const [working, setWorking] = useState("");
  const [paymentOpen, setPaymentOpen] = useState(false);
  const [payment, setPayment] = useState({ amount: 0, method: "BANK_TRANSFER" as PaymentMethod, reference: "", notes: "" });
  const [paymentMessage, setPaymentMessage] = useState("");
  const [dte, setDTE] = useState<DTEDocumentDetail | null>(null);
  const [dteMessage, setDTEMessage] = useState("");
  const [documentType, setDocumentType] = useState<"AUTO" | "01" | "03">("AUTO");

  const load = useCallback(() => {
    setError("");
    const invoiceRequest = api<InvoiceDetail>(`/api/v1/invoices/${params.id}`);
    const dteRequest = canReadDTE
      ? api<DTEDocumentDetail>(`/api/v1/invoices/${params.id}/dte`).catch((reason) => {
          if (reason instanceof ApiError && reason.status === 404) return null;
          throw reason;
        })
      : Promise.resolve(null);
    Promise.all([invoiceRequest, dteRequest])
      .then(([item, fiscalDocument]) => {
        setInvoice(item);
        setDTE(fiscalDocument);
        setPayment((current) => ({ ...current, amount: item.balance_due }));
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la factura."));
  }, [params.id, canReadDTE]);

  useEffect(() => { load(); }, [load]);

  async function issue() {
    if (!window.confirm("¿Emitir esta factura y asignar el siguiente número? Después ya no podrá editarse.")) return;
    setWorking("issue");
    try {
      const updated = await api<InvoiceDetail>(`/api/v1/invoices/${params.id}/issue`, { method: "POST", body: "{}" });
      setInvoice(updated);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible emitir la factura.");
    } finally {
      setWorking("");
    }
  }

  async function voidInvoice() {
    const reason = window.prompt("Motivo de anulación:");
    if (!reason?.trim()) return;
    setWorking("void");
    try {
      const updated = await api<InvoiceDetail>(`/api/v1/invoices/${params.id}/void`, { method: "POST", body: JSON.stringify({ reason }) });
      setInvoice(updated);
    } catch (reasonValue) {
      setError(reasonValue instanceof Error ? reasonValue.message : "No fue posible anular la factura.");
    } finally {
      setWorking("");
    }
  }


  async function prepareDTE() {
    if (!invoice || !canManageDTE) return;
    setWorking("dte");
    setDTEMessage("");
    setError("");
    try {
      const created = await api<DTEDocumentDetail>(`/api/v1/invoices/${invoice.id}/dte`, {
        method: "POST",
        body: JSON.stringify({ document_type: documentType === "AUTO" ? "" : documentType }),
      });
      setDTE(created);
      setDTEMessage(`DTE ${created.control_number} preparado. Revisa el JSON antes de transmitir.`);
      const refreshed = await api<InvoiceDetail>(`/api/v1/invoices/${invoice.id}`);
      setInvoice(refreshed);
    } catch (reason) {
      if (reason instanceof ApiError && reason.fields) {
        setDTEMessage(Object.values(reason.fields).join(" "));
      } else {
        setDTEMessage(reason instanceof Error ? reason.message : "No fue posible preparar el DTE.");
      }
    } finally {
      setWorking("");
    }
  }

  async function recordPayment(event: FormEvent) {
    event.preventDefault();
    if (!invoice) return;
    setWorking("payment");
    setPaymentMessage("");
    try {
      const created = await api<PaymentDetail>("/api/v1/payments", {
        method: "POST",
        body: JSON.stringify({
          customer_id: invoice.customer_id,
          amount: Number(payment.amount),
          currency: invoice.currency,
          method: payment.method,
          reference: payment.reference,
          notes: payment.notes,
          received_at: new Date().toISOString(),
          allocations: [{ invoice_id: invoice.id, amount: Number(payment.amount) }],
        }),
      });
      setPaymentMessage(`Pago ${created.display_number} registrado.`);
      setPaymentOpen(false);
      load();
    } catch (reason) {
      if (reason instanceof ApiError && reason.fields) {
        setPaymentMessage(Object.values(reason.fields).join(" "));
      } else {
        setPaymentMessage(reason instanceof Error ? reason.message : "No fue posible registrar el pago.");
      }
    } finally {
      setWorking("");
    }
  }

  if (error && !invoice) return <div className="panel inline-error">{error}</div>;
  if (!invoice) return <div className="skeleton detail-skeleton" />;

  const openForPayment = invoice.status === "ISSUED" || invoice.status === "PARTIALLY_PAID";
  return (
    <div className="page-stack invoice-detail-page">
      <div className="breadcrumbs"><Link href="/invoices">Facturas</Link><span>/</span><span>{invoice.display_number}</span></div>
      {error && <div className="form-alert">{error}</div>}
      {paymentMessage && <div className={paymentMessage.includes("registrado") ? "success-banner" : "form-alert"}>{paymentMessage}</div>}
      {dteMessage && <div className={dteMessage.includes("preparado") ? "success-banner" : "form-alert"}>{dteMessage}</div>}

      <section className="panel invoice-hero">
        <div><p className="eyebrow">INTERNAL INVOICE</p><h2>{invoice.display_number}</h2><p>{invoice.customer_name} · emitida {formatDate(invoice.issue_date)}</p><div className="invoice-hero-meta"><span className={`billing-status-chip ${invoiceStatusTone(invoice.display_status)}`}>{invoiceStatusLabel(invoice.display_status)}</span><span>{invoice.fiscal_status === "READY_FOR_DTE" ? "Lista para DTE" : invoice.fiscal_status === "ACCEPTED" ? "DTE aceptado" : invoice.fiscal_status === "SUBMITTED" ? "DTE transmitido" : invoice.fiscal_status === "VOIDED" ? "DTE invalidado" : invoice.fiscal_status}</span></div></div>
        <div className="invoice-hero-total"><span>Saldo pendiente</span><strong>{formatCurrency(invoice.balance_due, invoice.currency)}</strong><small>Total {formatCurrency(invoice.total_amount, invoice.currency)}</small></div>
        <div className="invoice-hero-actions">
          <Link className="button button-secondary" href={`/invoices/${invoice.id}/print`} target="_blank">Imprimir / PDF</Link>
          {canManage && invoice.status === "DRAFT" && <button className="button button-primary" type="button" disabled={working === "issue"} onClick={() => void issue()}>{working === "issue" ? "Emitiendo…" : "Emitir factura"}</button>}
          {canManagePayments && openForPayment && invoice.balance_due > 0 && <button className="button button-primary" type="button" onClick={() => setPaymentOpen((value) => !value)}>Registrar pago</button>}
          {canManage && (invoice.status === "DRAFT" || (invoice.status === "ISSUED" && invoice.paid_amount === 0 && !["SUBMITTED", "ACCEPTED"].includes(invoice.fiscal_status))) && <button className="button button-danger" type="button" disabled={working === "void"} onClick={() => void voidInvoice()}>Anular</button>}
        </div>
      </section>

      {paymentOpen && <form className="panel invoice-payment-form" onSubmit={recordPayment}>
        <div className="panel-header"><div><p className="eyebrow">PAYMENT</p><h2>Registrar pago</h2></div><button type="button" className="icon-button" onClick={() => setPaymentOpen(false)}>×</button></div>
        <div className="form-grid four-columns">
          <label className="field"><span>Monto</span><input type="number" min="0.01" max={invoice.balance_due} step="0.01" value={payment.amount} onChange={(event) => setPayment({ ...payment, amount: Number(event.target.value) })} /></label>
          <label className="field"><span>Método</span><select value={payment.method} onChange={(event) => setPayment({ ...payment, method: event.target.value as PaymentMethod })}><option value="BANK_TRANSFER">Transferencia</option><option value="CASH">Efectivo</option><option value="CARD">Tarjeta</option><option value="CHECK">Cheque</option><option value="OTHER">Otro</option></select></label>
          <label className="field"><span>Referencia</span><input value={payment.reference} onChange={(event) => setPayment({ ...payment, reference: event.target.value })} /></label>
          <label className="field"><span>Notas</span><input value={payment.notes} onChange={(event) => setPayment({ ...payment, notes: event.target.value })} /></label>
        </div>
        <div className="form-actions"><button className="button button-primary" type="submit" disabled={working === "payment"}>{working === "payment" ? "Registrando…" : `Registrar ${formatCurrency(payment.amount, invoice.currency)}`}</button></div>
      </form>}

      <div className="invoice-detail-grid">
        <section className="panel invoice-document-panel">
          <div className="invoice-document-parties">
            <article><small>EMISOR</small><strong>{invoice.seller_legal_name || invoice.seller_trade_name || "Perfil fiscal pendiente"}</strong><span>{invoice.seller_tax_id ? `NIT ${invoice.seller_tax_id}` : "NIT pendiente"}</span><span>{invoice.seller_registration_number ? `NRC ${invoice.seller_registration_number}` : "NRC pendiente"}</span><span>{invoice.seller_address || "Dirección fiscal pendiente"}</span></article>
            <article><small>CLIENTE</small><strong>{invoice.customer_name}</strong><span>{invoice.customer_tax_id ? `ID fiscal ${invoice.customer_tax_id}` : "Sin identificación fiscal"}</span><span>{invoice.customer_email || invoice.customer_phone || "Sin contacto"}</span><span>{invoice.customer_address || "Sin dirección de facturación"}</span></article>
          </div>
          <div className="invoice-date-strip"><span>Emisión <strong>{formatDate(invoice.issue_date)}</strong></span><span>Vencimiento <strong>{formatDate(invoice.due_date)}</strong></span><span>Moneda <strong>{invoice.currency}</strong></span><span>Precios <strong>{invoice.prices_include_tax ? "IVA incluido" : "IVA separado"}</strong></span></div>
          <div className="data-table-wrap"><table className="data-table invoice-items-table"><thead><tr><th>Descripción</th><th>Cant.</th><th>Precio</th><th>Descuento</th><th>Clasificación</th><th>IVA</th><th>Total</th></tr></thead><tbody>{invoice.items.map((item) => <tr key={item.id}><td><strong>{item.description}</strong></td><td>{item.quantity}</td><td>{formatCurrency(item.unit_price, invoice.currency)}</td><td>{formatCurrency(item.discount_amount, invoice.currency)}</td><td><span className="category-pill">{item.tax_category === "TAXABLE" ? "Gravada" : item.tax_category === "EXEMPT" ? "Exenta" : "No sujeta"}</span></td><td>{formatCurrency(item.tax_amount, invoice.currency)}<small>{item.tax_rate}%</small></td><td><strong>{formatCurrency(item.line_total, invoice.currency)}</strong></td></tr>)}</tbody></table></div>
          <div className="invoice-document-bottom"><div><h3>Notas</h3><p>{invoice.notes || "Sin notas."}</p><h3>Términos</h3><p>{invoice.terms || "Sin términos adicionales."}</p></div><dl><div><dt>Ventas gravadas</dt><dd>{formatCurrency(invoice.taxable_amount, invoice.currency)}</dd></div><div><dt>Ventas exentas</dt><dd>{formatCurrency(invoice.exempt_amount, invoice.currency)}</dd></div><div><dt>No sujetas</dt><dd>{formatCurrency(invoice.non_taxable_amount, invoice.currency)}</dd></div><div><dt>IVA</dt><dd>{formatCurrency(invoice.tax_amount, invoice.currency)}</dd></div><div className="invoice-total-row"><dt>Total</dt><dd>{formatCurrency(invoice.total_amount, invoice.currency)}</dd></div><div><dt>Pagado</dt><dd>{formatCurrency(invoice.paid_amount, invoice.currency)}</dd></div><div className="invoice-balance-row"><dt>Saldo</dt><dd>{formatCurrency(invoice.balance_due, invoice.currency)}</dd></div></dl></div>
        </section>

        <aside className="page-stack">
          <section className="panel invoice-side-panel"><p className="eyebrow">SOURCE</p><h2>Origen</h2><dl className="profile-definition-list"><div><dt>Tipo</dt><dd>{invoice.source_type}</dd></div>{invoice.quote_id && <div><dt>Cotización</dt><dd><Link href={`/quotes/${invoice.quote_id}`}>QT-{String(invoice.quote_number || 0).padStart(6, "0")}</Link></dd></div>}{invoice.reservation_id && <div><dt>Reserva</dt><dd><Link href={`/reservations/${invoice.reservation_id}`}>RS-{String(invoice.reservation_number || 0).padStart(6, "0")}</Link></dd></div>}<div><dt>Fiscal</dt><dd>{invoice.fiscal_status}</dd></div></dl></section>
          {canReadDTE && <section className="panel invoice-side-panel invoice-dte-panel"><p className="eyebrow">DTE</p><h2>Documento electrónico</h2>{dte ? <><span className={`dte-status-chip ${dte.status === "ACCEPTED" ? "accepted" : dte.status === "REJECTED" ? "rejected" : dte.status === "RETRY_REQUIRED" ? "retry" : "ready"}`}>{dte.status.replaceAll("_", " ")}</span><dl className="profile-definition-list"><div><dt>Control</dt><dd className="mono-copy">{dte.control_number}</dd></div><div><dt>Tipo</dt><dd>{dte.document_type_label}</dd></div><div><dt>Sello</dt><dd className="mono-copy">{dte.receipt_seal || "Pendiente"}</dd></div></dl><Link className="button button-secondary button-full" href={`/dte/${dte.id}`}>Abrir DTE</Link>{canManageDTE && dte.status === "REJECTED" && <><label className="field"><span>Documento de reemplazo</span><select value={documentType} onChange={(event) => setDocumentType(event.target.value as "AUTO" | "01" | "03")}><option value="AUTO">Automático</option><option value="01">01 · Factura</option><option value="03">03 · Crédito fiscal</option></select></label><button className="button button-primary button-full" type="button" disabled={working === "dte"} onClick={() => void prepareDTE()}>{working === "dte" ? "Preparando…" : "Preparar reemplazo"}</button></>}</> : invoice.status === "DRAFT" ? <p className="muted-copy">Emite la factura antes de preparar su documento electrónico.</p> : invoice.fiscal_status !== "READY_FOR_DTE" && invoice.fiscal_status !== "REJECTED" ? <><p className="muted-copy">El perfil fiscal no está listo para preparar un DTE.</p><Link href="/settings/billing">Completar perfil →</Link></> : <><label className="field"><span>Tipo de documento</span><select value={documentType} disabled={!canManageDTE} onChange={(event) => setDocumentType(event.target.value as "AUTO" | "01" | "03")}><option value="AUTO">Automático</option><option value="01">01 · Factura</option><option value="03">03 · Crédito fiscal</option></select></label>{canManageDTE && <button className="button button-primary button-full" type="button" disabled={working === "dte"} onClick={() => void prepareDTE()}>{working === "dte" ? "Preparando…" : "Preparar DTE"}</button>}<small className="field-hint">Preparar congela el JSON y reserva el número de control; todavía no transmite.</small></>}</section>}
          <section className="panel invoice-side-panel"><p className="eyebrow">PAYMENTS</p><h2>Aplicaciones</h2>{invoice.allocations.length === 0 ? <p className="muted-copy">Sin pagos aplicados.</p> : <div className="invoice-allocation-list">{invoice.allocations.map((item) => item.payment_id ? <Link key={item.id} href={`/payments/${item.payment_id}`}><span>{item.payment_display_number || "Pago aplicado"}</span><strong>{formatCurrency(item.amount, invoice.currency)}</strong></Link> : <div key={item.id}><span>{item.payment_display_number || "Pago aplicado"}</span><strong>{formatCurrency(item.amount, invoice.currency)}</strong></div>)}</div>}</section>
          <section className="panel invoice-side-panel"><p className="eyebrow">HISTORY</p><h2>Historial</h2><div className="invoice-event-list">{invoice.events.map((event) => <div key={event.id}><span /><p><strong>{event.event_type.replaceAll("_", " ")}</strong><small>{formatDateTime(event.created_at)}</small></p></div>)}</div></section>
        </aside>
      </div>

      {invoice.status === "VOID" && <section className="panel billing-warning-panel"><strong>Documento anulado</strong><p>{invoice.void_reason || "Sin motivo registrado."}</p></section>}
      <section className="architecture-note"><span>i</span><div><strong>Factura interna y DTE son registros distintos</strong><p>La factura mantiene la cuenta por cobrar. El documento electrónico conserva su JSON, firma, sello, reintentos e invalidación mediante el módulo fiscal.</p></div></section>
    </div>
  );
}
