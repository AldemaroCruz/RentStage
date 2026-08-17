"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { formatCurrency, formatDate } from "@/lib/format";
import type { InvoiceDetail } from "@/lib/types";

export default function InvoicePrintPage() {
  const params = useParams<{ id: string }>();
  const [invoice, setInvoice] = useState<InvoiceDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<InvoiceDetail>(`/api/v1/invoices/${params.id}`)
      .then(setInvoice)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el documento."));
  }, [params.id]);

  if (error) return <div className="panel inline-error">{error}</div>;
  if (!invoice) return <div className="skeleton detail-skeleton" />;

  return (
    <div className="invoice-print-page">
      <div className="invoice-print-actions"><button className="button button-primary" onClick={() => window.print()}>Imprimir / Guardar PDF</button><button className="button button-secondary" onClick={() => window.close()}>Cerrar</button></div>
      <article className="invoice-print-sheet">
        <header><div><p>RENTSTAGE · DOCUMENTO INTERNO</p><h1>{invoice.display_number}</h1><span>{invoice.seller_trade_name || invoice.seller_legal_name || "RentStage"}</span></div><div><strong>{formatCurrency(invoice.total_amount, invoice.currency)}</strong><span>{invoice.status === "VOID" ? "ANULADA" : invoice.display_status}</span></div></header>
        <section className="invoice-print-parties"><div><small>EMISOR</small><h2>{invoice.seller_legal_name || "Perfil pendiente"}</h2><p>{invoice.seller_tax_id ? `NIT: ${invoice.seller_tax_id}` : "NIT pendiente"}<br />{invoice.seller_registration_number ? `NRC: ${invoice.seller_registration_number}` : "NRC pendiente"}<br />{invoice.seller_address}<br />{invoice.seller_email} {invoice.seller_phone}</p></div><div><small>CLIENTE</small><h2>{invoice.customer_name}</h2><p>{invoice.customer_tax_id ? `ID fiscal: ${invoice.customer_tax_id}` : "Sin ID fiscal"}<br />{invoice.customer_address}<br />{invoice.customer_email} {invoice.customer_phone}</p></div></section>
        <section className="invoice-print-dates"><span>Fecha de emisión<strong>{formatDate(invoice.issue_date)}</strong></span><span>Fecha de vencimiento<strong>{formatDate(invoice.due_date)}</strong></span><span>Moneda<strong>{invoice.currency}</strong></span><span>Origen<strong>{invoice.source_type}</strong></span></section>
        <table><thead><tr><th>Descripción</th><th>Cantidad</th><th>Precio</th><th>Descuento</th><th>IVA</th><th>Total</th></tr></thead><tbody>{invoice.items.map((item) => <tr key={item.id}><td>{item.description}<small>{item.tax_category === "TAXABLE" ? `Gravada ${item.tax_rate}%` : item.tax_category === "EXEMPT" ? "Exenta" : "No sujeta"}</small></td><td>{item.quantity}</td><td>{formatCurrency(item.unit_price, invoice.currency)}</td><td>{formatCurrency(item.discount_amount, invoice.currency)}</td><td>{formatCurrency(item.tax_amount, invoice.currency)}</td><td>{formatCurrency(item.line_total, invoice.currency)}</td></tr>)}</tbody></table>
        <section className="invoice-print-summary"><div><h3>Notas</h3><p>{invoice.notes || "—"}</p><h3>Términos</h3><p>{invoice.terms || "—"}</p></div><dl><div><dt>Gravado</dt><dd>{formatCurrency(invoice.taxable_amount, invoice.currency)}</dd></div><div><dt>Exento</dt><dd>{formatCurrency(invoice.exempt_amount, invoice.currency)}</dd></div><div><dt>No sujeto</dt><dd>{formatCurrency(invoice.non_taxable_amount, invoice.currency)}</dd></div><div><dt>IVA</dt><dd>{formatCurrency(invoice.tax_amount, invoice.currency)}</dd></div><div><dt>Total</dt><dd>{formatCurrency(invoice.total_amount, invoice.currency)}</dd></div><div><dt>Pagado</dt><dd>{formatCurrency(invoice.paid_amount, invoice.currency)}</dd></div><div className="final"><dt>Saldo</dt><dd>{formatCurrency(invoice.balance_due, invoice.currency)}</dd></div></dl></section>
        <footer>Representación interna generada por RentStage. Consulta el módulo DTE para verificar JSON, proveedor y sello de recepción.</footer>
      </article>
    </div>
  );
}
