"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { CustomerForm } from "@/components/CustomerForm";
import { useAuth } from "@/components/AuthProvider";
import { EmptyState } from "@/components/EmptyState";
import { Modal } from "@/components/Modal";
import { api } from "@/lib/api";
import { customerSourceLabel, formatCurrency, formatDate } from "@/lib/format";
import type { Customer, CustomerSource } from "@/lib/types";

export default function CustomersPage() {
  const { can } = useAuth();
  const canManage = can("customer.manage");
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [search, setSearch] = useState("");
  const [source, setSource] = useState<"" | CustomerSource>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setLoading(true);
      const params = new URLSearchParams();
      if (search.trim()) params.set("q", search.trim());
      if (source) params.set("source", source);
      api<{ items: Customer[] }>(`/api/v1/customers?${params.toString()}`)
        .then((response) => {
          setCustomers(response.items);
          setError("");
        })
        .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar los clientes."))
        .finally(() => setLoading(false));
    }, 220);
    return () => window.clearTimeout(timer);
  }, [search, source]);

  return (
    <div className="page-stack">
      <section className="page-heading">
        <div>
          <p className="eyebrow">CUSTOMER CORE</p>
          <h2>Clientes</h2>
          <p>Centraliza contactos, origen comercial y el historial que luego utilizará el agente de WhatsApp.</p>
        </div>
        {canManage && <button className="button button-primary" onClick={() => setOpen(true)}>
          <span className="button-plus">+</span> Nuevo cliente
        </button>}
      </section>

      <section className="panel inventory-panel">
        <div className="inventory-toolbar">
          <label className="search-box">
            <span aria-hidden="true">⌕</span>
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Buscar por nombre, empresa, teléfono o correo"
            />
          </label>
          <select value={source} onChange={(event) => setSource(event.target.value as "" | CustomerSource)}>
            <option value="">Todos los orígenes</option>
            <option value="MANUAL">Manual</option>
            <option value="WHATSAPP">WhatsApp</option>
            <option value="WEB">Web</option>
            <option value="IMPORT">Importado</option>
          </select>
          <span className="toolbar-count">{customers.length} clientes</span>
        </div>

        {loading ? (
          <div className="table-skeleton">Cargando clientes…</div>
        ) : error ? (
          <div className="inline-error">{error}</div>
        ) : customers.length === 0 ? (
          <EmptyState
            icon="◎"
            title="No encontramos clientes"
            description={search || source ? "Prueba con otros filtros." : "Crea el primer cliente para comenzar a generar cotizaciones."}
            action={!search && !source && canManage ? <button className="button button-primary" onClick={() => setOpen(true)}>Crear cliente</button> : undefined}
          />
        ) : (
          <div className="data-table-wrap">
            <table className="data-table customers-table">
              <thead>
                <tr>
                  <th>Cliente</th>
                  <th>Contacto</th>
                  <th>Origen</th>
                  <th>Cotizaciones</th>
                  <th>Ventas aceptadas</th>
                  <th>Última actividad</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {customers.map((customer) => (
                  <tr key={customer.id}>
                    <td>
                      <div className="table-resource-name">
                        <span>{customer.display_name.slice(0, 2).toUpperCase()}</span>
                        <div>
                          <strong>{customer.display_name}</strong>
                          <small>{customer.company_name || "Cliente particular"}</small>
                        </div>
                      </div>
                    </td>
                    <td>
                      <strong className="table-primary-copy">{customer.phone || "Sin teléfono"}</strong>
                      <span className="table-subline">{customer.email || "Sin correo"}</span>
                    </td>
                    <td><span className="category-pill">{customerSourceLabel(customer.source)}</span></td>
                    <td><strong className="table-primary-copy">{customer.quote_count}</strong><span className="table-subline">{customer.accepted_quote_count} aceptadas</span></td>
                    <td><strong className="table-primary-copy">{formatCurrency(customer.accepted_quote_revenue)}</strong></td>
                    <td>{formatDate(customer.last_quote_at || customer.created_at)}</td>
                    <td>
                      <div className="row-actions">
                        <Link className="icon-action" href={`/customers/${customer.id}`} title="Ver cliente">→</Link>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="architecture-note">
        <span>i</span>
        <div>
          <strong>Teléfono preparado para WhatsApp</strong>
          <p>RentStage guarda el número en formato E.164, por ejemplo +50371234567, para asociarlo de forma consistente con futuras conversaciones de WhatsApp.</p>
        </div>
      </section>

      <Modal open={open} title="Nuevo cliente" eyebrow="CUSTOMER CORE" onClose={() => setOpen(false)} width="720px">
        <CustomerForm
          onCancel={() => setOpen(false)}
          onSaved={(customer) => {
            setCustomers((items) => [customer, ...items]);
            setOpen(false);
          }}
        />
      </Modal>
    </div>
  );
}
