export function metricBarPercent(value: number, maximum: number): number {
  if (!Number.isFinite(value) || !Number.isFinite(maximum) || value <= 0 || maximum <= 0) return 0;
  return Math.min(100, Math.max(4, value / maximum * 100));
}

export function responseTimeLabel(minutes: number, samples: number): string {
  if (!Number.isFinite(minutes) || samples <= 0) return "Sin muestra";
  if (minutes < 1) return "< 1 min";
  if (minutes < 60) return `${Math.round(minutes)} min`;
  const hours = Math.floor(minutes / 60);
  const remainder = Math.round(minutes % 60);
  return remainder > 0 ? `${hours} h ${remainder} min` : `${hours} h`;
}

export function customerSourceLabel(source: string): string {
  const labels: Record<string, string> = {
    WEB: "Catálogo web",
    WHATSAPP: "WhatsApp",
    MANUAL: "Registro manual",
    IMPORT: "Importación",
  };
  return labels[source] || source;
}
