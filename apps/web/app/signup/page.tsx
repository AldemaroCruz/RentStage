"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { safeInternalPath } from "@/lib/navigation";

export default function SignupPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { signup } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (password.length < 8) { setError("La contraseña debe tener al menos 8 caracteres."); return; }
    if (password !== confirmation) { setError("Las contraseñas no coinciden."); return; }
    setSubmitting(true); setError("");
    try {
      const me = await signup(name, email, password);
      const next = safeInternalPath(searchParams.get("next"));
      router.replace(next || (me.active_workspace ? "/" : "/onboarding"));
    } catch (reason) {
      const code = typeof reason === "object" && reason && "code" in reason ? String((reason as { code?: string }).code) : "";
      setError(code.includes("email-already-in-use") ? "Ya existe una cuenta con ese correo." : reason instanceof Error ? reason.message : "No fue posible crear la cuenta.");
    } finally { setSubmitting(false); }
  }

  return <main className="auth-page"><section className="auth-visual"><div className="auth-brand"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Rental operations</small></span></div><div><p className="eyebrow">CREA TU CUENTA</p><h1>Un solo usuario. Todos tus workspaces.</h1><p>Empieza con alquiler de audio y conserva la base para operar cualquier recurso reservable.</p></div></section><section className="auth-form-column"><form className="auth-card panel" onSubmit={submit}><div><p className="eyebrow">REGISTRO</p><h2>Crear cuenta</h2><p>Después configuraremos tu primera empresa.</p></div>{error && <div className="form-alert">{error}</div>}<label className="field"><span>Nombre</span><input value={name} onChange={(event) => setName(event.target.value)} required maxLength={160} /></label><label className="field"><span>Correo</span><input type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /></label><div className="form-grid two"><label className="field"><span>Contraseña</span><input type="password" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required /></label><label className="field"><span>Confirmar</span><input type="password" autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} required /></label></div><button className="button button-primary auth-submit" disabled={submitting}>{submitting ? "Creando…" : "Crear cuenta"}</button><p className="auth-switch">¿Ya tienes cuenta? <Link href="/login">Iniciar sesión</Link></p></form></section></main>;
}
