"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { safeInternalPath } from "@/lib/navigation";

function messageFor(reason: unknown): string {
  const code = typeof reason === "object" && reason && "code" in reason ? String((reason as { code?: string }).code) : "";
  if (code.includes("invalid-credential") || code.includes("wrong-password") || code.includes("user-not-found")) return "Correo o contraseña incorrectos.";
  if (code.includes("too-many-requests")) return "Demasiados intentos. Espera un momento e inténtalo nuevamente.";
  return reason instanceof Error ? reason.message : "No fue posible iniciar sesión.";
}

export default function LoginPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();
  const [email, setEmail] = useState("owner@rentstage.local");
  const [password, setPassword] = useState("RentStage123!");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const me = await login(email, password);
      const next = safeInternalPath(searchParams.get("next"));
      router.replace(next || (me.active_workspace ? "/" : "/onboarding"));
    } catch (reason) {
      setError(messageFor(reason));
    } finally {
      setSubmitting(false);
    }
  }

  return <main className="auth-page"><section className="auth-visual"><div className="auth-brand"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Rental operations</small></span></div><div><p className="eyebrow">IDENTITY & SAAS FOUNDATION</p><h1>Tu operación, protegida y separada por empresa.</h1><p>Sesiones seguras, roles, workspaces y trazabilidad con usuarios reales.</p></div><div className="auth-feature-grid"><span>Multi-tenant</span><span>RBAC</span><span>HttpOnly sessions</span><span>Audit actors</span></div></section><section className="auth-form-column"><form className="auth-card panel" onSubmit={submit}><div><p className="eyebrow">BIENVENIDO</p><h2>Iniciar sesión</h2><p>Accede al workspace de tu empresa.</p></div>{error && <div className="form-alert">{error}</div>}<label className="field"><span>Correo</span><input type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /></label><label className="field"><span>Contraseña</span><input type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required /></label><button className="button button-primary auth-submit" disabled={submitting}>{submitting ? "Ingresando…" : "Iniciar sesión"}</button><p className="auth-switch">¿Primera vez? <Link href="/signup">Crear cuenta</Link></p><div className="local-credentials"><strong>Cuenta local de desarrollo</strong><code>owner@rentstage.local</code><code>RentStage123!</code></div></form></section></main>;
}
