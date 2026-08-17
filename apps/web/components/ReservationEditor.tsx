"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { AvailabilityResult, Customer, ReservationDetail, Resource } from "@/lib/types";

type DraftItem = {
  key: string;
  resource_id: string;
  description: string;
  quantity: string;
  unit_price: string;
  discount_amount: string;
};

function localInputFromDate(date: Date): string {
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

function initialDates() {
  const day = new Date();
  day.setDate(day.getDate() + 1);
  day.setSeconds(0, 0);

  const blockStart = new Date(day);
  blockStart.setHours(14, 0, 0, 0);
  const eventStart = new Date(day);
  eventStart.setHours(18, 0, 0, 0);
  const eventEnd = new Date(day);
  eventEnd.setHours(23, 0, 0, 0);
  const blockEnd = new Date(day);
  blockEnd.setDate(blockEnd.getDate() + 1);
  blockEnd.setHours(2, 0, 0, 0);

  return {
    block_start_at: localInputFromDate(blockStart),
    block_end_at: localInputFromDate(blockEnd),
    event_start_at: localInputFromDate(eventStart),
    event_end_at: localInputFromDate(eventEnd),
  };
}

function newItem(): DraftItem {
  return {
    key: typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`,
    resource_id: "",
    description: "",
    quantity: "1",
    unit_price: "0",
    discount_amount: "0",
  };
}

function asISO(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

function money(value: string): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? Math.max(0, Math.round(parsed * 100) / 100) : 0;
}

export function ReservationEditor() {
  const router = useRouter();
  const defaults = useMemo(initialDates, []);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [resources, setResources] = useState<Resource[]>([]);
  const [loadingOptions, setLoadingOptions] = useState(true);
  const [optionsError, setOptionsError] = useState("");
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [message, setMessage] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [availability, setAvailability] = useState<AvailabilityResult | null>(null);
  const [form, setForm] = useState({
    customer_id: "",
    ...defaults,
    event_type: "",
    event_location: "",
    discount_amount: "0",
    extra_charges: "0",
    notes: "",
  });
  const [items, setItems] = useState<DraftItem[]>([newItem()]);

  useEffect(() => {
    Promise.all([
      api<{ items: Customer[] }>("/api/v1/customers"),
      api<{ items: Resource[] }>("/api/v1/resources?active=true"),
    ])
      .then(([customerResponse, resourceResponse]) => {
        setCustomers(customerResponse.items);
        setResources(resourceResponse.items);
      })
      .catch((reason) => setOptionsError(reason instanceof Error ? reason.message : "No fue posible cargar clientes e inventario."))
      .finally(() => setLoadingOptions(false));
  }, []);

  const calculations = useMemo(() => {
    const lines = items.map((item) => {
      const quantity = Math.max(0, Number(item.quantity) || 0);
      const gross = quantity * money(item.unit_price);
      return Math.max(0, Math.round((gross - money(item.discount_amount)) * 100) / 100);
    });
    const subtotal = Math.round(lines.reduce((sum, value) => sum + value, 0) * 100) / 100;
    const reservationDiscount = money(form.discount_amount);
    const extraCharges = money(form.extra_charges);
    const total = Math.max(0, Math.round((subtotal - reservationDiscount + extraCharges) * 100) / 100);
    return { lines, subtotal, reservationDiscount, extraCharges, total };
  }, [form.discount_amount, form.extra_charges, items]);

  function invalidateAvailability() {
    setAvailability(null);
  }

  function updateForm(patch: Partial<typeof form>) {
    setForm((current) => ({ ...current, ...patch }));
    if (
      "block_start_at" in patch ||
      "block_end_at" in patch ||
      "event_start_at" in patch ||
      "event_end_at" in patch
    ) {
      invalidateAvailability();
    }
  }

  function updateItem(index: number, patch: Partial<DraftItem>) {
    setItems((current) => current.map((item, itemIndex) => (itemIndex === index ? { ...item, ...patch } : item)));
    invalidateAvailability();
  }

  function selectResource(index: number, resourceID: string) {
    const resource = resources.find((item) => item.id === resourceID);
    updateItem(index, {
      resource_id: resourceID,
      description: resource?.name || "",
      unit_price: resource ? String(resource.base_price) : "0",
      discount_amount: "0",
    });
  }

  function availabilityPayload() {
    return {
      start_at: asISO(form.block_start_at),
      end_at: asISO(form.block_end_at),
      items: items.map((item) => ({
        resource_id: item.resource_id,
        quantity: Number(item.quantity),
      })),
    };
  }

  async function checkAvailability() {
    setChecking(true);
    setMessage("");
    setFields({});
    try {
      const result = await api<AvailabilityResult>("/api/v1/availability/check", {
        method: "POST",
        body: JSON.stringify(availabilityPayload()),
      });
      setAvailability(result);
    } catch (error) {
      setAvailability(null);
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible verificar disponibilidad.");
      }
    } finally {
      setChecking(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    setFields({});
    try {
      const reservation = await api<ReservationDetail>("/api/v1/reservations", {
        method: "POST",
        body: JSON.stringify({
          customer_id: form.customer_id,
          block_start_at: asISO(form.block_start_at),
          block_end_at: asISO(form.block_end_at),
          event_start_at: asISO(form.event_start_at),
          event_end_at: asISO(form.event_end_at),
          event_type: form.event_type || null,
          event_location: form.event_location || null,
          discount_amount: money(form.discount_amount),
          extra_charges: money(form.extra_charges),
          notes: form.notes,
          items: items.map((item) => ({
            resource_id: item.resource_id,
            description: item.description,
            quantity: Number(item.quantity),
            unit_price: money(item.unit_price),
            discount_amount: money(item.discount_amount),
          })),
        }),
      });
      router.push(`/reservations/${reservation.id}`);
      router.refresh();
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
        const result = error.payload.availability;
        if (result && typeof result === "object") {
          setAvailability(result as AvailabilityResult);
        }
      } else {
        setMessage("No fue posible crear la reserva.");
      }
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setSaving(false);
    }
  }

  if (loadingOptions) return <div className="skeleton detail-skeleton" />;
  if (optionsError) return <div className="panel inline-error">{optionsError}</div>;

  return (
    <form onSubmit={submit} className="quote-editor-layout reservation-editor-layout">
      <div className="quote-editor-main page-stack">
        {message && <div className="form-alert">{message}</div>}

        <section className="panel quote-form-section">
          <div className="panel-header">
            <div>
              <p className="eyebrow">QUICK BOOKING</p>
              <h2>Cliente y evento</h2>
              <p>Registra una reserva recibida por llamada, WhatsApp o atención presencial sin crear primero una cotización.</p>
            </div>
          </div>
          <div className="quote-form-body">
            <div className="form-grid two-columns">
              <label className="field">
                <span>Cliente *</span>
                <select value={form.customer_id} onChange={(event) => updateForm({ customer_id: event.target.value })}>
                  <option value="">Seleccionar cliente</option>
                  {customers.map((customer) => (
                    <option value={customer.id} key={customer.id}>
                      {customer.display_name}{customer.company_name ? ` · ${customer.company_name}` : ""}
                    </option>
                  ))}
                </select>
                {fields.customer_id && <small className="field-error">{fields.customer_id}</small>}
                {customers.length === 0 && <small className="field-hint">No hay clientes. <Link href="/customers">Crea el primero</Link>.</small>}
              </label>
              <label className="field">
                <span>Tipo de evento</span>
                <input value={form.event_type} onChange={(event) => updateForm({ event_type: event.target.value })} placeholder="Boda, concierto, ensayo…" />
                {fields.event_type && <small className="field-error">{fields.event_type}</small>}
              </label>
              <label className="field form-grid-full">
                <span>Ubicación</span>
                <input value={form.event_location} onChange={(event) => updateForm({ event_location: event.target.value })} placeholder="Dirección, venue o punto de entrega" />
                {fields.event_location && <small className="field-error">{fields.event_location}</small>}
              </label>
            </div>
          </div>
        </section>

        <section className="panel quote-form-section">
          <div className="panel-header">
            <div>
              <p className="eyebrow">AGENDA OPERACIONAL</p>
              <h2>Bloqueo y horario del evento</h2>
              <p>El período bloqueado incluye preparación, transporte, montaje, desmontaje y retorno.</p>
            </div>
          </div>
          <div className="quote-form-body">
            <div className="schedule-editor-grid">
              <label className="field">
                <span>Inicio del bloqueo *</span>
                <input type="datetime-local" value={form.block_start_at} onChange={(event) => updateForm({ block_start_at: event.target.value })} />
                {fields.block_start_at && <small className="field-error">{fields.block_start_at}</small>}
              </label>
              <label className="field">
                <span>Fin del bloqueo *</span>
                <input type="datetime-local" value={form.block_end_at} onChange={(event) => updateForm({ block_end_at: event.target.value })} />
                {fields.block_end_at && <small className="field-error">{fields.block_end_at}</small>}
              </label>
              <label className="field">
                <span>Inicio del evento *</span>
                <input type="datetime-local" value={form.event_start_at} onChange={(event) => updateForm({ event_start_at: event.target.value })} />
                {fields.event_start_at && <small className="field-error">{fields.event_start_at}</small>}
              </label>
              <label className="field">
                <span>Fin del evento *</span>
                <input type="datetime-local" value={form.event_end_at} onChange={(event) => updateForm({ event_end_at: event.target.value })} />
                {fields.event_end_at && <small className="field-error">{fields.event_end_at}</small>}
              </label>
            </div>
          </div>
        </section>

        <section className="panel quote-form-section">
          <div className="panel-header quote-items-header">
            <div>
              <p className="eyebrow">INVENTARIO Y PRECIOS</p>
              <h2>Recursos reservados</h2>
              <p>RentStage validará las cantidades contra reservas superpuestas antes de guardar.</p>
            </div>
            <button type="button" className="button button-secondary button-small" onClick={() => { setItems((current) => [...current, newItem()]); invalidateAvailability(); }}>
              + Agregar recurso
            </button>
          </div>
          <div className="quote-lines">
            {items.map((item, index) => {
              const resource = resources.find((resourceItem) => resourceItem.id === item.resource_id);
              const availabilityLine = availability?.items.find((entry) => entry.resource_id === item.resource_id);
              return (
                <article className={`quote-line-card reservation-line-card ${availabilityLine ? (availabilityLine.can_fulfill ? "line-available" : "line-conflict") : ""}`} key={item.key}>
                  <div className="quote-line-number">{index + 1}</div>
                  <div className="quote-line-fields">
                    <div className="form-grid quote-line-grid">
                      <label className="field quote-resource-field">
                        <span>Recurso *</span>
                        <select value={item.resource_id} onChange={(event) => selectResource(index, event.target.value)}>
                          <option value="">Seleccionar del inventario</option>
                          {resources.map((resourceOption) => (
                            <option value={resourceOption.id} key={resourceOption.id}>
                              {resourceOption.name} · {formatCurrency(resourceOption.base_price)}/{pricingUnitLabel(resourceOption.pricing_unit)}
                            </option>
                          ))}
                        </select>
                        {fields[`items[${index}].resource_id`] && <small className="field-error">{fields[`items[${index}].resource_id`]}</small>}
                        {resource && !availabilityLine && <small className="field-hint">{resource.available_asset_count} de {resource.asset_count} unidades físicamente disponibles ahora</small>}
                        {availabilityLine && (
                          <small className={availabilityLine.can_fulfill ? "availability-inline-ok" : "availability-inline-error"}>
                            {availabilityLine.can_fulfill
                              ? `${availabilityLine.available_quantity} disponibles para el período`
                              : `Solicitadas ${availabilityLine.requested_quantity}; disponibles ${availabilityLine.available_quantity}`}
                          </small>
                        )}
                      </label>
                      <label className="field">
                        <span>Cantidad *</span>
                        <input type="number" min="1" step="1" value={item.quantity} onChange={(event) => updateItem(index, { quantity: event.target.value })} />
                        {fields[`items[${index}].quantity`] && <small className="field-error">{fields[`items[${index}].quantity`]}</small>}
                      </label>
                      <label className="field">
                        <span>Precio unitario</span>
                        <div className="input-prefix"><span>$</span><input type="number" min="0" step="0.01" value={item.unit_price} onChange={(event) => updateItem(index, { unit_price: event.target.value })} /></div>
                        {fields[`items[${index}].unit_price`] && <small className="field-error">{fields[`items[${index}].unit_price`]}</small>}
                      </label>
                      <label className="field">
                        <span>Descuento línea</span>
                        <div className="input-prefix"><span>$</span><input type="number" min="0" step="0.01" value={item.discount_amount} onChange={(event) => updateItem(index, { discount_amount: event.target.value })} /></div>
                        {fields[`items[${index}].discount_amount`] && <small className="field-error">{fields[`items[${index}].discount_amount`]}</small>}
                      </label>
                    </div>
                    <div className="quote-line-bottom">
                      <label className="field quote-description-field">
                        <span>Descripción</span>
                        <input value={item.description} onChange={(event) => updateItem(index, { description: event.target.value })} placeholder="Se usará el nombre del recurso si se deja vacío" />
                      </label>
                      <div className="quote-line-total"><span>Total línea</span><strong>{formatCurrency(calculations.lines[index] || 0)}</strong></div>
                      <button type="button" className="icon-action quote-remove-line" onClick={() => { setItems((current) => current.filter((_, itemIndex) => itemIndex !== index)); invalidateAvailability(); }} disabled={items.length === 1} title="Eliminar línea">×</button>
                    </div>
                  </div>
                </article>
              );
            })}
            {fields.items && <div className="quote-items-error">{fields.items}</div>}
          </div>
        </section>

        <section className="panel quote-form-section">
          <div className="panel-header"><div><p className="eyebrow">OPERACIÓN</p><h2>Notas de la reserva</h2></div></div>
          <div className="quote-form-body">
            <label className="field">
              <span>Notas internas</span>
              <textarea rows={5} value={form.notes} onChange={(event) => updateForm({ notes: event.target.value })} placeholder="Incluye entrega, instalación, contacto del venue o instrucciones especiales." />
              {fields.notes && <small className="field-error">{fields.notes}</small>}
            </label>
          </div>
        </section>
      </div>

      <aside className="quote-summary-panel panel reservation-summary-panel">
        <div className="quote-summary-header">
          <p className="eyebrow">RESERVA MANUAL</p>
          <h2>Resumen operativo</h2>
          <p>Al guardar, esta reserva quedará pendiente y bloqueará inventario durante el período indicado.</p>
        </div>

        <div className="availability-summary-card">
          <div className="availability-summary-heading">
            <span className={`availability-indicator ${availability ? (availability.available ? "is-available" : "has-conflict") : "is-pending"}`} />
            <div>
              <strong>{availability ? (availability.available ? "Inventario disponible" : "Conflicto de inventario") : "Disponibilidad sin validar"}</strong>
              <small>{availability ? `${availability.items.length} recursos verificados` : "Comprueba el período antes de confirmar"}</small>
            </div>
          </div>
          {availability && (
            <div className="availability-summary-lines">
              {availability.items.map((item) => (
                <div key={item.resource_id}>
                  <span>{item.resource_name}</span>
                  <strong className={item.can_fulfill ? "availability-ok" : "availability-fail"}>{item.available_quantity}/{item.requested_quantity}</strong>
                </div>
              ))}
            </div>
          )}
          <button type="button" className="button button-secondary button-full" disabled={checking} onClick={() => void checkAvailability()}>
            {checking ? "Verificando…" : "Verificar disponibilidad"}
          </button>
        </div>

        <div className="quote-summary-money-fields">
          <label className="field">
            <span>Descuento general</span>
            <div className="input-prefix"><span>$</span><input type="number" min="0" step="0.01" value={form.discount_amount} onChange={(event) => updateForm({ discount_amount: event.target.value })} /></div>
            {fields.discount_amount && <small className="field-error">{fields.discount_amount}</small>}
          </label>
          <label className="field">
            <span>Cargos adicionales</span>
            <div className="input-prefix"><span>$</span><input type="number" min="0" step="0.01" value={form.extra_charges} onChange={(event) => updateForm({ extra_charges: event.target.value })} /></div>
            {fields.extra_charges && <small className="field-error">{fields.extra_charges}</small>}
          </label>
        </div>
        <div className="quote-total-breakdown">
          <div><span>Subtotal</span><strong>{formatCurrency(calculations.subtotal)}</strong></div>
          <div><span>Descuento</span><strong>− {formatCurrency(calculations.reservationDiscount)}</strong></div>
          <div><span>Cargos adicionales</span><strong>{formatCurrency(calculations.extraCharges)}</strong></div>
          <div className="quote-grand-total"><span>Total</span><strong>{formatCurrency(calculations.total)}</strong></div>
        </div>
        <div className="quote-summary-actions">
          <Link href="/reservations" className="button button-secondary">Cancelar</Link>
          <button className="button button-primary" type="submit" disabled={saving || customers.length === 0 || resources.length === 0}>
            {saving ? "Creando…" : "Crear reserva"}
          </button>
        </div>
      </aside>
    </form>
  );
}
