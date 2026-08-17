"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import type { TeamInvitation, Workspace } from "@/lib/types";

export default function InvitationPage() {
  const params = useParams<{ token: string }>();
  const router = useRouter();
  const { me, selectWorkspace, refresh } = useAuth();
  const [invite, setInvite] = useState<TeamInvitation | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [accepting, setAccepting] = useState(false);

  useEffect(() => { api<TeamInvitation>(`/api/v1/invitations/${params.token}`).then(setInvite).catch((reason) => setError(reason instanceof Error ? reason.message : "Invitación no encontrada.")).finally(() => setLoading(false)); }, [params.token]);
  async function accept() { setAccepting(true); setError(""); try { const workspace = await api<Workspace>(`/api/v1/invitations/${params.token}/accept`, { method: "POST" }); await refresh(); await selectWorkspace(workspace.tenant_id); router.replace("/"); } catch (reason) { setError(reason instanceof ApiError ? reason.message : "No fue posible aceptar la invitación."); } finally { setAccepting(false); } }

  return <main className="invite-page"><section className="invite-card panel"><div className="auth-brand dark"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Workspace invitation</small></span></div>{loading ? <div className="skeleton detail-skeleton" /> : error && !invite ? <div className="form-alert">{error}</div> : invite && <><div className="invite-icon">✦</div><p className="eyebrow">INVITACIÓN DE EQUIPO</p><h1>Únete a {invite.tenant_name}</h1><p><strong>{invite.invited_by}</strong> te invitó con el rol <strong>{invite.role}</strong>.</p><div className="invite-meta"><span><small>Correo invitado</small><strong>{invite.email}</strong></span><span><small>Estado</small><strong>{invite.status}</strong></span></div>{error && <div className="form-alert">{error}</div>}{!me ? <div className="invite-actions"><Link className="button button-primary" href={`/login?next=${encodeURIComponent(`/invites/${params.token}`)}`}>Iniciar sesión</Link><Link className="button button-secondary" href={`/signup?next=${encodeURIComponent(`/invites/${params.token}`)}`}>Crear cuenta</Link></div> : <><p className="signed-in-as">Sesión actual: {me.user.email}</p><button className="button button-primary auth-submit" disabled={accepting || invite.status !== "PENDING"} onClick={() => void accept()}>{accepting ? "Aceptando…" : "Aceptar invitación"}</button></>}</>}</section></main>;
}
