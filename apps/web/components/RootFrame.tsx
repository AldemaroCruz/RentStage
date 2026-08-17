"use client";

import Link from "next/link";
import { ReactNode, Suspense, useEffect } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { AppShell } from "@/components/AppShell";
import { AuthProvider, useAuth } from "@/components/AuthProvider";
import type { Permission } from "@/lib/types";

const publicRoutes = new Set(["/login", "/signup"]);
const standaloneAuthenticated = ["/workspaces", "/onboarding"];

function isPublicPath(pathname: string): boolean {
  return publicRoutes.has(pathname) || pathname.startsWith("/invites/") || pathname.startsWith("/p/") || pathname === "/q";
}

function requiredPermission(pathname: string): Permission | null {
  if (pathname === "/") return "operations.read";
  if (pathname === "/calendar") return "operations.read";
  if (pathname === "/audit") return "audit.read";
  if (pathname === "/categories") return "catalog.read";
  if (pathname === "/inventory" || pathname.startsWith("/inventory/")) return "catalog.read";
  if (pathname === "/packages/new") return "package.manage";
  if (pathname === "/packages" || pathname.startsWith("/packages/")) return "package.read";
  if (pathname === "/customers" || pathname.startsWith("/customers/")) return "customer.read";
  if (pathname === "/quotes/new" || (pathname.startsWith("/quotes/") && pathname.endsWith("/edit"))) return "quote.manage";
  if (pathname === "/quotes" || pathname.startsWith("/quotes/")) return "quote.read";
  if (pathname === "/reservations/new") return "reservation.manage";
  if (pathname === "/reservations" || pathname.startsWith("/reservations/")) return "reservation.read";
  if (pathname === "/quote-requests" || pathname.startsWith("/quote-requests/")) return "quote_request.read";
  if (pathname === "/billing") return "billing.read";
  if (pathname === "/invoices/new") return "billing.manage";
  if (pathname === "/invoices" || pathname.startsWith("/invoices/")) return "billing.read";
  if (pathname === "/payments" || pathname.startsWith("/payments/")) return "payment.read";
  if (pathname === "/security-deposits" || pathname.startsWith("/security-deposits/")) return "payment.read";
  if (pathname === "/dte" || pathname.startsWith("/dte/")) return "fiscal.read";
  if (pathname === "/settings/dte") return "fiscal.read";
  if (pathname === "/settings/billing") return "billing.read";
  if (pathname === "/settings/public-catalog") return "public_catalog.read";
  if (pathname === "/settings/quote-portal") return "quote.read";
  if (pathname === "/settings/team") return "team.manage";
  if (pathname === "/settings/organization") return "tenant.manage";
  return null;
}

function AccessDenied() {
  return (
    <AppShell>
      <section className="panel access-denied-panel">
        <span className="access-denied-icon">!</span>
        <p className="eyebrow">ACCESS CONTROL</p>
        <h2>No tienes permiso para abrir esta sección</h2>
        <p>Tu rol actual no incluye el permiso requerido. RentStage también aplica esta validación en la API.</p>
        <div className="form-actions">
          <Link className="button button-primary" href="/">Volver al dashboard</Link>
          <Link className="button button-secondary" href="/workspaces">Cambiar workspace</Link>
        </div>
      </section>
    </AppShell>
  );
}

function Frame({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const router = useRouter();
  const { loading, me, can } = useAuth();
  const isPublic = isPublicPath(pathname);
  const isStandalone = standaloneAuthenticated.some((route) => pathname === route || pathname.startsWith(`${route}/`));
  const permission = requiredPermission(pathname);

  useEffect(() => {
    if (loading) return;
    if (!me && !isPublic) {
      const query = searchParams.toString();
      const next = `${pathname}${query ? `?${query}` : ""}`;
      router.replace(`/login?next=${encodeURIComponent(next)}`);
      return;
    }
    if (me && !me.active_workspace && !isPublic && pathname !== "/onboarding") {
      router.replace("/onboarding");
      return;
    }
    if (me?.active_workspace && (pathname === "/login" || pathname === "/signup")) {
      router.replace("/");
    }
  }, [loading, me, isPublic, pathname, router, searchParams]);

  if (loading) {
    return <div className="auth-loading"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><p>Inicializando RentStage…</p></div>;
  }
  if (isPublic) return <>{children}</>;
  if (!me) return <div className="auth-loading"><p>Redirigiendo al inicio de sesión…</p></div>;
  if (!me.active_workspace && pathname !== "/onboarding") return <div className="auth-loading"><p>Preparando onboarding…</p></div>;
  if (isStandalone) return <>{children}</>;
  if (permission && !can(permission)) return <AccessDenied />;
  return <AppShell>{children}</AppShell>;
}

function FrameFallback() {
  return (
    <div className="auth-loading">
      <span className="brand-mark">
        <span className="brand-wave" />
        <span className="brand-wave brand-wave-two" />
        <span className="brand-wave brand-wave-three" />
      </span>
      <p>Inicializando RentStage…</p>
    </div>
  );
}

export function RootFrame({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <Suspense fallback={<FrameFallback />}>
        <Frame>{children}</Frame>
      </Suspense>
    </AuthProvider>
  );
}
