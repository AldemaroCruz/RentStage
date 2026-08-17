export function formatCurrency(value: number, currency = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
    maximumFractionDigits: 2,
  }).format(value || 0);
}

export function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("es-SV", {
    year: "numeric",
    month: "short",
    day: "2-digit",
  }).format(date);
}

export function formatDateTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("es-SV", {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function pricingUnitLabel(unit: string): string {
  const labels: Record<string, string> = {
    HOUR: "hora",
    DAY: "día",
    EVENT: "evento",
    FIXED: "precio fijo",
  };
  return labels[unit] || unit.toLowerCase();
}

export function formatQuoteNumber(value: number): string {
  return `QT-${String(value).padStart(6, "0")}`;
}

export function quoteStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    DRAFT: "Borrador",
    SENT: "Enviada",
    ACCEPTED: "Aceptada",
    REJECTED: "Rechazada",
    EXPIRED: "Expirada",
    CANCELLED: "Cancelada",
  };
  return labels[status] || status;
}

export function customerSourceLabel(source: string): string {
  const labels: Record<string, string> = {
    MANUAL: "Manual",
    WEB: "Web",
    WHATSAPP: "WhatsApp",
    IMPORT: "Importado",
  };
  return labels[source] || source;
}

export function toLocalDateTimeInput(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

export function formatReservationNumber(value: number): string {
  return `RS-${String(value).padStart(6, "0")}`;
}

export function reservationStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    PENDING: "Pendiente",
    CONFIRMED: "Confirmada",
    PREPARING: "Preparando",
    READY: "Lista",
    CHECKED_OUT: "Entregada",
    RETURNED: "Devuelta",
    COMPLETED: "Completada",
    CANCELLED: "Cancelada",
  };
  return labels[status] || status;
}

export function assignmentStateLabel(state: string): string {
  const labels: Record<string, string> = {
    ASSIGNED: "Asignada",
    CHECKED_OUT: "Entregada",
    RETURNED: "Devuelta",
    RELEASED: "Liberada",
  };
  return labels[state] || state;
}

export function returnConditionLabel(condition?: string): string {
  const labels: Record<string, string> = {
    GOOD: "Buen estado",
    MAINTENANCE_REQUIRED: "Requiere mantenimiento",
    DAMAGED: "Dañada",
    LOST: "Perdida",
  };
  if (!condition) return "—";
  return labels[condition] || condition;
}

export function warehouseActivityLabel(eventType: string): string {
  const labels: Record<string, string> = {
    ASSET_ASSIGNED: "Unidad asignada",
    ASSET_UNASSIGNED: "Unidad desasignada",
    ASSET_CHECKED_OUT: "Unidad entregada",
    ASSET_RETURNED: "Unidad devuelta",
    ASSIGNMENTS_RELEASED: "Asignación liberada",
  };
  return labels[eventType] || eventType;
}

export function reservationSourceLabel(source: string): string {
  const labels: Record<string, string> = {
    QUOTE: "Cotización",
    MANUAL: "Manual",
    WEB: "Web",
    WHATSAPP: "WhatsApp",
    AI_AGENT: "Agente AI",
  };
  return labels[source] || source;
}

export function formatTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("es-SV", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function formatLongDate(value?: string | Date): string {
  if (!value) return "—";
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("es-SV", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(date);
}

export function toLocalDateInput(value?: string | Date): string {
  if (!value) return "";
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 10);
}

export function operationAlertLabel(type: string): string {
  const labels: Record<string, string> = {
    OVERDUE_RETURN: "Retorno atrasado",
    PREPARATION_NOT_STARTED: "Preparación pendiente",
    PREPARATION_INCOMPLETE: "Inventario incompleto",
    CHECKOUT_PENDING: "Entrega pendiente",
    COMPLETION_PENDING: "Cierre pendiente",
  };
  return labels[type] || type;
}

export function quoteRequestStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    NEW: "Nueva",
    IN_REVIEW: "En revisión",
    CONVERTED: "Convertida",
    CLOSED: "Cerrada",
    SPAM: "Spam",
  };
  return labels[status] || status;
}

export function quoteRequestStatusTone(status: string): string {
  const tones: Record<string, string> = {
    NEW: "new",
    IN_REVIEW: "review",
    CONVERTED: "converted",
    CLOSED: "closed",
    SPAM: "spam",
  };
  return tones[status] || "closed";
}

export function invoiceStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    DRAFT: "Borrador",
    ISSUED: "Emitida",
    PARTIALLY_PAID: "Pago parcial",
    PAID: "Pagada",
    OVERDUE: "Vencida",
    VOID: "Anulada",
  };
  return labels[status] || status;
}

export function invoiceStatusTone(status: string): string {
  const tones: Record<string, string> = {
    DRAFT: "draft",
    ISSUED: "issued",
    PARTIALLY_PAID: "partial",
    PAID: "paid",
    OVERDUE: "overdue",
    VOID: "void",
  };
  return tones[status] || "draft";
}

export function paymentMethodLabel(method: string): string {
  const labels: Record<string, string> = {
    CASH: "Efectivo",
    BANK_TRANSFER: "Transferencia bancaria",
    CARD: "Tarjeta",
    CHECK: "Cheque",
    OTHER: "Otro",
  };
  return labels[method] || method;
}

export function depositStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    PENDING: "Pendiente",
    RECEIVED: "Recibido",
    PARTIALLY_SETTLED: "Liquidación parcial",
    RETURNED: "Devuelto",
    RETAINED: "Retenido",
    SETTLED: "Liquidado",
  };
  return labels[status] || status;
}

export function monthLabel(value: string): string {
  const date = new Date(`${value}-01T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("es-SV", { month: "short", year: "2-digit" }).format(date);
}
