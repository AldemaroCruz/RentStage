"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReservationStatusBadge } from "@/components/ReservationStatusBadge";
import { useAuth } from "@/components/AuthProvider";
import { api } from "@/lib/api";
import {
  formatCurrency,
  formatLongDate,
  formatReservationNumber,
  formatTime,
  operationAlertLabel,
  toLocalDateInput,
} from "@/lib/format";
import type {
  CalendarReservation,
  CalendarResult,
  Customer,
  OperationAlert,
  OperationAlertsResult,
  OperationsAgenda,
  ReservationStatus,
  Resource,
} from "@/lib/types";

type CalendarView = "month" | "week" | "agenda";

const activeStatuses: ReservationStatus[] = ["PENDING", "CONFIRMED", "PREPARING", "READY", "CHECKED_OUT", "RETURNED"];
const allStatuses: Array<{ value: ReservationStatus; label: string }> = [
  { value: "PENDING", label: "Pendiente" },
  { value: "CONFIRMED", label: "Confirmada" },
  { value: "PREPARING", label: "Preparando" },
  { value: "READY", label: "Lista" },
  { value: "CHECKED_OUT", label: "Entregada" },
  { value: "RETURNED", label: "Devuelta" },
  { value: "COMPLETED", label: "Completada" },
  { value: "CANCELLED", label: "Cancelada" },
];
const weekdays = ["Lun", "Mar", "Mié", "Jue", "Vie", "Sáb", "Dom"];

function startOfDay(date: Date): Date {
  const value = new Date(date);
  value.setHours(0, 0, 0, 0);
  return value;
}

function addDays(date: Date, count: number): Date {
  const value = new Date(date);
  value.setDate(value.getDate() + count);
  return value;
}

function startOfWeek(date: Date): Date {
  const value = startOfDay(date);
  const day = value.getDay();
  const offset = day === 0 ? -6 : 1 - day;
  return addDays(value, offset);
}

function startOfMonth(date: Date): Date {
  const value = startOfDay(date);
  value.setDate(1);
  return value;
}

function rangeFor(view: CalendarView, anchor: Date): { from: Date; to: Date } {
  if (view === "month") {
    const monthStart = startOfMonth(anchor);
    const from = startOfWeek(monthStart);
    return { from, to: addDays(from, 42) };
  }
  if (view === "week") {
    const from = startOfWeek(anchor);
    return { from, to: addDays(from, 7) };
  }
  const from = startOfDay(anchor);
  return { from, to: addDays(from, 1) };
}

function sameDay(left: Date, right: Date): boolean {
  return left.getFullYear() === right.getFullYear() && left.getMonth() === right.getMonth() && left.getDate() === right.getDate();
}

function eventOnDay(item: CalendarReservation, day: Date): boolean {
  const from = startOfDay(day);
  const to = addDays(from, 1);
  return new Date(item.block_start_at) < to && new Date(item.block_end_at) > from;
}

function itemMatches(
  item: CalendarReservation,
  search: string,
  customerID: string,
  resourceID: string,
  statuses: Set<ReservationStatus>,
): boolean {
  if (!statuses.has(item.status)) return false;
  if (customerID && item.customer_id !== customerID) return false;
  if (resourceID && !item.resource_summary.toLowerCase().includes(resourceID.toLowerCase())) return false;
  if (!search.trim()) return true;
  const term = search.trim().toLowerCase();
  return [item.customer_name, item.event_type, item.event_location, item.resource_summary, formatReservationNumber(item.reservation_number)]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(term));
}

function CalendarCard({ item, compact = false }: { item: CalendarReservation; compact?: boolean }) {
  const missing = Math.max(0, item.required_asset_count - item.assigned_asset_count);
  return (
    <Link href={`/reservations/${item.id}`} className={`calendar-event-card calendar-status-${item.status.toLowerCase()} ${compact ? "is-compact" : ""}`}>
      <span className="calendar-event-time">{formatTime(item.block_start_at)}</span>
      <strong>{item.event_type || item.customer_name}</strong>
      <small>{formatReservationNumber(item.reservation_number)} · {item.customer_name}</small>
      {!compact && <p>{item.resource_summary || `${item.item_count} recursos`}</p>}
      {item.required_asset_count > 0 && (
        <span className={`calendar-assignment-pill ${missing > 0 ? "has-missing" : "is-complete"}`}>
          {item.assigned_asset_count}/{item.required_asset_count} unidades
        </span>
      )}
    </Link>
  );
}

function AgendaGroup({ title, empty, items }: { title: string; empty: string; items: CalendarReservation[] }) {
  return (
    <section className="panel agenda-group">
      <header><div><p className="eyebrow">OPERACIÓN</p><h3>{title}</h3></div><span>{items.length}</span></header>
      {items.length === 0 ? <p className="agenda-empty">{empty}</p> : (
        <div className="agenda-list">
          {items.map((item) => (
            <Link href={`/reservations/${item.id}`} className="agenda-row" key={`${title}-${item.id}`}>
              <div className="agenda-time"><strong>{formatTime(title === "Retornos" ? item.block_end_at : item.block_start_at)}</strong><small>{title === "Eventos" ? formatTime(item.event_start_at) : ""}</small></div>
              <div className="agenda-copy"><strong>{item.event_type || "Reserva de alquiler"}</strong><span>{formatReservationNumber(item.reservation_number)} · {item.customer_name}</span><small>{item.resource_summary || `${item.item_count} recursos`}</small></div>
              <ReservationStatusBadge status={item.status} />
              <span className="row-arrow">→</span>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

export default function CalendarPage() {
  const { can } = useAuth();
  const canManageReservations = can("reservation.manage");
  const [view, setView] = useState<CalendarView>("month");
  const [anchor, setAnchor] = useState(() => startOfDay(new Date()));
  const [selectedDate, setSelectedDate] = useState(() => startOfDay(new Date()));
  const [statuses, setStatuses] = useState<Set<ReservationStatus>>(() => new Set(activeStatuses));
  const [customerID, setCustomerID] = useState("");
  const [resourceID, setResourceID] = useState("");
  const [search, setSearch] = useState("");
  const [calendar, setCalendar] = useState<CalendarResult | null>(null);
  const [agenda, setAgenda] = useState<OperationsAgenda | null>(null);
  const [alerts, setAlerts] = useState<OperationAlertsResult | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const requestedView = params.get("view");
    if (requestedView === "month" || requestedView === "week" || requestedView === "agenda") {
      setView(requestedView);
    }
    const requestedDate = params.get("date");
    if (requestedDate) {
      const parsed = new Date(`${requestedDate}T00:00:00`);
      if (!Number.isNaN(parsed.getTime())) {
        setAnchor(startOfDay(parsed));
        setSelectedDate(startOfDay(parsed));
      }
    }
  }, []);

  useEffect(() => {
    Promise.all([
      api<{ items: Customer[] }>("/api/v1/customers"),
      api<{ items: Resource[] }>("/api/v1/resources?active=true"),
    ]).then(([customerResult, resourceResult]) => {
      setCustomers(customerResult.items);
      setResources(resourceResult.items);
    }).catch(() => undefined);
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    const range = rangeFor(view, anchor);
    const params = new URLSearchParams({ from: range.from.toISOString(), to: range.to.toISOString() });
    if (statuses.size > 0) params.set("status", Array.from(statuses).join(","));
    if (customerID) params.set("customer_id", customerID);
    if (resourceID) params.set("resource_id", resourceID);
    try {
      const [calendarResult, agendaResult, alertsResult] = await Promise.all([
        api<CalendarResult>(`/api/v1/calendar?${params.toString()}`),
        api<OperationsAgenda>(`/api/v1/operations/agenda?date=${toLocalDateInput(selectedDate)}`),
        api<OperationAlertsResult>("/api/v1/operations/alerts"),
      ]);
      setCalendar(calendarResult);
      setAgenda(agendaResult);
      setAlerts(alertsResult);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible cargar el calendario operacional.");
    } finally {
      setLoading(false);
    }
  }, [anchor, customerID, resourceID, selectedDate, statuses, view]);

  useEffect(() => { void load(); }, [load]);

  const filteredItems = useMemo(() => {
    if (!calendar) return [];
    const resourceName = resourceID ? resources.find((item) => item.id === resourceID)?.name || resourceID : "";
    return calendar.items.filter((item) => itemMatches(item, search, customerID, resourceName, statuses));
  }, [calendar, customerID, resourceID, resources, search, statuses]);

  const filteredAgenda = useMemo(() => {
    if (!agenda) return null;
    const resourceName = resourceID ? resources.find((item) => item.id === resourceID)?.name || resourceID : "";
    const filter = (items: CalendarReservation[]) => items.filter((item) => itemMatches(item, search, customerID, resourceName, statuses));
    return {
      ...agenda,
      departures: filter(agenda.departures),
      events: filter(agenda.events),
      returns: filter(agenda.returns),
      pending_close: filter(agenda.pending_close),
    };
  }, [agenda, customerID, resourceID, resources, search, statuses]);

  const visibleDays = useMemo(() => {
    const range = rangeFor(view === "agenda" ? "week" : view, anchor);
    const count = view === "month" ? 42 : 7;
    return Array.from({ length: count }, (_, index) => addDays(range.from, index));
  }, [anchor, view]);

  function move(direction: -1 | 1) {
    const next = new Date(anchor);
    if (view === "month") next.setMonth(next.getMonth() + direction);
    else next.setDate(next.getDate() + direction * (view === "week" ? 7 : 1));
    setAnchor(startOfDay(next));
    if (view === "agenda") setSelectedDate(startOfDay(next));
  }

  function chooseDay(day: Date) {
    setSelectedDate(startOfDay(day));
    if (view === "agenda") setAnchor(startOfDay(day));
  }

  function selectView(next: CalendarView) {
    setView(next);
    if (next === "agenda") setAnchor(selectedDate);
  }

  function toggleStatus(status: ReservationStatus) {
    setStatuses((current) => {
      const next = new Set(current);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  }

  const title = view === "month"
    ? new Intl.DateTimeFormat("es-SV", { month: "long", year: "numeric" }).format(anchor)
    : view === "week"
      ? `${new Intl.DateTimeFormat("es-SV", { day: "2-digit", month: "short" }).format(visibleDays[0])} – ${new Intl.DateTimeFormat("es-SV", { day: "2-digit", month: "short", year: "numeric" }).format(visibleDays[6])}`
      : formatLongDate(selectedDate);

  return (
    <div className="page-stack calendar-page">
      <section className="page-heading calendar-page-heading">
        <div>
          <p className="eyebrow">OPERATIONS CENTER</p>
          <h2>Calendario y agenda</h2>
          <p>Visualiza compromisos, preparación, salidas, eventos y retornos desde el período que realmente bloquea inventario.</p>
        </div>
        {canManageReservations && <Link className="button button-primary" href="/reservations/new">+ Nueva reserva</Link>}
      </section>

      <section className="calendar-toolbar panel">
        <div className="calendar-navigation">
          <button className="icon-action" onClick={() => move(-1)} aria-label="Período anterior">←</button>
          <button className="button button-secondary button-small" onClick={() => { const today = startOfDay(new Date()); setAnchor(today); setSelectedDate(today); }}>Hoy</button>
          <button className="icon-action" onClick={() => move(1)} aria-label="Período siguiente">→</button>
          <h3>{title}</h3>
        </div>
        <div className="calendar-view-switch" role="group" aria-label="Vista del calendario">
          {(["month", "week", "agenda"] as CalendarView[]).map((item) => (
            <button key={item} className={view === item ? "active" : ""} onClick={() => selectView(item)}>
              {item === "month" ? "Mes" : item === "week" ? "Semana" : "Agenda"}
            </button>
          ))}
        </div>
      </section>

      <section className="calendar-filter-panel panel">
        <label className="search-box calendar-search"><span aria-hidden="true">⌕</span><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Buscar cliente, evento, reserva o recurso" /></label>
        <select value={customerID} onChange={(event) => setCustomerID(event.target.value)}><option value="">Todos los clientes</option>{customers.map((customer) => <option key={customer.id} value={customer.id}>{customer.display_name}</option>)}</select>
        <select value={resourceID} onChange={(event) => setResourceID(event.target.value)}><option value="">Todo el inventario</option>{resources.map((resource) => <option key={resource.id} value={resource.id}>{resource.name}</option>)}</select>
        <div className="calendar-status-filters">
          {allStatuses.map((item) => (
            <button key={item.value} className={statuses.has(item.value) ? "is-selected" : ""} onClick={() => toggleStatus(item.value)}>
              <span />{item.label}
            </button>
          ))}
        </div>
      </section>

      {error && <div className="form-alert">{error}</div>}

      {view !== "agenda" && (
        <section className={`calendar-board panel calendar-${view}-view`}>
          <div className="calendar-weekday-row">{weekdays.map((day) => <span key={day}>{day}</span>)}</div>
          <div className="calendar-grid">
            {visibleDays.map((day) => {
              const dayItems = filteredItems.filter((item) => eventOnDay(item, day));
              const outside = view === "month" && day.getMonth() !== anchor.getMonth();
              return (
                <article key={day.toISOString()} className={`calendar-day ${outside ? "is-outside" : ""} ${sameDay(day, new Date()) ? "is-today" : ""} ${sameDay(day, selectedDate) ? "is-selected" : ""}`}>
                  <button className="calendar-day-number" onClick={() => chooseDay(day)}><span>{day.getDate()}</span><small>{dayItems.length || ""}</small></button>
                  <div className="calendar-day-events">
                    {dayItems.slice(0, view === "month" ? 3 : 8).map((item) => <CalendarCard key={`${day.toISOString()}-${item.id}`} item={item} compact={view === "month"} />)}
                    {dayItems.length > (view === "month" ? 3 : 8) && <button className="calendar-more" onClick={() => { chooseDay(day); selectView("agenda"); }}>+ {dayItems.length - (view === "month" ? 3 : 8)} más</button>}
                  </div>
                </article>
              );
            })}
          </div>
          {loading && <div className="calendar-loading">Actualizando calendario…</div>}
        </section>
      )}

      {view === "agenda" && (
        <div className="agenda-layout">
          <div className="page-stack">
            <section className="panel agenda-date-picker">
              <label className="field"><span>Día operativo</span><input type="date" value={toLocalDateInput(selectedDate)} onChange={(event) => { const next = new Date(`${event.target.value}T00:00:00`); setSelectedDate(next); setAnchor(next); }} /></label>
              <div><p className="eyebrow">AGENDA DEL DÍA</p><h2>{formatLongDate(selectedDate)}</h2><p>{filteredItems.length} reservas intersectan el período consultado.</p></div>
            </section>
            {filteredAgenda ? (
              <>
                <AgendaGroup title="Salidas" empty="No hay pedidos programados para salir este día." items={filteredAgenda.departures} />
                <AgendaGroup title="Eventos" empty="No hay eventos iniciando este día." items={filteredAgenda.events} />
                <AgendaGroup title="Retornos" empty="No hay retornos esperados este día." items={filteredAgenda.returns} />
                <AgendaGroup title="Pendientes de cierre" empty="No hay reservas devueltas pendientes de completar." items={filteredAgenda.pending_close} />
              </>
            ) : <div className="skeleton skeleton-panel" />}
          </div>
          <aside className="page-stack">
            <section className="panel agenda-summary-card">
              <p className="eyebrow">RESUMEN</p>
              <div><span>Salidas</span><strong>{filteredAgenda?.departures.length || 0}</strong></div>
              <div><span>Eventos</span><strong>{filteredAgenda?.events.length || 0}</strong></div>
              <div><span>Retornos</span><strong>{filteredAgenda?.returns.length || 0}</strong></div>
              <div><span>Valor visible</span><strong>{formatCurrency(filteredItems.reduce((sum, item) => sum + item.total, 0))}</strong></div>
            </section>
            {(filteredAgenda?.overdue_returns.length || 0) > 0 && (
              <section className="panel overdue-summary-card"><p className="eyebrow">ATENCIÓN</p><h3>{filteredAgenda?.overdue_returns.length} retornos atrasados</h3><p>Hay equipo fuera después del horario esperado.</p></section>
            )}
          </aside>
        </div>
      )}

      <section className="panel operations-alert-panel" id="operations-alerts">
        <header className="panel-header">
          <div><p className="eyebrow">ALERTAS OPERATIVAS</p><h2>Atención requerida</h2><p>Derivadas de horarios, estados y asignaciones físicas.</p></div>
          <div className="alert-counts"><span className="critical">{alerts?.counts.critical || 0}</span><span className="warning">{alerts?.counts.warning || 0}</span><span className="info">{alerts?.counts.info || 0}</span></div>
        </header>
        {!alerts || alerts.items.length === 0 ? (
          <div className="operations-all-clear"><span>✓</span><div><strong>Operación al día</strong><p>No hay retornos atrasados ni preparaciones urgentes.</p></div></div>
        ) : (
          <div className="operations-alert-list">
            {alerts.items.map((alert: OperationAlert) => (
              <Link href={`/reservations/${alert.reservation_id}`} key={alert.id} className={`operation-alert-row severity-${alert.severity.toLowerCase()}`}>
                <span className="operation-alert-icon">{alert.severity === "CRITICAL" ? "!" : alert.severity === "WARNING" ? "△" : "i"}</span>
                <div><strong>{operationAlertLabel(alert.type)}</strong><p>{alert.message}</p><small>{formatReservationNumber(alert.reservation_number)} · {alert.customer_name}{alert.due_at ? ` · ${formatTime(alert.due_at)}` : ""}</small></div>
                {alert.missing_asset_count > 0 && <span className="operation-alert-gap">Faltan {alert.missing_asset_count}</span>}
                {alert.minutes_overdue > 0 && <span className="operation-alert-gap">{Math.floor(alert.minutes_overdue / 60)}h {alert.minutes_overdue % 60}m</span>}
                <span className="row-arrow">→</span>
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
