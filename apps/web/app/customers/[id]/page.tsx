"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { CustomerForm } from "@/components/CustomerForm";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { Modal } from "@/components/Modal";
import { QuoteStatusBadge } from "@/components/QuoteStatusBadge";
import { api } from "@/lib/api";
import { customerSourceLabel, formatCurrency, formatDate, formatDateTime, formatQuoteNumber } from "@/lib/format";
import type { Customer, QuoteSummary } from "@/lib/types";

export default function CustomerDetailPage() {
  const { can } = useAuth();
  const canManageCustomer = can("customer.manage");
  const canManageQuotes = can("quote.manage");
  const params = useParams<{ id: string }>();
  const [customer, setCustomer] = useState<Customer | null>(null);
  const [quotes, setQuotes] = useState<QuoteSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editOpen, setEditOpen] = useState(false);

  useEffect(() => {
    Promise.all([
      api<Customer>(`/api/v1/customers/${params.id}`),
      api<{ items: QuoteSummary[] }>(`/api/v1/quotes?customer_id=${params.id}`),
    ])
      .then(([customerResponse, quoteResponse]) => {
        setCustomer(customerResponse);
        setQuotes(quoteResponse.items);
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el cliente."))
      .finally(() => setLoading(false));
  }, [params.id]);

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error || !customer) return <div className="panel inline-error">{error || "Cliente no encontrado."}</div>;

  return (
    <div className="page-stack">
      <div className="breadcrumbs"><Link href="/customers">Clientes</Link><span>/</span><span>{customer.display_name}</span></div>

      <section className="customer-hero panel">
        <div className="customer-avatar-large">{customer.display_name.slice(0, 2).toUpperCase()}</div>
        <div className="customer-hero-copy">
          <p className="eyebrow">CUSTOMER PROFILE</p>
          <h2>{customer.display_name}</h2>
          <p>{customer.company_name || "Cliente particular"}</p>
          <div className="customer-contact-row">
            <span>{customer.phone || "Sin teléfono"}</span>
            <span>{customer.email || "Sin correo"}</span>
            <span>{customerSourceLabel(customer.source)}</span>
          </div>
        </div>
        <div className="customer-hero-actions">
          {canManageQuotes && <Link className="button button-secondary" href={`/quotes/new?customer_id=${customer.id}`}>Nueva cotización</Link>}
          {canManageCustomer && <button className="button button-primary" onClick={() => setEditOpen(true)}>Editar cliente</button>}
        </div>
      </section>

      <section className="customer-metric-grid">
        <article><span>Cotizaciones</span><strong>{customer.quote_count}</strong><small>Históricas</small></article>
        <article><span>Aceptadas</span><strong>{customer.accepted_quote_count}</strong><small>Conversiones registradas</small></article>
        <article><span>Ventas aceptadas</span><strong>{formatCurrency(customer.accepted_quote_revenue)}</strong><small>Valor histórico</small></article>
        <article><span>Cliente desde</span><strong className="metric-date">{formatDate(customer.created_at)}</strong><small>Última edición {formatDate(customer.updated_at)}</small></article>
      </section>

      <div className="customer-detail-grid">
        <section className="panel customer-notes-panel">
          <div className="panel-header"><div><p className="eyebrow">CONTEXTO</p><h2>Notas del cliente</h2></div></div>
          <div className="customer-notes-body">{customer.notes || "No hay notas registradas."}</div>
        </section>
        <section className="panel customer-profile-panel">
          <div className="panel-header"><div><p className="eyebrow">PREFERENCIAS</p><h2>Perfil comercial</h2></div></div>
          <dl className="profile-definition-list">
            <div><dt>Idioma</dt><dd>{customer.preferred_language === "en" ? "English" : "Español"}</dd></div>
            <div><dt>Origen</dt><dd>{customerSourceLabel(customer.source)}</dd></div>
            <div><dt>Empresa</dt><dd>{customer.company_name || "—"}</dd></div>
            <div><dt>NIT / ID fiscal</dt><dd>{customer.tax_id || "—"}</dd></div>
            <div><dt>NRC / registro</dt><dd>{customer.tax_registration_number || "—"}</dd></div>
            <div><dt>Dirección de facturación</dt><dd>{customer.billing_address || "—"}</dd></div>
            <div><dt>Tipo de documento DTE</dt><dd>{customer.document_type_code || "36"}</dd></div>
            <div><dt>Nombre comercial DTE</dt><dd>{customer.trade_name || "—"}</dd></div>
            <div><dt>Actividad económica</dt><dd>{customer.economic_activity_code ? `${customer.economic_activity_code} · ${customer.economic_activity || "Sin descripción"}` : "—"}</dd></div>
            <div><dt>Códigos geográficos</dt><dd>{[customer.department_code, customer.municipality_code, customer.district_code].filter(Boolean).join(" / ") || "—"}</dd></div>
            <div><dt>Última cotización</dt><dd>{formatDateTime(customer.last_quote_at)}</dd></div>
          </dl>
        </section>
      </div>

      <section className="panel">
        <div className="panel-header">
          <div><p className="eyebrow">HISTORIAL COMERCIAL</p><h2>Cotizaciones</h2><p>Todos los documentos asociados a este cliente.</p></div>
          {canManageQuotes && <Link href={`/quotes/new?customer_id=${customer.id}`} className="button button-secondary">Crear cotización</Link>}
        </div>
        {quotes.length === 0 ? (
          <EmptyState icon="▤" title="Sin cotizaciones" description="Crea la primera cotización para este cliente." />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table">
              <thead><tr><th>Número</th><th>Evento</th><th>Período</th><th>Estado</th><th>Total</th><th /></tr></thead>
              <tbody>
                {quotes.map((quote) => (
                  <tr key={quote.id}>
                    <td><strong className="mono-copy">{formatQuoteNumber(quote.quote_number)}</strong></td>
                    <td><strong className="table-primary-copy">{quote.event_type || "Sin tipo"}</strong><span className="table-subline">{quote.event_location || "Sin ubicación"}</span></td>
                    <td>{formatDateTime(quote.start_at)}<span className="table-subline">hasta {formatDateTime(quote.end_at)}</span></td>
                    <td><QuoteStatusBadge status={quote.status} /></td>
                    <td><strong className="table-primary-copy">{formatCurrency(quote.total)}</strong></td>
                    <td><div className="row-actions"><Link className="icon-action" href={`/quotes/${quote.id}`}>→</Link></div></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Modal open={editOpen} title="Editar cliente" eyebrow="CUSTOMER CORE" onClose={() => setEditOpen(false)} width="720px">
        <CustomerForm
          initial={customer}
          onCancel={() => setEditOpen(false)}
          onSaved={(updated) => {
            setCustomer(updated);
            setEditOpen(false);
          }}
        />
      </Modal>
    </div>
  );
}
