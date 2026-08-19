"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ReactNode, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { ThemeToggle } from "@/components/ThemeToggle";
import { api } from "@/lib/api";
import type { OperationAlertsResult, Permission } from "@/lib/types";

type IconName =
  | "dashboard"
  | "metrics"
  | "quotes"
  | "packages"
  | "storefront"
  | "inbox"
  | "customers"
  | "inventory"
  | "categories"
  | "audit"
  | "calendar"
  | "finance"
  | "payments"
  | "deposit"
  | "dte"
  | "sparkles"
  | "menu"
  | "close"
  | "bell"
  | "settings"
  | "team"
  | "user"
  | "logout"
  | "chevron";

function Icon({ name, size = 20 }: { name: IconName; size?: number }) {
  const common = {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };
  const paths: Record<IconName, ReactNode> = {
    dashboard: <><rect x="3" y="3" width="7" height="7" rx="2" /><rect x="14" y="3" width="7" height="7" rx="2" /><rect x="3" y="14" width="7" height="7" rx="2" /><rect x="14" y="14" width="7" height="7" rx="2" /></>,
    metrics: <><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /><path d="m3 7 6-4 6 6 6-5" /></>,
    quotes: <><path d="M6 3h9l3 3v15H6z" /><path d="M14 3v4h4M9 11h6M9 15h6" /></>,
    packages: <><path d="m12 3 8 4-8 4-8-4z" /><path d="m4 12 8 4 8-4" /><path d="m4 17 8 4 8-4" /></>,
    storefront: <><path d="M3 10h18" /><path d="m5 10 1-6h12l1 6" /><path d="M5 10v10h14V10" /><path d="M9 20v-6h6v6" /><path d="M3 10a3 3 0 0 0 6 0 3 3 0 0 0 6 0 3 3 0 0 0 6 0" /></>,
    inbox: <><path d="M4 4h16v16H4z" /><path d="M4 14h4l2 3h4l2-3h4" /><path d="M8 8h8M8 11h8" /></>,
    customers: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /></>,
    inventory: <><path d="M4 7.5 12 3l8 4.5v9L12 21l-8-4.5z" /><path d="m4.3 7.7 7.7 4.4 7.7-4.4M12 12v9" /></>,
    categories: <path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z" />,
    audit: <><path d="M12 3a9 9 0 1 0 9 9" /><path d="M12 7v5l3 2M18 3h3v3" /></>,
    calendar: <><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M16 3v4M8 3v4M3 10h18" /></>,
    finance: <><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /><path d="m3 7 6-4 6 6 6-5" /></>,
    payments: <><rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 10h18M7 15h3" /></>,
    deposit: <><path d="M4 8h16v11H4z" /><path d="M7 8V5h10v3M8 13h8M8 16h5" /></>,
    dte: <><path d="M6 3h9l3 3v15H6z" /><path d="M14 3v4h4" /><path d="M9 11h6M9 15h3" /><path d="m14.5 16 1.4 1.4 2.6-3" /></>,
    sparkles: <><path d="m12 3 1.2 3.2L16.5 7.5l-3.3 1.3L12 12l-1.2-3.2-3.3-1.3 3.3-1.3z" /><path d="m18 13 .8 2.2L21 16l-2.2.8L18 19l-.8-2.2L15 16l2.2-.8zM6 14l.7 1.8 1.8.7-1.8.7L6 19l-.7-1.8-1.8-.7 1.8-.7z" /></>,
    menu: <path d="M4 7h16M4 12h16M4 17h16" />,
    close: <path d="m6 6 12 12M18 6 6 18" />,
    bell: <><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" /><path d="M10 21h4" /></>,
    settings: <><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21h-4v-.09A1.7 1.7 0 0 0 8.6 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H3v-4h.09A1.7 1.7 0 0 0 4.6 8.6a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V3h4v.09A1.7 1.7 0 0 0 15.4 4.6a1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.4 9c.1.37.31.71.6 1 .29.29.63.5 1 .6h.09v4H21a1.7 1.7 0 0 0-1.6.4z" /></>,
    team: <><circle cx="8" cy="8" r="3" /><circle cx="17" cy="9" r="2" /><path d="M3 20v-2a5 5 0 0 1 10 0v2M14 20v-1a4 4 0 0 1 7-2.6" /></>,
    user: <><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></>,
    logout: <><path d="M10 17l5-5-5-5M15 12H3" /><path d="M14 3h6v18h-6" /></>,
    chevron: <path d="m8 10 4 4 4-4" />,
  };
  return <svg {...common}>{paths[name]}</svg>;
}

type NavItem = { href: string; label: string; icon: IconName; permission?: Permission };

const operationsNavigation: NavItem[] = [
  { href: "/", label: "Dashboard", icon: "dashboard", permission: "operations.read" },
  { href: "/metrics", label: "Métricas", icon: "metrics", permission: "operations.read" },
  { href: "/demo", label: "Demo guiada", icon: "sparkles", permission: "operations.read" },
  { href: "/assistant", label: "WhatsApp AI", icon: "inbox", permission: "assistant.read" },
  { href: "/calendar", label: "Calendario", icon: "calendar", permission: "operations.read" },
  { href: "/packages", label: "Paquetes", icon: "packages", permission: "package.read" },
  { href: "/quotes", label: "Cotizaciones", icon: "quotes", permission: "quote.read" },
  { href: "/quote-requests", label: "Solicitudes web", icon: "inbox", permission: "quote_request.read" },
  { href: "/reservations", label: "Reservas", icon: "calendar", permission: "reservation.read" },
  { href: "/customers", label: "Clientes", icon: "customers", permission: "customer.read" },
];
const inventoryNavigation: NavItem[] = [
  { href: "/inventory", label: "Inventario", icon: "inventory", permission: "inventory.read" },
  { href: "/categories", label: "Categorías", icon: "categories", permission: "catalog.read" },
];
const financeNavigation: NavItem[] = [
  { href: "/billing", label: "Finanzas", icon: "finance", permission: "billing.read" },
  { href: "/invoices", label: "Facturas", icon: "quotes", permission: "billing.read" },
  { href: "/dte", label: "DTE", icon: "dte", permission: "fiscal.read" },
  { href: "/payments", label: "Pagos", icon: "payments", permission: "payment.read" },
  { href: "/security-deposits", label: "Depósitos", icon: "deposit", permission: "payment.read" },
];
const systemNavigation: NavItem[] = [
  { href: "/settings/public-catalog", label: "Catálogo público", icon: "storefront", permission: "public_catalog.read" },
  { href: "/settings/quote-portal", label: "Portal de cotización", icon: "quotes", permission: "quote.read" },
  { href: "/settings/billing", label: "Facturación", icon: "finance", permission: "billing.read" },
  { href: "/settings/dte", label: "Integración DTE", icon: "dte", permission: "fiscal.read" },
  { href: "/audit", label: "Auditoría", icon: "audit", permission: "audit.read" },
  { href: "/settings/team", label: "Equipo", icon: "team", permission: "team.manage" },
  { href: "/settings/organization", label: "Organización", icon: "settings", permission: "tenant.manage" },
];

function pageTitle(pathname: string): string {
  if (pathname === "/metrics") return "Métricas operativas";
  if (pathname === "/demo") return "Demo comercial";
  if (pathname === "/assistant") return "WhatsApp Sales Assistant";
  if (pathname === "/calendar") return "Calendario operacional";
  if (pathname === "/reservations/new") return "Nueva reserva";
  if (pathname.startsWith("/reservations/")) return "Detalle de reserva";
  if (pathname === "/reservations") return "Reservas";
  if (pathname === "/packages/new") return "Nuevo paquete";
  if (pathname.startsWith("/packages/")) return "Detalle de paquete";
  if (pathname === "/packages") return "Paquetes";
  if (pathname === "/quotes/new") return "Nueva cotización";
  if (pathname.endsWith("/edit") && pathname.startsWith("/quotes/")) return "Editar cotización";
  if (pathname.startsWith("/quotes/")) return "Detalle de cotización";
  if (pathname === "/quotes") return "Cotizaciones";
  if (pathname.startsWith("/quote-requests/")) return "Detalle de solicitud web";
  if (pathname === "/quote-requests") return "Solicitudes web";
  if (pathname === "/billing") return "Dashboard financiero";
  if (pathname === "/invoices/new") return "Nueva factura";
  if (pathname.endsWith("/print") && pathname.startsWith("/invoices/")) return "Documento de factura";
  if (pathname.startsWith("/invoices/")) return "Detalle de factura";
  if (pathname === "/invoices") return "Facturas";
  if (pathname.startsWith("/dte/")) return "Detalle DTE";
  if (pathname === "/dte") return "Documentos tributarios electrónicos";
  if (pathname.startsWith("/payments/")) return "Detalle de pago";
  if (pathname === "/payments") return "Pagos";
  if (pathname === "/security-deposits") return "Depósitos de garantía";
  if (pathname.startsWith("/customers/")) return "Perfil del cliente";
  if (pathname === "/customers") return "Clientes";
  if (pathname.startsWith("/inventory/")) return "Detalle de inventario";
  if (pathname === "/inventory") return "Inventario";
  if (pathname === "/categories") return "Categorías";
  if (pathname === "/audit") return "Auditoría";
  if (pathname === "/settings/public-catalog") return "Configuración de catálogo público";
  if (pathname === "/settings/quote-portal") return "Portal de cotización";
  if (pathname === "/settings/billing") return "Configuración de facturación";
  if (pathname === "/settings/dte") return "Integración DTE";
  if (pathname === "/settings/team") return "Equipo y acceso";
  if (pathname === "/settings/organization") return "Configuración de organización";
  if (pathname === "/settings/profile") return "Mi perfil";
  return "Dashboard";
}

function initials(value: string): string {
  return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "RS";
}

function NavigationLinks({ items, pathname, can }: { items: NavItem[]; pathname: string; can: (permission: Permission) => boolean }) {
  return items.filter((item) => !item.permission || can(item.permission)).map((item) => {
    const active = item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
    return <Link key={item.href} href={item.href} className={`nav-link ${active ? "active" : ""}`}><Icon name={item.icon} /><span>{item.label}</span></Link>;
  });
}

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { me, can, selectWorkspace, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [workspaceOpen, setWorkspaceOpen] = useState(false);
  const [alertsCount, setAlertsCount] = useState(0);

  useEffect(() => {
    let active = true;
    const loadAlerts = () => {
      if (!can("operations.read")) return;
      api<OperationAlertsResult>("/api/v1/operations/alerts")
        .then((result) => { if (active) setAlertsCount(result.counts.total); })
        .catch(() => { if (active) setAlertsCount(0); });
    };
    loadAlerts();
    const timer = window.setInterval(loadAlerts, 60_000);
    return () => { active = false; window.clearInterval(timer); };
  }, [pathname, can]);

  useEffect(() => { setMenuOpen(false); setProfileOpen(false); setWorkspaceOpen(false); }, [pathname]);

  const workspace = me?.active_workspace;
  const tenantInitials = useMemo(() => initials(workspace?.name || "RentStage"), [workspace?.name]);
  const userInitials = useMemo(() => initials(me?.user.display_name || me?.user.email || "User"), [me?.user.display_name, me?.user.email]);

  async function switchWorkspace(tenantID: string) {
    if (tenantID === workspace?.tenant_id) return;
    await selectWorkspace(tenantID);
    setWorkspaceOpen(false);
    router.push("/");
  }

  return (
    <div className="app-shell">
      <aside className={`sidebar ${menuOpen ? "sidebar-open" : ""}`}>
        <div className="brand-row">
          <Link href="/" className="brand" aria-label="RentStage dashboard"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Rental operations</small></span></Link>
          <button className="icon-button sidebar-close" onClick={() => setMenuOpen(false)} aria-label="Cerrar menú"><Icon name="close" /></button>
        </div>
        <nav className="sidebar-nav" aria-label="Navegación principal">
          <p className="nav-section-label">OPERACIONES</p><NavigationLinks items={operationsNavigation} pathname={pathname} can={can} />
          <p className="nav-section-label nav-section-spaced">INVENTARIO</p><NavigationLinks items={inventoryNavigation} pathname={pathname} can={can} />
          <p className="nav-section-label nav-section-spaced">FINANZAS</p><NavigationLinks items={financeNavigation} pathname={pathname} can={can} />
          <p className="nav-section-label nav-section-spaced">SISTEMA</p><NavigationLinks items={systemNavigation} pathname={pathname} can={can} />
        </nav>
        <div className="sidebar-footer workspace-switcher">
          <button className="tenant-card tenant-card-button" onClick={() => setWorkspaceOpen((value) => !value)}>
            <span className="tenant-avatar">{tenantInitials}</span><span className="tenant-copy"><strong>{workspace?.name || "Sin workspace"}</strong><small>{workspace?.role || "Sin rol"}</small></span><Icon name="chevron" size={16} />
          </button>
          {workspaceOpen && <div className="workspace-menu panel">
            <p>WORKSPACES</p>
            {me?.workspaces.map((item) => <button key={item.tenant_id} className={item.tenant_id === workspace?.tenant_id ? "selected" : ""} onClick={() => void switchWorkspace(item.tenant_id)}><span>{initials(item.name)}</span><div><strong>{item.name}</strong><small>{item.role}</small></div></button>)}
            <Link href="/workspaces">Administrar workspaces</Link>
          </div>}
        </div>
      </aside>
      {menuOpen && <button className="sidebar-scrim" onClick={() => setMenuOpen(false)} aria-label="Cerrar menú" />}
      <div className="main-column">
        <header className="topbar">
          <div className="topbar-left"><button className="icon-button mobile-menu" onClick={() => setMenuOpen(true)} aria-label="Abrir menú"><Icon name="menu" /></button><div><p className="eyebrow">RENTSTAGE ADMIN</p><h1>{pageTitle(pathname)}</h1></div></div>
          <div className="topbar-actions">
            <ThemeToggle />
            {can("operations.read") && <Link className="icon-button notification-button" href="/calendar#operations-alerts" aria-label={`${alertsCount} alertas operativas`}><Icon name="bell" />{alertsCount > 0 && <span className="notification-count">{alertsCount > 99 ? "99+" : alertsCount}</span>}</Link>}
            <div className="profile-menu-wrap">
              <button className="profile-chip profile-chip-button" onClick={() => setProfileOpen((value) => !value)}><span>{userInitials}</span><div><strong>{me?.user.display_name || "Usuario"}</strong><small>{workspace?.role || "Sin rol"}</small></div><Icon name="chevron" size={15} /></button>
              {profileOpen && <div className="profile-menu panel"><div className="profile-menu-email">{me?.user.email}</div><Link href="/settings/profile"><Icon name="user" size={17} />Mi perfil</Link><Link href="/workspaces"><Icon name="inventory" size={17} />Cambiar workspace</Link><button onClick={() => void logout()}><Icon name="logout" size={17} />Cerrar sesión</button></div>}
            </div>
          </div>
        </header>
        <main className="content">{children}</main>
      </div>
    </div>
  );
}
