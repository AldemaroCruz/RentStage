"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { formatCurrency, formatQuoteNumber, formatReservationNumber, toLocalDateInput } from "@/lib/format";
import type { BillingSettings, Customer, InvoiceDetail, QuoteSummary, ReservationSummary, TaxRule } from "@/lib/types";

type DraftLine = { description: string; quantity: number; unit_price: number; discount_amount: number; tax_rule_id: string };

function todayInput() {
  return toLocalDateInput(new Date());
}

export default function NewInvoicePage() {
  const { me } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [settings, setSettings] = useState<BillingSettings | null>(null);
  const [rules, setRules] = useState<TaxRule[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [quotes, setQuotes] = useState<QuoteSummary[]>([]);
  const [reservations, setReservations] = useState<ReservationSummary[]>([]);
  const initialSource = (searchParams.get("source_type") || "MANUAL").toUpperCase();
  const [sourceType, setSourceType] = useState<"MANUAL" | "QUOTE" | "RESERVATION">(initialSource === "QUOTE" || initialSource === "RESERVATION" ? initialSource : "MANUAL");
  const [sourceID, setSourceID] = useState(searchParams.get("source_id") || "");
  const [customerID, setCustomerID] = useState(searchParams.get("customer_id") || "");
  const [issueDate, setIssueDate] = useState(todayInput());
  const [dueDate, setDueDate] = useState(todayInput());
  const [currency, setCurrency] = useState("USD");
  const [notes, setNotes] = useState("");
  const [terms, setTerms] = useState("Gracias por confiar en nosotros. El pago se aplicará según las condiciones acordadas.");
  const [lines, setLines] = useState<DraftLine[]>([{ description: "", quantity: 1, unit_price: 0, discount_amount: 0, tax_rule_id: "" }]);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.all([
      api<BillingSettings>("/api/v1/billing/settings"),
      api<{ items: TaxRule[] }>("/api/v1/billing/tax-rules"),
      api<{ items: Customer[] }>("/api/v1/customers"),
      api<{ items: QuoteSummary[] }>("/api/v1/quotes?status=ACCEPTED"),
      api<{ items: ReservationSummary[] }>("/api/v1/reservations"),
    ]).then(([billing, taxRules, customerList, quoteList, reservationList]) => {
      setSettings(billing);
      setRules(taxRules.items);
      setCustomers(customerList.items);
      setQuotes(quoteList.items);
      setReservations(reservationList.items.filter((item) => item.status !== "CANCELLED"));
      setCurrency(me?.active_workspace?.currency || "USD");
      const due = new Date();
      due.setDate(due.getDate() + billing.default_payment_terms_days);
      setDueDate(toLocalDateInput(due));
      const defaultRule = taxRules.items.find((item) => item.is_default) || taxRules.items[0];
      if (defaultRule) setLines((current) => current.map((item) => ({ ...item, tax_rule_id: defaultRule.id })));
      if (!customerID && customerList.items.length === 1) setCustomerID(customerList.items[0].id);
    }).catch((reason) => setMessage(reason instanceof Error ? reason.message : "No fue posible preparar la factura."))
      .finally(() => setLoading(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (sourceType === "QUOTE") {
      const selected = quotes.find((item) => item.id === sourceID);
      if (selected) setCustomerID(selected.customer_id);
    }
    if (sourceType === "RESERVATION") {
      const selected = reservations.find((item) => item.id === sourceID);
      if (selected) setCustomerID(selected.customer_id);
    }
  }, [sourceType, sourceID, quotes, reservations]);

  const preview = useMemo(() => {
    if (sourceType !== "MANUAL") return 0;
    return lines.reduce((sum, line) => sum + Math.max(0, line.quantity * line.unit_price - line.discount_amount), 0);
  }, [sourceType, lines]);

  function updateLine(index: number, patch: Partial<DraftLine>) {
    setLines((current) => current.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item));
  }

  function addLine() {
    const defaultRule = rules.find((item) => item.is_default) || rules[0];
    setLines((current) => [...current, { description: "", quantity: 1, unit_price: 0, discount_amount: 0, tax_rule_id: defaultRule?.id || "" }]);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setFields({});
    setMessage("");
    try {
      const created = await api<InvoiceDetail>("/api/v1/invoices", {
        method: "POST",
        body: JSON.stringify({
          source_type: sourceType,
          source_id: sourceType === "MANUAL" ? null : sourceID,
          customer_id: customerID,
          issue_date: issueDate,
          due_date: dueDate,
          currency,
          notes,
          terms,
          items: sourceType === "MANUAL" ? lines : [],
        }),
      });
      router.push(`/invoices/${created.id}`);
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible crear la factura.");
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="skeleton detail-skeleton" />;

  return (
    <div className="page-stack invoice-editor-page">
      <div className="breadcrumbs"><Link href="/invoices">Facturas</Link><span>/</span><span>Nueva</span></div>
      <section className="page-heading"><div><p className="eyebrow">NEW INTERNAL INVOICE</p><h2>Nueva factura</h2><p>Usa un origen comercial existente o crea un documento manual con cálculo de impuestos por línea.</p></div></section>
      {message && <div className="form-alert">{message}</div>}

      <form className="invoice-editor-grid" onSubmit={submit}>
        <div className="page-stack">
          <section className="panel invoice-editor-section">
            <div className="panel-header"><div><p className="eyebrow">SOURCE</p><h2>Origen del documento</h2></div></div>
            <div className="invoice-source-selector">
              {(["MANUAL", "QUOTE", "RESERVATION"] as const).map((type) => <button key={type} type="button" className={sourceType === type ? "active" : ""} onClick={() => { setSourceType(type); setSourceID(""); }}>{type === "MANUAL" ? "Manual" : type === "QUOTE" ? "Cotización aceptada" : "Reserva"}</button>)}
            </div>
            {sourceType === "QUOTE" && <label className="field"><span>Cotización aceptada *</span><select value={sourceID} onChange={(event) => setSourceID(event.target.value)}><option value="">Seleccionar…</option>{quotes.map((item) => <option key={item.id} value={item.id}>{formatQuoteNumber(item.quote_number)} · {item.customer_name} · {formatCurrency(item.total)}</option>)}</select>{fields.source_id && <small className="field-error">{fields.source_id}</small>}</label>}
            {sourceType === "RESERVATION" && <label className="field"><span>Reserva *</span><select value={sourceID} onChange={(event) => setSourceID(event.target.value)}><option value="">Seleccionar…</option>{reservations.map((item) => <option key={item.id} value={item.id}>{formatReservationNumber(item.reservation_number)} · {item.customer_name} · {formatCurrency(item.total)}</option>)}</select>{fields.source_id && <small className="field-error">{fields.source_id}</small>}</label>}
            <label className="field"><span>Cliente *</span><select value={customerID} disabled={sourceType !== "MANUAL"} onChange={(event) => setCustomerID(event.target.value)}><option value="">Seleccionar…</option>{customers.map((item) => <option key={item.id} value={item.id}>{item.display_name}{item.company_name ? ` · ${item.company_name}` : ""}</option>)}</select>{fields.customer_id && <small className="field-error">{fields.customer_id}</small>}</label>
          </section>

          {sourceType === "MANUAL" && <section className="panel invoice-editor-section">
            <div className="panel-header"><div><p className="eyebrow">LINES</p><h2>Conceptos</h2><p>Clasifica cada línea como gravada, exenta o no sujeta.</p></div><button type="button" className="button button-secondary" onClick={addLine}>+ Agregar línea</button></div>
            <div className="invoice-line-editor-list">{lines.map((line, index) => <article key={index} className="invoice-line-editor">
              <div className="invoice-line-editor-head"><strong>Línea {index + 1}</strong>{lines.length > 1 && <button type="button" onClick={() => setLines((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Eliminar</button>}</div>
              <label className="field invoice-line-description"><span>Descripción</span><input value={line.description} onChange={(event) => updateLine(index, { description: event.target.value })} />{fields[`items[${index}].description`] && <small className="field-error">{fields[`items[${index}].description`]}</small>}</label>
              <div className="form-grid four-columns">
                <label className="field"><span>Cantidad</span><input type="number" min="0.001" step="0.001" value={line.quantity} onChange={(event) => updateLine(index, { quantity: Number(event.target.value) })} /></label>
                <label className="field"><span>Precio unitario</span><input type="number" min="0" step="0.01" value={line.unit_price} onChange={(event) => updateLine(index, { unit_price: Number(event.target.value) })} /></label>
                <label className="field"><span>Descuento</span><input type="number" min="0" step="0.01" value={line.discount_amount} onChange={(event) => updateLine(index, { discount_amount: Number(event.target.value) })} /></label>
                <label className="field"><span>Impuesto</span><select value={line.tax_rule_id} onChange={(event) => updateLine(index, { tax_rule_id: event.target.value })}>{rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name} · {rule.rate}%</option>)}</select></label>
              </div>
            </article>)}</div>
          </section>}

          <section className="panel invoice-editor-section">
            <div className="panel-header"><div><p className="eyebrow">CONDITIONS</p><h2>Notas y términos</h2></div></div>
            <label className="field"><span>Notas internas / visibles</span><textarea rows={3} value={notes} onChange={(event) => setNotes(event.target.value)} /></label>
            <label className="field"><span>Términos</span><textarea rows={4} value={terms} onChange={(event) => setTerms(event.target.value)} /></label>
          </section>
        </div>

        <aside className="panel invoice-editor-summary">
          <p className="eyebrow">DOCUMENT SUMMARY</p><h2>Resumen</h2>
          <div className="form-grid two-columns"><label className="field"><span>Emisión</span><input type="date" value={issueDate} onChange={(event) => setIssueDate(event.target.value)} /></label><label className="field"><span>Vencimiento</span><input type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} /></label></div>
          <label className="field"><span>Moneda</span><input value={currency} onChange={(event) => setCurrency(event.target.value.toUpperCase())} maxLength={3} /></label>
          <div className="invoice-editor-preview"><span>Origen</span><strong>{sourceType === "MANUAL" ? "Manual" : sourceType === "QUOTE" ? "Cotización" : "Reserva"}</strong><span>Precios</span><strong>{settings?.prices_include_tax ? "Incluyen IVA" : "IVA separado"}</strong>{sourceType === "MANUAL" && <><span>Subtotal preliminar</span><strong>{formatCurrency(preview, currency)}</strong></>}</div>
          {fields.items && <small className="field-error">{fields.items}</small>}
          <button className="button button-primary button-full" type="submit" disabled={saving}>{saving ? "Creando…" : "Crear borrador"}</button>
          <Link className="button button-secondary button-full" href="/invoices">Cancelar</Link>
          <small className="invoice-editor-help">El número definitivo se asigna al emitir. Crear el borrador no genera un DTE ni bloquea inventario.</small>
        </aside>
      </form>
    </div>
  );
}
