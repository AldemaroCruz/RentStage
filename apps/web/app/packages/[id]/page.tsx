"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { PackageAvailabilityPanel } from "@/components/PackageAvailabilityPanel";
import { PackageEditor } from "@/components/PackageEditor";
import { api } from "@/lib/api";
import type { RentalPackage } from "@/lib/types";

export default function PackageDetailPage() {
  const params = useParams<{ id: string }>();
  const { can } = useAuth();
  const [item, setItem] = useState<RentalPackage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<RentalPackage>(`/api/v1/packages/${params.id}`)
      .then(setItem)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el paquete."))
      .finally(() => setLoading(false));
  }, [params.id]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <div className="skeleton detail-skeleton" />;
  if (error || !item) return <section className="panel connection-panel"><span className="connection-icon">!</span><div><h2>Paquete no disponible</h2><p>{error || "No encontramos este paquete."}</p><Link href="/packages" className="text-link">← Volver a paquetes</Link></div></section>;

  return (
    <div className="page-stack">
      <div className="breadcrumbs"><Link href="/packages">Paquetes</Link><span>/</span><span>{item.name}</span></div>
      <section className="package-detail-strip panel">
        <div><p className="eyebrow">REUSABLE COMMERCIAL TEMPLATE</p><h2>{item.name}</h2><p>Al agregarlo a una cotización, RentStage crea líneas normales con cantidades y precios históricos.</p></div>
        <div className="package-detail-actions">
          {item.ready && can("quote.manage") && <Link className="button button-primary" href={`/quotes/new?package_id=${item.id}`}>Usar en cotización</Link>}
          {!can("package.manage") && <span className="read-only-pill">Solo lectura</span>}
        </div>
      </section>
      <PackageEditor key={item.id} initial={item} readOnly={!can("package.manage")} onSaved={setItem} />
      <PackageAvailabilityPanel rentalPackage={item} />
    </div>
  );
}
