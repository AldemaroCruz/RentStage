"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { ApiError, api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel, toLocalDateTimeInput } from "@/lib/format";
import type { Customer, PackageAvailabilityResult, PackageQuoteTemplate, QuoteDetail, RentalPackageSummary, Resource } from "@/lib/types";

type DraftItem = {
  key: string;
  resource_id: string;
  description: string;
  quantity: string;
  unit_price: string;
  discount_amount: string;
};

type Props = {
  initial?: QuoteDetail;
  presetCustomerID?: string;
  presetPackageID?: string;
};

type AppliedPackage = {
  id: string;
  name: string;
  quantity: number;
  effectivePrice: number;
};

function localInputFromDate(date: Date): string {
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

function initialDates() {
  const start = new Date();
  start.setDate(start.getDate() + 1);
  start.setHours(14, 0, 0, 0);
  const end = new Date(start);
  end.setHours(23, 0, 0, 0);
  return { start: localInputFromDate(start), end: localInputFromDate(end) };
}

function draftKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`;
}

function newItem(): DraftItem {
  return {
    key: draftKey(),
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

export function QuoteEditor({ initial, presetCustomerID, presetPackageID }: Props) {
  const router = useRouter();
  const { can } = useAuth();
  const defaults = useMemo(initialDates, []);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [resources, setResources] = useState<Resource[]>([]);
  const [packages, setPackages] = useState<RentalPackageSummary[]>([]);
  const [selectedPackageID, setSelectedPackageID] = useState(presetPackageID || "");
  const [packageSets, setPackageSets] = useState("1");
  const [applyingPackage, setApplyingPackage] = useState(false);
  const [packageMessage, setPackageMessage] = useState("");
  const [packageAvailability, setPackageAvailability] = useState<PackageAvailabilityResult | null>(null);
  const [appliedPackages, setAppliedPackages] = useState<AppliedPackage[]>([]);
  const [loadingOptions, setLoadingOptions] = useState(true);
  const [optionsError, setOptionsError] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [form, setForm] = useState({
    customer_id: initial?.customer_id || presetCustomerID || "",
    start_at: initial ? toLocalDateTimeInput(initial.start_at) : defaults.start,
    end_at: initial ? toLocalDateTimeInput(initial.end_at) : defaults.end,
    event_type: initial?.event_type || "",
    event_location: initial?.event_location || "",
    discount_amount: initial?.discount_amount?.toString() || "0",
    extra_charges: initial?.extra_charges?.toString() || "0",
    expires_at: initial?.expires_at ? toLocalDateTimeInput(initial.expires_at) : "",
    notes: initial?.notes || "",
  });
  const [items, setItems] = useState<DraftItem[]>(
    initial?.items?.length
      ? initial.items.map((item) => ({
          key: item.id,
          resource_id: item.resource_id,
          description: item.description,
          quantity: String(item.quantity),
          unit_price: String(item.unit_price),
          discount_amount: String(item.discount_amount),
        }))
      : [newItem()],
  );

  useEffect(() => {
    Promise.all([
      api<{ items: Customer[] }>("/api/v1/customers"),
      api<{ items: Resource[] }>("/api/v1/resources?active=true"),
      api<{ items: RentalPackageSummary[] }>("/api/v1/packages?active=true"),
    ])
      .then(([customerResponse, resourceResponse, packageResponse]) => {
        const readyPackages = packageResponse.items.filter((item) => item.ready);
        setCustomers(customerResponse.items);
        setResources(resourceResponse.items);
        setPackages(readyPackages);
        if (presetPackageID) {
          const preset = readyPackages.find((item) => item.id === presetPackageID);
          if (preset) {
            setPackageMessage(`“${preset.name}” está preseleccionado. Revisa el período y agrégalo a la cotización.`);
          } else {
            setSelectedPackageID("");
            setPackageMessage("El paquete solicitado no está activo o requiere atención antes de poder cotizarse.");
          }
        }
      })
      .catch((reason) => setOptionsError(reason instanceof Error ? reason.message : "No fue posible cargar clientes, inventario y paquetes."))
      .finally(() => setLoadingOptions(false));
  }, [presetPackageID]);

  const calculations = useMemo(() => {
    const lines = items.map((item) => {
      const quantity = Math.max(0, Number(item.quantity) || 0);
      const gross = quantity * money(item.unit_price);
      const discount = money(item.discount_amount);
      return Math.max(0, Math.round((gross - discount) * 100) / 100);
    });
    const subtotal = Math.round(lines.reduce((sum, value) => sum + value, 0) * 100) / 100;
    const quoteDiscount = money(form.discount_amount);
    const extraCharges = money(form.extra_charges);
    const total = Math.max(0, Math.round((subtotal - quoteDiscount + extraCharges) * 100) / 100);
    return { lines, subtotal, quoteDiscount, extraCharges, total };
  }, [form.discount_amount, form.extra_charges, items]);

  function updateItem(index: number, patch: Partial<DraftItem>) {
    setItems((current) => current.map((item, itemIndex) => (itemIndex === index ? { ...item, ...patch } : item)));
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

  function lineGrossCents(item: DraftItem): number {
    const quantity = Math.max(0, Number(item.quantity) || 0);
    return Math.round(quantity * money(item.unit_price) * 100);
  }

  function lineTotalCents(item: DraftItem): number {
    const gross = lineGrossCents(item);
    const discount = Math.round(money(item.discount_amount) * 100);
    return Math.max(0, gross - discount);
  }

  function mergedQuoteLine(
    resourceID: string,
    description: string,
    quantity: number,
    grossCents: number,
    totalCents: number,
    key: string,
  ): DraftItem {
    const safeQuantity = Math.max(1, quantity);
    const unitPriceCents = grossCents > 0 ? Math.ceil(grossCents / safeQuantity) : 0;
    const normalizedGrossCents = unitPriceCents * safeQuantity;
    const discountCents = Math.max(0, normalizedGrossCents - Math.max(0, totalCents));
    return {
      key,
      resource_id: resourceID,
      description,
      quantity: String(safeQuantity),
      unit_price: String(unitPriceCents / 100),
      discount_amount: String(discountCents / 100),
    };
  }

  function mergePackageTemplate(template: PackageQuoteTemplate) {
    setItems((current) => {
      const merged = current.filter((item) => item.resource_id);
      for (const incoming of template.items) {
        const index = merged.findIndex((item) => item.resource_id === incoming.resource_id);
        if (index < 0) {
          merged.push({
            key: draftKey(),
            resource_id: incoming.resource_id,
            description: incoming.description,
            quantity: String(incoming.quantity),
            unit_price: String(incoming.unit_price),
            discount_amount: String(incoming.discount_amount),
          });
          continue;
        }
        const existing = merged[index];
        const quantity = Math.max(0, Number(existing.quantity) || 0) + incoming.quantity;
        const grossCents = lineGrossCents(existing) + Math.round(incoming.quantity * incoming.unit_price * 100);
        const totalCents = lineTotalCents(existing) + Math.round(incoming.line_total * 100);
        merged[index] = mergedQuoteLine(
          existing.resource_id,
          existing.description || incoming.description,
          quantity,
          grossCents,
          totalCents,
          existing.key,
        );
      }
      return merged.length ? merged : [newItem()];
    });
  }

  async function applyPackage() {
    if (!selectedPackageID) {
      setPackageMessage("Selecciona un paquete.");
      return;
    }
    const quantity = Math.max(1, Math.min(100, Number(packageSets) || 1));
    setApplyingPackage(true);
    setPackageMessage("");
    setPackageAvailability(null);
    try {
      const template = await api<PackageQuoteTemplate>(`/api/v1/packages/${selectedPackageID}/quote-template?quantity=${quantity}`);
      const availability = await api<PackageAvailabilityResult>(`/api/v1/packages/${selectedPackageID}/availability`, {
        method: "POST",
        body: JSON.stringify({
          start_at: asISO(form.start_at),
          end_at: asISO(form.end_at),
          quantity,
        }),
      });
      setPackageAvailability(availability);
      if (!availability.available && !window.confirm("Este paquete no tiene capacidad completa para el período seleccionado. ¿Deseas agregarlo a la cotización de todas formas?")) {
        setPackageMessage("No se agregó el paquete porque existen conflictos de disponibilidad.");
        return;
      }
      mergePackageTemplate(template);
      if (template.extra_charges > 0) {
        setForm((current) => ({
          ...current,
          extra_charges: String(Math.round((money(current.extra_charges) + template.extra_charges) * 100) / 100),
        }));
      }
      setAppliedPackages((current) => [...current, {
        id: template.package_id,
        name: template.package_name,
        quantity: template.package_quantity,
        effectivePrice: template.effective_price,
      }]);
      setPackageMessage(`${template.package_quantity} × ${template.package_name} agregado a la cotización.`);
    } catch (reason) {
      setPackageMessage(reason instanceof ApiError ? reason.message : "No fue posible agregar el paquete.");
    } finally {
      setApplyingPackage(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setMessage("");
    setFields({});
    try {
      const payload = {
        customer_id: form.customer_id,
        start_at: asISO(form.start_at),
        end_at: asISO(form.end_at),
        event_type: form.event_type || null,
        event_location: form.event_location || null,
        discount_amount: money(form.discount_amount),
        extra_charges: money(form.extra_charges),
        notes: form.notes,
        expires_at: form.expires_at ? asISO(form.expires_at) : null,
        items: items.map((item) => ({
          resource_id: item.resource_id,
          description: item.description,
          quantity: Number(item.quantity),
          unit_price: money(item.unit_price),
          discount_amount: money(item.discount_amount),
        })),
      };
      const quote = await api<QuoteDetail>(initial ? `/api/v1/quotes/${initial.id}` : "/api/v1/quotes", {
        method: initial ? "PATCH" : "POST",
        body: JSON.stringify(payload),
      });
      router.push(`/quotes/${quote.id}`);
      router.refresh();
    } catch (error) {
      if (error instanceof ApiError) {
        setFields(error.fields || {});
        setMessage(error.message);
      } else {
        setMessage("No fue posible guardar la cotización.");
      }
      window.scrollTo({ top: 0, behavior: "smooth" });
    } finally {
      setSaving(false);
    }
  }

  if (loadingOptions) {
    return <div className="skeleton detail-skeleton" />;
  }

  if (optionsError) {
    return <div className="panel inline-error">{optionsError}</div>;
  }

  return (
    <form onSubmit={submit} className="quote-editor-layout">
      <div className="quote-editor-main page-stack">
        {message && <div className="form-alert">{message}</div>}

        <section className="panel quote-form-section">
          <div className="panel-header">
            <div>
              <p className="eyebrow">CLIENTE Y EVENTO</p>
              <h2>Información comercial</h2>
              <p>La cotización conserva una fotografía histórica de precios y condiciones.</p>
            </div>
          </div>
          <div className="quote-form-body">
            <div className="form-grid two-columns">
              <label className="field">
                <span>Cliente *</span>
                <select
                  value={form.customer_id}
                  onChange={(event) => setForm({ ...form, customer_id: event.target.value })}
                >
                  <option value="">Seleccionar cliente</option>
                  {customers.map((customer) => (
                    <option value={customer.id} key={customer.id}>
                      {customer.display_name}{customer.company_name ? ` · ${customer.company_name}` : ""}
                    </option>
                  ))}
                </select>
                {fields.customer_id && <small className="field-error">{fields.customer_id}</small>}
                {customers.length === 0 && (
                  <small className="field-hint">
                    No hay clientes. <Link href="/customers">Crea el primero</Link>.
                  </small>
                )}
              </label>
              <label className="field">
                <span>Tipo de evento</span>
                <input
                  value={form.event_type}
                  onChange={(event) => setForm({ ...form, event_type: event.target.value })}
                  placeholder="Fiesta, boda, concierto, evento corporativo…"
                />
                {fields.event_type && <small className="field-error">{fields.event_type}</small>}
              </label>
            </div>

            <label className="field">
              <span>Ubicación del evento</span>
              <input
                value={form.event_location}
                onChange={(event) => setForm({ ...form, event_location: event.target.value })}
                placeholder="Dirección, venue o zona"
              />
              {fields.event_location && <small className="field-error">{fields.event_location}</small>}
            </label>

            <div className="form-grid three-columns">
              <label className="field">
                <span>Bloqueo desde *</span>
                <input
                  type="datetime-local"
                  value={form.start_at}
                  onChange={(event) => setForm({ ...form, start_at: event.target.value })}
                />
                {fields.start_at && <small className="field-error">{fields.start_at}</small>}
              </label>
              <label className="field">
                <span>Bloqueo hasta *</span>
                <input
                  type="datetime-local"
                  value={form.end_at}
                  onChange={(event) => setForm({ ...form, end_at: event.target.value })}
                />
                {fields.end_at && <small className="field-error">{fields.end_at}</small>}
              </label>
              <label className="field">
                <span>Válida hasta</span>
                <input
                  type="datetime-local"
                  value={form.expires_at}
                  onChange={(event) => setForm({ ...form, expires_at: event.target.value })}
                />
                {fields.expires_at && <small className="field-error">{fields.expires_at}</small>}
              </label>
            </div>
          </div>
        </section>

        <section className="panel quote-form-section">
          <div className="panel-header quote-lines-header">
            <div>
              <p className="eyebrow">RECURSOS COTIZADOS</p>
              <h2>Equipo y servicios</h2>
              <p>Los precios guardados aquí no cambiarán aunque luego modifiques el catálogo.</p>
            </div>
            <button type="button" className="button button-secondary" onClick={() => setItems((current) => [...current, newItem()])}>
              <span className="button-plus quote-plus">+</span> Agregar línea
            </button>
          </div>
          <div className="package-quote-picker">
            <div className="package-quote-picker-copy">
              <span className="package-quote-picker-icon">◇</span>
              <div><strong>Agregar desde un paquete</strong><small>Expande una plantilla reusable y conserva el total comercial exacto.</small></div>
            </div>
            <div className="package-quote-picker-controls">
              <select value={selectedPackageID} onChange={(event) => { setSelectedPackageID(event.target.value); setPackageMessage(""); setPackageAvailability(null); }}>
                <option value="">Seleccionar paquete</option>
                {packages.map((item) => <option key={item.id} value={item.id}>{item.name} · {formatCurrency(item.effective_price)}</option>)}
              </select>
              <label><span>Paquetes</span><input type="number" min="1" max="100" value={packageSets} onChange={(event) => setPackageSets(event.target.value)} /></label>
              <button type="button" className="button button-secondary" disabled={applyingPackage || !selectedPackageID} onClick={() => void applyPackage()}>{applyingPackage ? "Validando…" : "Agregar paquete"}</button>
            </div>
            {packages.length === 0 && (
              <small className="package-picker-empty">
                No hay paquetes activos y listos. {can("package.manage") ? <Link href="/packages/new">Crea el primero</Link> : <Link href="/packages">Revisa Paquetes</Link>}.
              </small>
            )}
            {packageMessage && <div className={`package-picker-message ${packageAvailability && !packageAvailability.available ? "warning" : ""}`}>{packageMessage}</div>}
            {packageAvailability && (
              <div className={`package-picker-availability ${packageAvailability.available ? "available" : "conflict"}`}>
                <strong>{packageAvailability.available ? "Disponible para el período" : "Disponibilidad incompleta"}</strong>
                <span>{packageAvailability.items.filter((item) => item.can_fulfill).length}/{packageAvailability.items.length} recursos pueden cubrirse</span>
              </div>
            )}
            {appliedPackages.length > 0 && <div className="applied-package-list">{appliedPackages.map((item, index) => <span key={`${item.id}-${index}`}><strong>{item.quantity}×</strong> {item.name} · {formatCurrency(item.effectivePrice)}</span>)}</div>}
          </div>
          <div className="quote-lines">
            {items.map((item, index) => {
              const resource = resources.find((resourceItem) => resourceItem.id === item.resource_id);
              return (
                <article className="quote-line-card" key={item.key}>
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
                        {fields[`items[${index}].resource_id`] && (
                          <small className="field-error">{fields[`items[${index}].resource_id`]}</small>
                        )}
                        {resource && (
                          <small className="field-hint">
                            {resource.available_asset_count} de {resource.asset_count} unidades físicamente disponibles ahora
                          </small>
                        )}
                      </label>
                      <label className="field">
                        <span>Cantidad *</span>
                        <input
                          type="number"
                          min="1"
                          step="1"
                          value={item.quantity}
                          onChange={(event) => updateItem(index, { quantity: event.target.value })}
                        />
                        {fields[`items[${index}].quantity`] && (
                          <small className="field-error">{fields[`items[${index}].quantity`]}</small>
                        )}
                      </label>
                      <label className="field">
                        <span>Precio unitario</span>
                        <div className="input-prefix">
                          <span>$</span>
                          <input
                            type="number"
                            min="0"
                            step="0.01"
                            value={item.unit_price}
                            onChange={(event) => updateItem(index, { unit_price: event.target.value })}
                          />
                        </div>
                        {fields[`items[${index}].unit_price`] && (
                          <small className="field-error">{fields[`items[${index}].unit_price`]}</small>
                        )}
                      </label>
                      <label className="field">
                        <span>Descuento línea</span>
                        <div className="input-prefix">
                          <span>$</span>
                          <input
                            type="number"
                            min="0"
                            step="0.01"
                            value={item.discount_amount}
                            onChange={(event) => updateItem(index, { discount_amount: event.target.value })}
                          />
                        </div>
                        {fields[`items[${index}].discount_amount`] && (
                          <small className="field-error">{fields[`items[${index}].discount_amount`]}</small>
                        )}
                      </label>
                    </div>
                    <div className="quote-line-bottom">
                      <label className="field quote-description-field">
                        <span>Descripción para el cliente</span>
                        <input
                          value={item.description}
                          onChange={(event) => updateItem(index, { description: event.target.value })}
                          placeholder="Se usará el nombre del recurso si se deja vacío"
                        />
                      </label>
                      <div className="quote-line-total">
                        <span>Total línea</span>
                        <strong>{formatCurrency(calculations.lines[index] || 0)}</strong>
                      </div>
                      <button
                        type="button"
                        className="icon-action quote-remove-line"
                        onClick={() => setItems((current) => current.filter((_, itemIndex) => itemIndex !== index))}
                        disabled={items.length === 1}
                        title="Eliminar línea"
                      >
                        ×
                      </button>
                    </div>
                  </div>
                </article>
              );
            })}
            {fields.items && <div className="quote-items-error">{fields.items}</div>}
          </div>
        </section>

        <section className="panel quote-form-section">
          <div className="panel-header">
            <div>
              <p className="eyebrow">CONDICIONES</p>
              <h2>Notas de la cotización</h2>
            </div>
          </div>
          <div className="quote-form-body">
            <label className="field">
              <span>Notas internas o comerciales</span>
              <textarea
                rows={5}
                value={form.notes}
                onChange={(event) => setForm({ ...form, notes: event.target.value })}
                placeholder="Incluye condiciones de entrega, cableado, instalación o restricciones."
              />
              {fields.notes && <small className="field-error">{fields.notes}</small>}
            </label>
          </div>
        </section>
      </div>

      <aside className="quote-summary-panel panel">
        <div className="quote-summary-header">
          <p className="eyebrow">RESUMEN</p>
          <h2>{initial ? "Actualizar borrador" : "Nueva cotización"}</h2>
          <p>Las cotizaciones no bloquean inventario hasta convertirse en reserva.</p>
        </div>
        <div className="quote-summary-money-fields">
          <label className="field">
            <span>Descuento general</span>
            <div className="input-prefix">
              <span>$</span>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.discount_amount}
                onChange={(event) => setForm({ ...form, discount_amount: event.target.value })}
              />
            </div>
            {fields.discount_amount && <small className="field-error">{fields.discount_amount}</small>}
          </label>
          <label className="field">
            <span>Cargos adicionales</span>
            <div className="input-prefix">
              <span>$</span>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.extra_charges}
                onChange={(event) => setForm({ ...form, extra_charges: event.target.value })}
              />
            </div>
            {fields.extra_charges && <small className="field-error">{fields.extra_charges}</small>}
          </label>
        </div>
        <div className="quote-total-breakdown">
          <div><span>Subtotal</span><strong>{formatCurrency(calculations.subtotal)}</strong></div>
          <div><span>Descuento</span><strong>− {formatCurrency(calculations.quoteDiscount)}</strong></div>
          <div><span>Cargos adicionales</span><strong>{formatCurrency(calculations.extraCharges)}</strong></div>
          <div className="quote-grand-total"><span>Total</span><strong>{formatCurrency(calculations.total)}</strong></div>
        </div>
        <div className="quote-summary-actions">
          <Link href={initial ? `/quotes/${initial.id}` : "/quotes"} className="button button-secondary">
            Cancelar
          </Link>
          <button className="button button-primary" type="submit" disabled={saving || customers.length === 0 || resources.length === 0}>
            {saving ? "Guardando…" : initial ? "Guardar cambios" : "Crear borrador"}
          </button>
        </div>
      </aside>
    </form>
  );
}
