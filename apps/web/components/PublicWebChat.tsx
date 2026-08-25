"use client";

import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError } from "@/lib/api";
import type {
  PublicCatalogViewSettings,
  PublicTenant,
  PublicWebChatCreateResult,
  PublicWebChatSession,
} from "@/lib/types";

type StoredChat = {
  session_id: string;
  token: string;
};

function storageKey(tenantSlug: string): string {
  return `rentstage-public-chat:${tenantSlug}`;
}

function readStoredChat(tenantSlug: string): StoredChat | null {
  try {
    const value = window.sessionStorage.getItem(storageKey(tenantSlug));
    if (!value) return null;
    const parsed = JSON.parse(value) as Partial<StoredChat>;
    if (!parsed.session_id || !parsed.token) return null;
    return { session_id: parsed.session_id, token: parsed.token };
  } catch {
    return null;
  }
}

function forgetStoredChat(tenantSlug: string): void {
  window.sessionStorage.removeItem(storageKey(tenantSlug));
}

function operationError(reason: unknown, fallback: string): string {
  if (reason instanceof ApiError) {
    const fieldMessage = reason.fields && Object.values(reason.fields)[0];
    if (fieldMessage) return fieldMessage;
    if (reason.status === 410) return "Esta conversación venció. Inicia una nueva para continuar.";
    if (reason.status === 429) return "Has enviado varios mensajes. Intenta nuevamente más tarde.";
  }
  return reason instanceof Error ? reason.message : fallback;
}

function messageTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("es-SV", { hour: "2-digit", minute: "2-digit" });
}

export function PublicWebChat({
  tenant,
  settings,
}: {
  tenant: PublicTenant;
  settings: PublicCatalogViewSettings;
}) {
  const [open, setOpen] = useState(false);
  const [restoring, setRestoring] = useState(true);
  const [working, setWorking] = useState(false);
  const [session, setSession] = useState<PublicWebChatSession | null>(null);
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [body, setBody] = useState("");
  const [draft, setDraft] = useState({
    contact_name: "",
    contact_email: "",
    message: "",
    consent_accepted: false,
    website: "",
  });
  const messageListRef = useRef<HTMLDivElement>(null);

  const getSession = useCallback(async (sessionID: string, sessionToken: string) => {
    return api<PublicWebChatSession>(
      `/api/v1/public/chat/${encodeURIComponent(tenant.slug)}/sessions/${encodeURIComponent(sessionID)}`,
      { headers: { "X-RentStage-Chat-Token": sessionToken } },
    );
  }, [tenant.slug]);

  useEffect(() => {
    let active = true;
    const stored = readStoredChat(tenant.slug);
    if (!stored) {
      setRestoring(false);
      return () => { active = false; };
    }

    setToken(stored.token);
    getSession(stored.session_id, stored.token)
      .then((item) => {
        if (active) setSession(item);
      })
      .catch(() => {
        if (!active) return;
        forgetStoredChat(tenant.slug);
        setToken("");
      })
      .finally(() => {
        if (active) setRestoring(false);
      });

    return () => { active = false; };
  }, [getSession, tenant.slug]);

  useEffect(() => {
    if (!open || !session || !token || session.status !== "ACTIVE") return;
    const sessionID = session.id;
    const interval = window.setInterval(() => {
      getSession(sessionID, token)
        .then(setSession)
        .catch((reason) => {
          if (reason instanceof ApiError && [404, 410].includes(reason.status)) {
            forgetStoredChat(tenant.slug);
            setSession(null);
            setToken("");
            setError("La conversación ya no está disponible. Puedes iniciar una nueva.");
          }
        });
    }, 4_000);
    return () => window.clearInterval(interval);
  }, [getSession, open, session, tenant.slug, token]);

  useEffect(() => {
    if (!open) return;
    messageListRef.current?.scrollTo({
      top: messageListRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [open, session?.messages.length]);

  async function createSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setWorking(true);
    setError("");
    try {
      const result = await api<PublicWebChatCreateResult>(
        `/api/v1/public/chat/${encodeURIComponent(tenant.slug)}/sessions`,
        {
          method: "POST",
          body: JSON.stringify({
            contact_name: draft.contact_name,
            contact_email: draft.contact_email.trim() || undefined,
            message: draft.message,
            client_message_id: crypto.randomUUID(),
            consent_accepted: draft.consent_accepted,
            website: draft.website,
          }),
        },
      );
      window.sessionStorage.setItem(storageKey(tenant.slug), JSON.stringify({
        session_id: result.session.id,
        token: result.token,
      } satisfies StoredChat));
      setSession(result.session);
      setToken(result.token);
      setDraft((value) => ({ ...value, message: "" }));
    } catch (reason) {
      setError(operationError(reason, "No fue posible iniciar la conversación."));
    } finally {
      setWorking(false);
    }
  }

  async function sendMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session || !token || !body.trim()) return;
    setWorking(true);
    setError("");
    try {
      const result = await api<PublicWebChatSession>(
        `/api/v1/public/chat/${encodeURIComponent(tenant.slug)}/sessions/${encodeURIComponent(session.id)}/messages`,
        {
          method: "POST",
          headers: { "X-RentStage-Chat-Token": token },
          body: JSON.stringify({ body, client_message_id: crypto.randomUUID() }),
        },
      );
      setSession(result);
      setBody("");
    } catch (reason) {
      setError(operationError(reason, "No fue posible enviar el mensaje."));
    } finally {
      setWorking(false);
    }
  }

  function startNewConversation() {
    forgetStoredChat(tenant.slug);
    setSession(null);
    setToken("");
    setBody("");
    setError("");
  }

  return (
    <div className={`public-web-chat ${open ? "open" : ""}`}>
      {open && (
        <section className="public-web-chat-panel" role="dialog" aria-label={`Chat con ${tenant.name}`}>
          <header className="public-web-chat-header">
            <span className="public-web-chat-avatar">{tenant.name.slice(0, 2).toUpperCase()}</span>
            <div><strong>{tenant.name}</strong><small><i /> Atención con revisión humana</small></div>
            <button type="button" onClick={() => setOpen(false)} aria-label="Cerrar chat">×</button>
          </header>

          {restoring ? (
            <div className="public-web-chat-state"><span className="public-loader" /><p>Recuperando conversación…</p></div>
          ) : session ? (
            <>
              <div className="public-web-chat-messages" ref={messageListRef} aria-live="polite">
                <div className="public-web-chat-intro">
                  Esta conversación permanece disponible durante siete días en este navegador.
                </div>
                {session.messages.map((message) => {
                  const fromVisitor = message.direction === "INBOUND";
                  return (
                    <article className={fromVisitor ? "visitor" : "team"} key={message.id}>
                      <small>{fromVisitor ? "Tú" : tenant.name}</small>
                      <p>{message.body}</p>
                      <time dateTime={message.created_at}>{messageTime(message.created_at)}</time>
                    </article>
                  );
                })}
              </div>
              {session.status === "ACTIVE" ? (
                <form className="public-web-chat-composer" onSubmit={sendMessage}>
                  {error && <p className="public-web-chat-error">{error}</p>}
                  <label>
                    <span className="sr-only">Escribe tu mensaje</span>
                    <textarea
                      value={body}
                      onChange={(event) => setBody(event.target.value)}
                      placeholder="Escribe tu mensaje…"
                      maxLength={2000}
                      required
                    />
                  </label>
                  <button type="submit" disabled={working || !body.trim()}>{working ? "…" : "Enviar"}</button>
                </form>
              ) : (
                <div className="public-web-chat-ended">
                  <p>Esta conversación está cerrada.</p>
                  <button type="button" onClick={startNewConversation}>Iniciar otra conversación</button>
                </div>
              )}
            </>
          ) : (
            <form className="public-web-chat-start" onSubmit={createSession}>
              <div><strong>¿Cómo podemos ayudarte?</strong><p>Escríbenos y el equipo responderá desde RentStage.</p></div>
              {error && <p className="public-web-chat-error">{error}</p>}
              <label><span>Nombre</span><input value={draft.contact_name} onChange={(event) => setDraft({ ...draft, contact_name: event.target.value })} minLength={2} maxLength={120} required /></label>
              <label><span>Correo <em>opcional</em></span><input type="email" value={draft.contact_email} onChange={(event) => setDraft({ ...draft, contact_email: event.target.value })} maxLength={320} /></label>
              <label><span>Mensaje</span><textarea value={draft.message} onChange={(event) => setDraft({ ...draft, message: event.target.value })} maxLength={2000} required /></label>
              <label className="public-web-chat-honeypot" aria-hidden="true"><span>Sitio web</span><input value={draft.website} onChange={(event) => setDraft({ ...draft, website: event.target.value })} tabIndex={-1} autoComplete="off" /></label>
              <label className="public-web-chat-consent">
                <input type="checkbox" checked={draft.consent_accepted} onChange={(event) => setDraft({ ...draft, consent_accepted: event.target.checked })} required />
                <span>{settings.terms_text || `Acepto que ${tenant.name} me contacte para responder esta consulta.`}</span>
              </label>
              <button className="public-web-chat-start-button" type="submit" disabled={working}>{working ? "Iniciando…" : "Iniciar conversación"}</button>
              <small className="public-web-chat-privacy">No compartas contraseñas ni información de pago.</small>
            </form>
          )}
        </section>
      )}

      <button
        className="public-web-chat-launcher"
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        aria-label={open ? "Cerrar chat" : `Chatear con ${tenant.name}`}
      >
        <span aria-hidden="true">{open ? "×" : "✦"}</span>
        {!open && <strong>¿Necesitas ayuda?</strong>}
      </button>
    </div>
  );
}
