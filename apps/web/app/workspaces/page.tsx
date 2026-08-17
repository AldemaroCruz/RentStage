"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";

export default function WorkspacesPage() {
  const router = useRouter();
  const { me, selectWorkspace, logout } = useAuth();
  async function enter(tenantID: string) { await selectWorkspace(tenantID); router.push("/"); }
  return <main className="standalone-page"><header className="standalone-header"><div className="auth-brand dark"><span className="brand-mark"><span className="brand-wave" /><span className="brand-wave brand-wave-two" /><span className="brand-wave brand-wave-three" /></span><span><strong>RentStage</strong><small>Workspaces</small></span></div><button className="button button-secondary" onClick={() => void logout()}>Cerrar sesión</button></header><section className="workspace-page"><div className="page-heading"><div><p className="eyebrow">ORGANIZACIONES</p><h1>Selecciona un workspace</h1><p>{me?.user.email}</p></div><Link className="button button-primary" href="/onboarding">+ Crear workspace</Link></div><div className="workspace-grid">{me?.workspaces.map((workspace) => <button key={workspace.tenant_id} className={`workspace-card panel ${workspace.tenant_id === me.active_workspace?.tenant_id ? "active" : ""}`} onClick={() => void enter(workspace.tenant_id)}><span className="tenant-avatar">{workspace.name.split(/\s+/).slice(0,2).map((word) => word[0]).join("")}</span><div><strong>{workspace.name}</strong><p>{workspace.slug}</p><small>{workspace.role}</small></div><span>→</span></button>)}</div>{me?.workspaces.length === 0 && <div className="panel empty-state"><h2>Aún no perteneces a una empresa</h2><p>Crea tu primer workspace o acepta una invitación.</p><Link className="button button-primary" href="/onboarding">Crear workspace</Link></div>}</section></main>;
}
