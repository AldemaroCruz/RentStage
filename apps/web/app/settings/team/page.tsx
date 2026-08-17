"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { Role, TeamInvitation, TeamMember, TeamResult } from "@/lib/types";

const roleLabels: Record<Role, string> = {
  OWNER: "Owner",
  ADMIN: "Administrador",
  MANAGER: "Manager",
  STAFF: "Staff",
};

const memberStatusLabels: Record<TeamMember["status"], string> = {
  ACTIVE: "Activo",
  SUSPENDED: "Suspendido",
  REMOVED: "Removido",
  INVITED: "Invitado",
};

const invitationStatusLabels: Record<TeamInvitation["status"], string> = {
  PENDING: "Pendiente",
  ACCEPTED: "Aceptada",
  REVOKED: "Revocada",
  EXPIRED: "Vencida",
};

function canManageTarget(actorRole: Role | undefined, targetRole: Role): boolean {
  if (actorRole === "OWNER") return targetRole !== "OWNER";
  if (actorRole === "ADMIN") return targetRole === "MANAGER" || targetRole === "STAFF";
  return false;
}

export default function TeamPage() {
  const { me } = useAuth();
  const actorRole = me?.active_workspace?.role;
  const [result, setResult] = useState<TeamResult>({ members: [], invitations: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<Exclude<Role, "OWNER">>("STAFF");
  const [createdInvite, setCreatedInvite] = useState<TeamInvitation | null>(null);
  const [acting, setActing] = useState("");
  const memberCountLabel = loading
    ? "Miembros del equipo"
    : `${result.members.length} ${result.members.length === 1 ? "miembro" : "miembros"}`;

  const load = useCallback(() => {
    setLoading(true);
    api<TeamResult>("/api/v1/team")
      .then((data) => {
        setResult(data);
        setError("");
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el equipo."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => load(), [load]);

  async function invite(event: FormEvent) {
    event.preventDefault();
    setActing("invite");
    setError("");
    setCreatedInvite(null);
    try {
      const item = await api<TeamInvitation>("/api/v1/team/invitations", {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail, role: inviteRole }),
      });
      setCreatedInvite(item);
      setInviteEmail("");
      load();
    } catch (reason) {
      if (reason instanceof ApiError) {
        setError(reason.fields?.email || reason.fields?.role || reason.message);
      } else {
        setError("No fue posible crear la invitación.");
      }
    } finally {
      setActing("");
    }
  }

  async function updateMember(member: TeamMember, patch: { role?: string; status?: string }) {
    setActing(member.user_id);
    setError("");
    try {
      const updated = await api<TeamMember>(`/api/v1/team/members/${member.user_id}`, {
        method: "PATCH",
        body: JSON.stringify(patch),
      });
      setResult((current) => ({
        ...current,
        members: current.members.map((item) => item.user_id === updated.user_id ? updated : item),
      }));
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible actualizar la membresía.");
    } finally {
      setActing("");
    }
  }

  async function revoke(invitation: TeamInvitation) {
    if (!window.confirm(`¿Revocar la invitación para ${invitation.email}?`)) return;
    setActing(invitation.id);
    setError("");
    try {
      await api<void>(`/api/v1/team/invitations/${invitation.id}`, { method: "DELETE" });
      load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible revocar la invitación.");
    } finally {
      setActing("");
    }
  }

  return (
    <div className="page-stack">
      <div className="page-heading">
        <div>
          <p className="eyebrow">IDENTITY & ACCESS</p>
          <h2>Equipo y permisos</h2>
          <p>Administra membresías activas e invitaciones del workspace actual.</p>
        </div>
      </div>

      {error && <div className="form-alert">{error}</div>}

      {createdInvite?.accept_url && (
        <div className="panel invitation-link-card">
          <div>
            <strong>Invitación creada</strong>
            <p>El envío por correo llegará en una fase posterior. Copia este enlace para la prueba local.</p>
            <code>{createdInvite.accept_url}</code>
          </div>
          <button
            type="button"
            className="button button-secondary"
            onClick={() => void navigator.clipboard.writeText(createdInvite.accept_url || "")}
          >
            Copiar enlace
          </button>
        </div>
      )}

      <section className="panel team-layout">
        <form className="team-invite-form" onSubmit={invite}>
          <div className="team-invite-intro">
            <p className="eyebrow">NUEVO ACCESO</p>
            <h3>Invitar miembro</h3>
            <p>Agrega personas al workspace y define su nivel de acceso.</p>
          </div>
          <label className="field">
            <span>Correo</span>
            <input type="email" value={inviteEmail} onChange={(event) => setInviteEmail(event.target.value)} required />
          </label>
          <label className="field">
            <span>Rol</span>
            <select value={inviteRole} onChange={(event) => setInviteRole(event.target.value as Exclude<Role, "OWNER">)}>
              {actorRole === "OWNER" && <option value="ADMIN">Administrador</option>}
              <option value="MANAGER">Manager</option>
              <option value="STAFF">Staff</option>
            </select>
          </label>
          <button type="submit" className="button button-primary" disabled={acting === "invite"}>
            {acting === "invite" ? "Creando…" : "Crear invitación"}
          </button>
        </form>
      </section>

      <section className="panel team-section">
        <div className="team-section-header">
          <div>
            <p className="eyebrow">MEMBRESÍAS</p>
            <h3>{memberCountLabel}</h3>
            <p>Usuarios con acceso vigente o suspendido dentro de la organización activa.</p>
          </div>
        </div>
        {loading ? (
          <div className="skeleton table-skeleton" />
        ) : (
          <div className="team-table" role="table" aria-label="Miembros del workspace">
            <div className="team-table-head" role="row">
              <span role="columnheader">Usuario</span>
              <span role="columnheader">Rol</span>
              <span role="columnheader">Estado</span>
              <span role="columnheader">Actividad</span>
              <span role="columnheader" className="team-action-heading">Acciones</span>
            </div>
            {result.members.map((member) => {
              const manageable = canManageTarget(actorRole, member.role);
              const self = member.user_id === me?.user.id;
              return (
                <div className="team-row" role="row" key={member.user_id}>
                  <div className="team-person" role="cell">
                    <span aria-hidden="true">{member.display_name.split(/\s+/).slice(0, 2).map((part) => part[0]).join("")}</span>
                    <div>
                      <strong>{member.display_name}</strong>
                      <small>{member.email}</small>
                    </div>
                  </div>
                  <div className="team-role-cell" role="cell">
                    <span className="team-mobile-label">Rol</span>
                    <select
                      value={member.role}
                      disabled={!manageable || acting === member.user_id}
                      onChange={(event) => void updateMember(member, { role: event.target.value })}
                      aria-label={`Rol de ${member.display_name}`}
                    >
                      {member.role === "OWNER" && <option value="OWNER">Owner</option>}
                      {(actorRole === "OWNER" || member.role === "ADMIN") && <option value="ADMIN">Administrador</option>}
                      <option value="MANAGER">Manager</option>
                      <option value="STAFF">Staff</option>
                    </select>
                  </div>
                  <div className="team-status-cell" role="cell">
                    <span className="team-mobile-label">Estado</span>
                    <span className={`membership-status ${member.status.toLowerCase()}`}>
                      {memberStatusLabels[member.status]}
                    </span>
                  </div>
                  <div className="team-activity-cell" role="cell">
                    <span className="team-mobile-label">Actividad</span>
                    <small>{member.last_login_at ? formatDateTime(member.last_login_at) : "Sin inicio de sesión"}</small>
                  </div>
                  <div className="team-action" role="cell">
                    <button
                      type="button"
                      className="button button-ghost"
                      disabled={!manageable || self || acting === member.user_id}
                      onClick={() => void updateMember(member, { status: member.status === "ACTIVE" ? "SUSPENDED" : "ACTIVE" })}
                    >
                      {member.status === "ACTIVE" ? "Suspender" : "Reactivar"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </section>

      <section className="panel team-section">
        <div className="team-section-header">
          <div>
            <p className="eyebrow">INVITACIONES</p>
            <h3>Accesos pendientes e históricos</h3>
            <p>Consulta invitaciones activas, aceptadas, revocadas o vencidas.</p>
          </div>
        </div>
        <div className="invitation-list">
          {result.invitations.length === 0 && <p className="empty-copy">No hay invitaciones todavía.</p>}
          {result.invitations.map((item) => (
            <div className="invitation-row" key={item.id}>
              <div className="invitation-person">
                <strong>{item.email}</strong>
                <small>{roleLabels[item.role]} · vence {formatDateTime(item.expires_at)}</small>
              </div>
              <span className={`membership-status ${item.status.toLowerCase()}`}>{invitationStatusLabels[item.status]}</span>
              {item.status === "PENDING" ? (
                <button type="button" className="button button-ghost" disabled={acting === item.id} onClick={() => void revoke(item)}>Revocar</button>
              ) : <span aria-hidden="true" />}
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
