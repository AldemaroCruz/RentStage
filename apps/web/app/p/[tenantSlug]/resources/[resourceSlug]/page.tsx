"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { PublicCatalogShell } from "@/components/PublicCatalogShell";
import { api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { PublicResourceResponse } from "@/lib/types";

export default function PublicResourcePage() {
  const params = useParams<{ tenantSlug: string; resourceSlug: string }>();
  const [data, setData] = useState<PublicResourceResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<PublicResourceResponse>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}/resources/${encodeURIComponent(params.resourceSlug)}`)
      .then(setData)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el recurso."))
      .finally(() => setLoading(false));
  }, [params.tenantSlug, params.resourceSlug]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <div className="public-page-state"><span className="public-loader" /><p>Cargando recurso…</p></div>;
  if (error || !data) return <div className="public-page-state public-page-error"><strong>Recurso no disponible</strong><p>{error || "Este recurso ya no está publicado."}</p><button type="button" onClick={load}>Reintentar</button></div>;

  const item = data.resource;
  return (
    <PublicCatalogShell tenant={data.tenant} settings={data.settings}>
      <div className="public-detail-breadcrumb"><Link href={`/p/${data.tenant.slug}`}>Catálogo</Link><span>/</span><span>{item.name}</span></div>
      <section className="public-detail-hero resource">
        <div className="public-detail-media">
          {item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <span className="placeholder">{item.name.slice(0, 2).toUpperCase()}</span>}
          {item.featured && <em>Recurso destacado</em>}
        </div>
        <div className="public-detail-copy">
          <p className="public-kicker">{item.category_name || item.resource_type}</p>
          <h1>{item.name}</h1>
          <p className="public-detail-description">{item.description || "Recurso disponible para cotizar como parte de una solución para tu evento."}</p>
          <div className="public-detail-price">
            {item.base_price !== undefined ? <><small>Precio base por {pricingUnitLabel(item.pricing_unit)}</small><strong>{formatCurrency(item.base_price, data.tenant.currency)}</strong></> : <><small>Precio</small><strong>Consultar</strong></>}
          </div>
          {data.settings.quote_requests_enabled && <Link className="public-button primary wide" href={`/p/${data.tenant.slug}/request`}>Solicitar una propuesta</Link>}
          <small className="public-detail-note">El equipo puede incorporarlo a una cotización junto con otros recursos.</small>
        </div>
      </section>
      <section className="public-catalog-cta compact">
        <div><p>ARMEMOS TU EVENTO</p><h2>Combínalo con un paquete completo</h2><span>Explora las opciones publicadas o envía los detalles de lo que necesitas.</span></div>
        <Link className="public-button secondary dark" href={`/p/${data.tenant.slug}`}>Volver al catálogo</Link>
      </section>
    </PublicCatalogShell>
  );
}
