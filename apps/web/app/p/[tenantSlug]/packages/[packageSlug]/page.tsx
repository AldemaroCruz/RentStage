"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { PublicCatalogShell } from "@/components/PublicCatalogShell";
import { api } from "@/lib/api";
import { formatCurrency } from "@/lib/format";
import type { PublicPackageResponse } from "@/lib/types";

export default function PublicPackagePage() {
  const params = useParams<{ tenantSlug: string; packageSlug: string }>();
  const [data, setData] = useState<PublicPackageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<PublicPackageResponse>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}/packages/${encodeURIComponent(params.packageSlug)}`)
      .then(setData)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el paquete."))
      .finally(() => setLoading(false));
  }, [params.tenantSlug, params.packageSlug]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <div className="public-page-state"><span className="public-loader" /><p>Cargando paquete…</p></div>;
  if (error || !data) return <div className="public-page-state public-page-error"><strong>Paquete no disponible</strong><p>{error || "Este paquete ya no está publicado."}</p><button type="button" onClick={load}>Reintentar</button></div>;

  const item = data.package;
  return (
    <PublicCatalogShell tenant={data.tenant} settings={data.settings}>
      <div className="public-detail-breadcrumb"><Link href={`/p/${data.tenant.slug}`}>Catálogo</Link><span>/</span><span>{item.name}</span></div>
      <section className="public-detail-hero">
        <div className="public-detail-media">
          {item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <span className="placeholder">{item.name.slice(0, 2).toUpperCase()}</span>}
          {item.featured && <em>Paquete destacado</em>}
        </div>
        <div className="public-detail-copy">
          <p className="public-kicker">PAQUETE COMERCIAL</p>
          <h1>{item.name}</h1>
          <p className="public-detail-description">{item.description || "Una solución lista para adaptar a tu fecha y evento."}</p>
          <div className="public-detail-facts">
            {item.guest_capacity && <div><strong>{item.guest_capacity}</strong><span>personas sugeridas</span></div>}
            <div><strong>{item.item_count}</strong><span>tipos de recurso</span></div>
            <div><strong>{item.total_quantity}</strong><span>unidades incluidas</span></div>
          </div>
          <div className="public-detail-price">
            {item.effective_price !== undefined ? <><small>Precio estimado del paquete</small><strong>{formatCurrency(item.effective_price, data.tenant.currency)}</strong></> : <><small>Precio</small><strong>Consultar</strong></>}
          </div>
          {data.settings.quote_requests_enabled && <Link className="public-button primary wide" href={`/p/${data.tenant.slug}/request?package=${encodeURIComponent(item.slug)}`}>Solicitar este paquete</Link>}
          <small className="public-detail-note">La disponibilidad se confirma para el período seleccionado durante la solicitud.</small>
        </div>
      </section>

      <section className="public-detail-section">
        <div className="public-section-heading"><div><p>INCLUYE</p><h2>Componentes del paquete</h2></div></div>
        <div className="public-package-item-list">
          {item.items.map((component, index) => (
            <article key={`${component.resource_name}-${index}`}>
              <span>{component.quantity}</span>
              <div><strong>{component.resource_name}</strong><p>{component.description || "Componente incluido."}</p></div>
            </article>
          ))}
        </div>
      </section>

      <section className="public-catalog-cta compact">
        <div><p>PERSONALIZA TU PROPUESTA</p><h2>¿Necesitas agregar o ajustar algo?</h2><span>Envía la solicitud con tus notas y el equipo podrá adaptar la cotización.</span></div>
        {data.settings.quote_requests_enabled && <Link className="public-button primary" href={`/p/${data.tenant.slug}/request?package=${encodeURIComponent(item.slug)}`}>Iniciar solicitud</Link>}
      </section>
    </PublicCatalogShell>
  );
}
