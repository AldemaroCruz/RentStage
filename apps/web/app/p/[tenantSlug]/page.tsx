"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { PublicCatalogShell } from "@/components/PublicCatalogShell";
import { api } from "@/lib/api";
import { formatCurrency, pricingUnitLabel } from "@/lib/format";
import type { PublicCatalog } from "@/lib/types";

export default function PublicCatalogPage() {
  const params = useParams<{ tenantSlug: string }>();
  const [catalog, setCatalog] = useState<PublicCatalog | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    api<PublicCatalog>(`/api/v1/public/catalogs/${encodeURIComponent(params.tenantSlug)}`)
      .then(setCatalog)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible abrir este catálogo."))
      .finally(() => setLoading(false));
  }, [params.tenantSlug]);

  useEffect(() => { load(); }, [load]);

  if (loading) {
    return <div className="public-page-state"><span className="public-loader" /><p>Cargando catálogo…</p></div>;
  }
  if (error || !catalog) {
    return (
      <div className="public-page-state public-page-error">
        <strong>Catálogo no disponible</strong>
        <p>{error || "La empresa no ha publicado su catálogo todavía."}</p>
        <button type="button" onClick={load}>Reintentar</button>
      </div>
    );
  }

  const featuredPackages = catalog.packages.filter((item) => item.featured);
  const regularPackages = catalog.packages.filter((item) => !item.featured);
  const featuredResources = catalog.resources.filter((item) => item.featured);
  const regularResources = catalog.resources.filter((item) => !item.featured);

  return (
    <PublicCatalogShell tenant={catalog.tenant} settings={catalog.settings}>
      <section
        className={`public-catalog-hero ${catalog.settings.cover_image_url ? "has-cover" : ""}`}
        style={catalog.settings.cover_image_url ? { backgroundImage: `linear-gradient(100deg, rgba(16,18,35,.94), rgba(16,18,35,.48)), url(${catalog.settings.cover_image_url})` } : undefined}
      >
        <div className="public-catalog-hero-copy">
          <p className="public-kicker">EQUIPO Y SOLUCIONES PARA EVENTOS</p>
          <h1>{catalog.settings.headline || `Haz realidad tu evento con ${catalog.tenant.name}`}</h1>
          <p>{catalog.settings.description || "Explora nuestros paquetes y solicita una propuesta adaptada a tu evento."}</p>
          <div className="public-catalog-hero-actions">
            {catalog.settings.quote_requests_enabled && (
              <Link className="public-button primary" href={`/p/${catalog.tenant.slug}/request`}>
                Solicitar cotización
              </Link>
            )}
            <a className="public-button secondary" href="#paquetes">Ver paquetes</a>
          </div>
        </div>
        <div className="public-catalog-hero-facts">
          <div><strong>{catalog.packages.length}</strong><span>paquetes disponibles</span></div>
          {catalog.settings.show_resources && <div><strong>{catalog.resources.length}</strong><span>recursos publicados</span></div>}
          <div><strong>Respuesta humana</strong><span>la solicitud llega al equipo de la empresa</span></div>
        </div>
      </section>

      <section className="public-catalog-section" id="paquetes">
        <div className="public-section-heading">
          <div><p>PAQUETES</p><h2>Soluciones listas para cotizar</h2></div>
          {catalog.settings.quote_requests_enabled && <Link href={`/p/${catalog.tenant.slug}/request`}>Crear mi solicitud →</Link>}
        </div>
        {catalog.packages.length === 0 ? (
          <div className="public-empty"><strong>Aún no hay paquetes publicados</strong><p>Vuelve pronto o utiliza los datos de contacto para consultar opciones.</p></div>
        ) : (
          <div className="public-package-grid">
            {[...featuredPackages, ...regularPackages].map((item) => (
              <article className={`public-package-card ${item.featured ? "featured" : ""}`} key={item.slug}>
                <Link href={`/p/${catalog.tenant.slug}/packages/${item.slug}`} className="public-card-media">
                  {item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <span className="placeholder">{item.name.slice(0, 2).toUpperCase()}</span>}
                  {item.featured && <em>Destacado</em>}
                </Link>
                <div className="public-package-card-body">
                  <div className="public-package-card-title">
                    <div>
                      {item.guest_capacity && <small>HASTA {item.guest_capacity} PERSONAS</small>}
                      <h3><Link href={`/p/${catalog.tenant.slug}/packages/${item.slug}`}>{item.name}</Link></h3>
                    </div>
                    {item.effective_price !== undefined && <strong>{formatCurrency(item.effective_price, catalog.tenant.currency)}</strong>}
                  </div>
                  <p>{item.description || "Paquete preparado para cubrir las necesidades principales de tu evento."}</p>
                  <div className="public-package-meta"><span>{item.item_count} tipos de recurso</span><span>{item.total_quantity} unidades</span></div>
                  <div className="public-card-actions">
                    <Link href={`/p/${catalog.tenant.slug}/packages/${item.slug}`}>Ver detalles</Link>
                    {catalog.settings.quote_requests_enabled && <Link className="request" href={`/p/${catalog.tenant.slug}/request?package=${encodeURIComponent(item.slug)}`}>Cotizar</Link>}
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      {catalog.settings.show_resources && catalog.resources.length > 0 && (
        <section className="public-catalog-section public-resources-section" id="equipo">
          <div className="public-section-heading"><div><p>CATÁLOGO</p><h2>Equipo y servicios individuales</h2></div></div>
          <div className="public-resource-grid">
            {[...featuredResources, ...regularResources].map((item) => (
              <Link className="public-resource-card" href={`/p/${catalog.tenant.slug}/resources/${item.slug}`} key={item.slug}>
                <div className="public-resource-media">
                  {item.image_url ? <span style={{ backgroundImage: `url(${item.image_url})` }} /> : <span className="placeholder">{item.name.slice(0, 2).toUpperCase()}</span>}
                  {item.featured && <em>Destacado</em>}
                </div>
                <div><small>{item.category_name || item.resource_type}</small><h3>{item.name}</h3><p>{item.description}</p></div>
                <footer>
                  {item.base_price !== undefined ? <strong>{formatCurrency(item.base_price, catalog.tenant.currency)} <small>/ {pricingUnitLabel(item.pricing_unit)}</small></strong> : <strong>Consultar precio</strong>}
                  <span>Ver →</span>
                </footer>
              </Link>
            ))}
          </div>
        </section>
      )}

      <section className="public-catalog-cta">
        <div><p>¿TIENES UNA FECHA EN MENTE?</p><h2>Cuéntanos sobre tu evento</h2><span>Selecciona paquetes, comparte el período y recibe seguimiento del equipo.</span></div>
        {catalog.settings.quote_requests_enabled ? (
          <Link className="public-button primary" href={`/p/${catalog.tenant.slug}/request`}>Solicitar cotización</Link>
        ) : (
          <div className="public-contact-cta">
            {catalog.settings.contact_phone && <a href={`tel:${catalog.settings.contact_phone}`}>{catalog.settings.contact_phone}</a>}
            {catalog.settings.contact_email && <a href={`mailto:${catalog.settings.contact_email}`}>{catalog.settings.contact_email}</a>}
          </div>
        )}
      </section>
    </PublicCatalogShell>
  );
}
