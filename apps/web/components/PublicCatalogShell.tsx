"use client";

import Link from "next/link";
import type { CSSProperties, ReactNode } from "react";
import { PublicWebChat } from "@/components/PublicWebChat";
import type { PublicCatalogViewSettings, PublicTenant } from "@/lib/types";

function initials(value: string): string {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase() || "RS";
}

export function PublicCatalogShell({
  tenant,
  settings,
  children,
}: {
  tenant: PublicTenant;
  settings: PublicCatalogViewSettings;
  children: ReactNode;
}) {
  const style = { "--catalog-accent": settings.accent_color || "#6558e8" } as CSSProperties;
  const catalogPath = `/p/${tenant.slug}`;

  return (
    <div className="public-catalog-shell" style={style}>
      <header className="public-catalog-header">
        <div className="public-catalog-header-inner">
          <Link href={catalogPath} className="public-catalog-brand" aria-label={`Catálogo de ${tenant.name}`}>
            {tenant.logo_url ? (
              <span className="public-catalog-logo image" style={{ backgroundImage: `url(${tenant.logo_url})` }} />
            ) : (
              <span className="public-catalog-logo">{initials(tenant.name)}</span>
            )}
            <span>
              <strong>{tenant.name}</strong>
              <small>Catálogo de alquiler</small>
            </span>
          </Link>
          <nav className="public-catalog-nav" aria-label="Navegación del catálogo">
            <Link href={catalogPath}>Catálogo</Link>
            {settings.quote_requests_enabled && (
              <Link className="public-catalog-nav-cta" href={`${catalogPath}/request`}>
                Solicitar cotización
              </Link>
            )}
          </nav>
        </div>
      </header>

      <main className="public-catalog-main">{children}</main>

      <footer className="public-catalog-footer">
        <div>
          <Link href={catalogPath} className="public-catalog-footer-brand">
            {tenant.name}
          </Link>
          <p>Catálogo publicado con RentStage.</p>
        </div>
        <div className="public-catalog-contact">
          {settings.contact_phone && <a href={`tel:${settings.contact_phone}`}>{settings.contact_phone}</a>}
          {settings.contact_email && <a href={`mailto:${settings.contact_email}`}>{settings.contact_email}</a>}
          {settings.contact_address && <span>{settings.contact_address}</span>}
        </div>
      </footer>

      {settings.web_chat_enabled && <PublicWebChat tenant={tenant} settings={settings} />}
    </div>
  );
}
