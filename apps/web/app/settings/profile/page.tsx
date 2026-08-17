"use client";

import { useAuth } from "@/components/AuthProvider";
import { formatDateTime } from "@/lib/format";

export default function ProfilePage() {
  const { me } = useAuth();
  const user = me?.user;
  if (!user) return null;
  return <div className="page-stack"><div className="page-heading"><div><p className="eyebrow">IDENTIDAD</p><h2>Mi perfil</h2><p>La autenticación se gestiona en Firebase; RentStage conserva membresías y permisos.</p></div></div><section className="panel profile-detail-card"><span className="profile-large-avatar">{user.display_name.split(/\s+/).slice(0,2).map((part) => part[0]).join("")}</span><div><h3>{user.display_name}</h3><p>{user.email}</p><div className="profile-detail-grid"><span><small>Correo verificado</small><strong>{user.email_verified ? "Sí" : "No"}</strong></span><span><small>Estado</small><strong>{user.status}</strong></span><span><small>Último acceso</small><strong>{user.last_login_at ? formatDateTime(user.last_login_at) : "Sin registro"}</strong></span><span><small>Identity UID</small><code>{user.identity_uid}</code></span></div></div></section><div className="panel info-banner"><strong>Edición de perfil</strong><p>La actualización de nombre, correo, contraseña y MFA se integrará con Identity Platform al preparar el entorno de GCP.</p></div></div>;
}
