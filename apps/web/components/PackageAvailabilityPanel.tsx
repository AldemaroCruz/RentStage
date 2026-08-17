"use client";

import { useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { formatCurrency } from "@/lib/format";
import type { PackageAvailabilityResult, RentalPackage } from "@/lib/types";

type Props = { rentalPackage: RentalPackage };

function localInput(date: Date): string {
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}

function initialPeriod() {
  const start = new Date();
  start.setDate(start.getDate() + 1);
  start.setHours(14, 0, 0, 0);
  const end = new Date(start);
  end.setHours(23, 0, 0, 0);
  return { start: localInput(start), end: localInput(end) };
}

export function PackageAvailabilityPanel({ rentalPackage }: Props) {
  const defaults = useMemo(initialPeriod, []);
  const [startAt, setStartAt] = useState(defaults.start);
  const [endAt, setEndAt] = useState(defaults.end);
  const [sets, setSets] = useState("1");
  const [result, setResult] = useState<PackageAvailabilityResult | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setResult(null);
    setError("");
  }, [rentalPackage.id, rentalPackage.updated_at]);

  async function check() {
    setChecking(true);
    setError("");
    setResult(null);
    try {
      const start = new Date(startAt);
      const end = new Date(endAt);
      if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end.getTime() <= start.getTime()) {
        setError("Selecciona un período válido; la fecha final debe ser posterior a la inicial.");
        return;
      }
      const quantity = Math.max(1, Math.min(100, Number(sets) || 1));
      const availability = await api<PackageAvailabilityResult>(`/api/v1/packages/${rentalPackage.id}/availability`, {
        method: "POST",
        body: JSON.stringify({
          start_at: start.toISOString(),
          end_at: end.toISOString(),
          quantity,
        }),
      });
      setResult(availability);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible validar disponibilidad.");
    } finally {
      setChecking(false);
    }
  }

  return (
    <section className="panel package-availability-panel">
      <div className="panel-header">
        <div>
          <p className="eyebrow">AVAILABILITY ENGINE</p>
          <h2>Probar disponibilidad</h2>
          <p>La consulta usa cantidades del paquete y reservas que ya bloquean inventario.</p>
        </div>
        <div className="package-availability-price"><small>Precio por paquete</small><strong>{formatCurrency(rentalPackage.effective_price)}</strong></div>
      </div>
      <div className="package-availability-controls">
        <label className="field"><span>Desde</span><input type="datetime-local" value={startAt} onChange={(event) => setStartAt(event.target.value)} /></label>
        <label className="field"><span>Hasta</span><input type="datetime-local" value={endAt} onChange={(event) => setEndAt(event.target.value)} /></label>
        <label className="field"><span>Paquetes</span><input type="number" min="1" max="100" value={sets} onChange={(event) => setSets(event.target.value)} /></label>
        <button className="button button-primary" type="button" disabled={checking || !rentalPackage.ready} onClick={() => void check()}>{checking ? "Validando…" : "Validar"}</button>
      </div>
      {!rentalPackage.ready && <div className="form-alert package-availability-alert">Este paquete está archivado, vacío o contiene recursos archivados. Corrígelo antes de validar disponibilidad.</div>}
      {error && <div className="form-alert package-availability-alert">{error}</div>}
      {result && (
        <div className={`package-availability-result ${result.available ? "available" : "conflict"}`}>
          <div className="package-availability-status">
            <span>{result.available ? "✓" : "!"}</span>
            <div><strong>{result.available ? "Paquete disponible" : "No hay capacidad suficiente"}</strong><small>{result.available ? "Todas las cantidades pueden cubrirse en este período." : "Revisa los recursos marcados antes de prometer la fecha."}</small></div>
          </div>
          <div className="package-availability-items">
            {result.items.map((item) => (
              <div key={item.resource_id} className={item.can_fulfill ? "ok" : "missing"}>
                <span>{item.resource_name}</span>
                <strong>{item.available_quantity} / {item.requested_quantity}</strong>
                <small>{item.can_fulfill ? "Disponible" : `Faltan ${Math.max(0, item.requested_quantity - item.available_quantity)}`}</small>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
