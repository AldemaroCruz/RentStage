"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, api } from "@/lib/api";
import { assignmentStateLabel, returnConditionLabel } from "@/lib/format";
import type { ReservationDetail, WarehouseInventory } from "@/lib/types";

export function WarehousePanel({
  reservation,
  onChanged,
  canOperate,
}: {
  reservation: ReservationDetail;
  onChanged: (reservation: ReservationDetail) => void;
  canOperate: boolean;
}) {
  const [warehouse, setWarehouse] = useState<WarehouseInventory | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [acting, setActing] = useState("");
  const [selectedAssets, setSelectedAssets] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api<WarehouseInventory>(`/api/v1/reservations/${reservation.id}/warehouse`);
      setWarehouse(response);
      setError("");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible cargar la preparación de inventario.");
    } finally {
      setLoading(false);
    }
  }, [reservation.id, reservation.status, reservation.assigned_asset_count]);

  useEffect(() => {
    void load();
  }, [load]);

  const progress = useMemo(() => {
    if (!warehouse || warehouse.required_asset_count === 0) return 100;
    return Math.min(100, Math.round((warehouse.assigned_asset_count / warehouse.required_asset_count) * 100));
  }, [warehouse]);

  async function assign(reservationItemID: string) {
    const assetID = selectedAssets[reservationItemID];
    if (!assetID) return;
    setActing(`assign:${assetID}`);
    setError("");
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${reservation.id}/assets`, {
        method: "POST",
        body: JSON.stringify({ asset_id: assetID }),
      });
      onChanged(updated);
      setSelectedAssets((current) => ({ ...current, [reservationItemID]: "" }));
      await load();
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible asignar la unidad física.");
    } finally {
      setActing("");
    }
  }

  async function unassign(assetID: string, assetCode: string) {
    if (!window.confirm(`¿Retirar ${assetCode} de esta preparación?`)) return;
    setActing(`unassign:${assetID}`);
    setError("");
    try {
      const updated = await api<ReservationDetail>(`/api/v1/reservations/${reservation.id}/assets/${assetID}`, {
        method: "DELETE",
      });
      onChanged(updated);
      await load();
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible retirar la unidad física.");
    } finally {
      setActing("");
    }
  }

  if (loading && !warehouse) {
    return <section className="panel warehouse-panel warehouse-loading">Cargando operación de almacén…</section>;
  }

  if (!warehouse) {
    return (
      <section className="panel warehouse-panel">
        <header className="warehouse-panel-header"><div><p className="eyebrow">WAREHOUSE OPERATIONS</p><h2>Preparación física</h2></div></header>
        <div className="form-alert">{error || "No se encontró información de preparación."}</div>
      </section>
    );
  }

  return (
    <section className="panel warehouse-panel">
      <header className="warehouse-panel-header">
        <div>
          <p className="eyebrow">WAREHOUSE OPERATIONS</p>
          <h2>Preparación de inventario</h2>
          <p>Asigna las unidades físicas que realmente saldrán del almacén.</p>
        </div>
        <div className={`warehouse-progress-copy ${warehouse.complete ? "warehouse-progress-complete" : ""}`}>
          <strong>{warehouse.assigned_asset_count} / {warehouse.required_asset_count}</strong>
          <span>{warehouse.complete ? "Preparación completa" : "Unidades asignadas"}</span>
        </div>
      </header>

      <div className="warehouse-progress-track" aria-label={`${progress}% de inventario asignado`}>
        <span style={{ width: `${progress}%` }} />
      </div>

      {error && <div className="form-alert warehouse-alert">{error}</div>}

      {reservation.status === "PENDING" && (
        <div className="warehouse-guidance"><span>1</span><div><strong>Primero confirma la reserva</strong><p>La asignación física comienza después de confirmar y mover la reserva a preparación.</p></div></div>
      )}
      {reservation.status === "CONFIRMED" && (
        <div className="warehouse-guidance"><span>2</span><div><strong>Inicia la preparación</strong><p>Usa “Comenzar preparación” para habilitar la selección de unidades físicas.</p></div></div>
      )}

      <div className="warehouse-item-list">
        {warehouse.items.map((item) => (
          <article className={`warehouse-item-card ${item.missing_quantity > 0 ? "warehouse-item-incomplete" : "warehouse-item-complete"}`} key={item.reservation_item_id}>
            <header>
              <div>
                <h3>{item.resource_name}</h3>
                <p>{item.track_individual_assets ? "Requiere trazabilidad por unidad" : "Recurso sin inventario individual"}</p>
              </div>
              <span className="warehouse-quantity-badge">{item.assigned_quantity} / {item.required_quantity}</span>
            </header>

            {!item.track_individual_assets ? (
              <div className="warehouse-untracked-note">Este recurso no requiere asignación de un asset físico.</div>
            ) : (
              <>
                <div className="warehouse-assignment-list">
                  {item.assignments.length === 0 ? (
                    <div className="warehouse-empty-assignment">Aún no hay unidades asignadas.</div>
                  ) : item.assignments.map((assignment) => (
                    <div className="warehouse-assignment-row" key={assignment.assignment_id}>
                      <div className="warehouse-asset-mark">{assignment.asset_code.slice(-2)}</div>
                      <div className="warehouse-assignment-copy">
                        <strong>{assignment.asset_code}</strong>
                        <small>{assignment.serial_number || "Sin número de serie"}</small>
                        {assignment.return_condition && (
                          <span className={`return-condition return-condition-${assignment.return_condition.toLowerCase()}`}>
                            {returnConditionLabel(assignment.return_condition)}
                          </span>
                        )}
                      </div>
                      <span className={`assignment-state assignment-state-${assignment.state.toLowerCase()}`}>
                        {assignmentStateLabel(assignment.state)}
                      </span>
                      {canOperate && warehouse.can_manage_assignments && (
                        <button
                          className="warehouse-remove-button"
                          disabled={Boolean(acting)}
                          onClick={() => void unassign(assignment.asset_id, assignment.asset_code)}
                          title="Desasignar unidad"
                        >
                          {acting === `unassign:${assignment.asset_id}` ? "…" : "×"}
                        </button>
                      )}
                    </div>
                  ))}
                </div>

                {canOperate && warehouse.can_manage_assignments && item.missing_quantity > 0 && (
                  <div className="warehouse-assign-control">
                    {item.available_assets.length > 0 ? (
                      <>
                        <label className="field">
                          <span>Unidad disponible</span>
                          <select
                            value={selectedAssets[item.reservation_item_id] || ""}
                            onChange={(event) => setSelectedAssets((current) => ({ ...current, [item.reservation_item_id]: event.target.value }))}
                          >
                            <option value="">Selecciona una unidad física</option>
                            {item.available_assets.map((asset) => (
                              <option value={asset.id} key={asset.id}>
                                {asset.asset_code}{asset.serial_number ? ` · ${asset.serial_number}` : ""}
                              </option>
                            ))}
                          </select>
                        </label>
                        <button
                          className="button button-primary"
                          disabled={!selectedAssets[item.reservation_item_id] || Boolean(acting)}
                          onClick={() => void assign(item.reservation_item_id)}
                        >
                          {acting.startsWith("assign:") ? "Asignando…" : "Asignar unidad"}
                        </button>
                      </>
                    ) : (
                      <div className="warehouse-no-candidates"><strong>Sin unidades elegibles</strong><span>Revisa el estado físico o conflictos con otras reservas.</span></div>
                    )}
                  </div>
                )}
              </>
            )}
          </article>
        ))}
      </div>

      {warehouse.complete && warehouse.required_asset_count > 0 && reservation.status === "PREPARING" && (
        <div className="warehouse-complete-banner"><span>✓</span><div><strong>Pedido completamente asignado</strong><p>Ya puedes marcar la reserva como lista para salir.</p></div></div>
      )}
    </section>
  );
}
