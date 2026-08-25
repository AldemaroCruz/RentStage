"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, formatDateTime, formatQuoteNumber, formatReservationNumber } from "@/lib/format";
import type {
  AssistantConversationDetail,
  AssistantConversationSummary,
  AssistantMessage,
  Customer,
  MetaReadiness,
} from "@/lib/types";

function localInput(date: Date): string {
  const adjusted = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return adjusted.toISOString().slice(0, 16);
}

function statusLabel(value: AssistantConversationSummary["status"]): string {
  if (value === "HUMAN_REVIEW") return "Revisión humana";
  if (value === "QUOTE_DRAFTED") return "Cotización creada";
  if (value === "CLOSED") return "Cerrada";
  return "Abierta";
}

function channelLabel(
  value: AssistantConversationSummary["channel"],
): string {
  if (value === "WEB_CHAT") return "Chat web";
  if (value === "WHATSAPP") return "Meta local";
  if (value === "INSTAGRAM") return "Instagram";
  if (value === "MESSENGER") return "Messenger";
  return "Canal demo";
}

function messageLabel(
  message: AssistantMessage,
  channel: AssistantConversationSummary["channel"],
): string {
  if (message.metadata.superseded === true) {
    return "ASISTENTE · BORRADOR REEMPLAZADO";
  }

  if (message.direction === "INBOUND") {
    if (channel === "WEB_CHAT") return "CLIENTE · CHAT WEB";
    if (channel === "WHATSAPP") return "CLIENTE · WEBHOOK DE WHATSAPP";
    return "CLIENTE · MENSAJE SIMULADO";
  }

  if (message.status === "DRAFT") {
    return "ASISTENTE · BORRADOR NO ENVIADO";
  }
  if (message.status === "APPROVED") {
    return "EQUIPO · APROBADO Y PENDIENTE";
  }
  if (message.status === "SENT") {
    if (channel === "WEB_CHAT") return "EQUIPO · PUBLICADO EN CHAT WEB";
    if (channel === "WHATSAPP") return "EQUIPO · ACEPTADO POR META LOCAL";
    return "EQUIPO · ENTREGADO EN EL SIMULADOR";
  }
  if (message.status === "DELIVERED") return "EQUIPO · ENTREGADO";
  if (message.status === "READ") return "EQUIPO · LEÍDO";
  if (message.status === "FAILED") return "EQUIPO · ENTREGA FALLIDA";
  return "EQUIPO";
}

function portalSessionKey(conversationID: string): string {
  return `rentstage-assistant-portal:${conversationID}`;
}

function portalStatusLabel(value: string | undefined): string {
  return {
    ACTIVE: "Esperando respuesta",
    ACCEPTED: "Aceptada por el cliente",
    REJECTED: "Rechazada por el cliente",
    REVOKED: "Enlace revocado",
    EXPIRED: "Enlace vencido",
  }[value || ""] || "No compartida";
}

const initialStart = new Date();
initialStart.setDate(initialStart.getDate() + 30);
initialStart.setHours(15, 0, 0, 0);
const initialEnd = new Date(initialStart);
initialEnd.setHours(23, 0, 0, 0);

const initialSimulation = {
  contact_name: "Ana Martínez",
  contact_phone: "+50370123456",
  message: "Hola, necesito sonido para una boda de 100 personas en San Salvador. ¿Qué paquete me recomienda y cuánto cuesta?",
  event_type: "Boda",
  event_location: "San Salvador",
  guest_count: "100",
  start_at: localInput(initialStart),
  end_at: localInput(initialEnd),
};

const emptyCustomerDraft = {
  first_name: "",
  last_name: "",
  phone: "",
  email: "",
  company_name: "",
};

export default function AssistantPage() {
  const { can } = useAuth();
  const canManage = can("assistant.manage");
  const canManageQuote = can("quote.manage");
  const canManageCustomers = can("customer.manage");
  const [items, setItems] = useState<AssistantConversationSummary[]>([]);
  const [detail, setDetail] = useState<AssistantConversationDetail | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [readiness, setReadiness] = useState<MetaReadiness | null>(null);
  const [customerID, setCustomerID] = useState("");
  const [responseBody, setResponseBody] = useState("");
  const [sendBody, setSendBody] = useState("");
  const [incomingBody, setIncomingBody] = useState("");
  const [customerDraft, setCustomerDraft] = useState(emptyCustomerDraft);
  const [customerCreatorOpen, setCustomerCreatorOpen] = useState(false);
  const [incomingOpen, setIncomingOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [composerOpen, setComposerOpen] = useState(false);
  const [simulation, setSimulation] = useState(initialSimulation);
  const [portalURL, setPortalURL] = useState("");
  const [portalCopied, setPortalCopied] = useState(false);
  const chatScrollRef = useRef<HTMLDivElement>(null);

  const pendingMessage = useMemo(() => {
    if (!detail) return undefined;
    return [...detail.messages].reverse().find((message) =>
      message.direction === "OUTBOUND" && ["DRAFT", "APPROVED"].includes(message.status));
  }, [detail]);

  async function loadList(preferredID?: string) {
    const response = await api<{ items: AssistantConversationSummary[] }>("/api/v1/assistant/conversations");
    setItems(response.items);
    const selectedID = preferredID || detail?.id || response.items[0]?.id;
    if (selectedID) {
      const loaded = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${selectedID}`);
      setDetail(loaded);
    } else {
      setDetail(null);
    }
  }

  async function loadCustomers(preferredID?: string) {
    const response = await api<{ items: Customer[] }>("/api/v1/customers");
    setCustomers(response.items);
    if (preferredID) setCustomerID(preferredID);
  }

  useEffect(() => {
    Promise.all([
      api<{ items: AssistantConversationSummary[] }>("/api/v1/assistant/conversations"),
      api<{ items: Customer[] }>("/api/v1/customers"),
      api<MetaReadiness>("/api/v1/integrations/meta/readiness"),
    ])
      .then(async ([conversationResponse, customerResponse, readinessResponse]) => {
        setItems(conversationResponse.items);
        setCustomers(customerResponse.items);
        setReadiness(readinessResponse);
        if (conversationResponse.items[0]) {
          setDetail(await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${conversationResponse.items[0].id}`));
        }
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el asistente."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    setResponseBody(detail?.proposal?.response_draft || "");
    setSendBody(pendingMessage?.body || "");
    setCustomerID(detail?.customer_id || customers[0]?.id || "");
    setCustomerDraft({
      ...emptyCustomerDraft,
      first_name: detail?.contact_name || "",
      phone: detail?.contact_phone || "",
      email: detail?.contact_email || "",
    });
    setCustomerCreatorOpen(false);
  }, [detail?.id, detail?.updated_at, pendingMessage?.id, customers]);

  useEffect(() => {
    if (!detail) {
      setPortalURL("");
      return;
    }
    setPortalURL(window.sessionStorage.getItem(portalSessionKey(detail.id)) || "");
    setPortalCopied(false);
  }, [detail?.id]);

  useEffect(() => {
    chatScrollRef.current?.scrollTo({ top: chatScrollRef.current.scrollHeight, behavior: "smooth" });
  }, [detail?.messages.length]);

  useEffect(() => {
    if (!detail?.id || detail.proposal?.portal_status !== "ACTIVE") return;
    const conversationID = detail.id;
    const timer = window.setInterval(() => {
      api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${conversationID}`)
        .then((updated) => setDetail((current) => current?.id === conversationID ? updated : current))
        .catch(() => undefined);
    }, 12_000);
    return () => window.clearInterval(timer);
  }, [detail?.id, detail?.proposal?.portal_status]);

  async function selectConversation(id: string) {
    setWorking(true);
    setError("");
    setNotice("");
    try {
      setDetail(await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${id}`));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible abrir la conversación.");
    } finally {
      setWorking(false);
    }
  }

  async function simulate(event: FormEvent) {
    event.preventDefault();
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const created = await api<AssistantConversationDetail>("/api/v1/assistant/conversations/simulate", {
        method: "POST",
        body: JSON.stringify({
          ...simulation,
          guest_count: Number(simulation.guest_count),
          start_at: new Date(simulation.start_at).toISOString(),
          end_at: new Date(simulation.end_at).toISOString(),
        }),
      });
      setDetail(created);
      setComposerOpen(false);
      setNotice("La consulta demo fue analizada con paquetes y disponibilidad reales.");
      await loadList(created.id);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "No fue posible simular la conversación.");
    } finally {
      setWorking(false);
    }
  }

  async function linkCustomer(selectedCustomerID = customerID) {
    if (!detail || !selectedCustomerID) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const linked = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/customer`, {
        method: "POST",
        body: JSON.stringify({ customer_id: selectedCustomerID }),
      });
      setCustomerID(selectedCustomerID);
      setDetail(linked);
      setNotice("El cliente quedó vinculado a la conversación dentro de este workspace.");
      await loadList(linked.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible vincular el cliente.");
    } finally {
      setWorking(false);
    }
  }

  async function createCustomer(event: FormEvent) {
    event.preventDefault();
    if (!detail) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const created = await api<Customer>("/api/v1/customers", {
        method: "POST",
        body: JSON.stringify({
          ...customerDraft,
          phone: customerDraft.phone || null,
          email: customerDraft.email || null,
          company_name: customerDraft.company_name || null,
          preferred_language: "es",
          source:
            detail.channel === "WEB_CHAT"
              ? "WEB"
              : detail.channel === "DEMO"
                ? "MANUAL"
                : "WHATSAPP",
          notes: `Creado desde la conversación ${detail.channel.toLowerCase()} ${detail.id}.`,
        }),
      });
      await loadCustomers(created.id);
      const linked = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/customer`, {
        method: "POST",
        body: JSON.stringify({ customer_id: created.id }),
      });
      setCustomerID(created.id);
      setDetail(linked);
      await loadList(linked.id);
      setCustomerCreatorOpen(false);
      setNotice(`${created.display_name} fue creado y vinculado desde el chat.`);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible crear el cliente.");
    } finally {
      setWorking(false);
    }
  }

  async function approve() {
    if (!detail) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const approved = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/approve`, {
        method: "POST",
        body: JSON.stringify({ customer_id: customerID, response_body: responseBody }),
      });
      setDetail(approved);
      setNotice("Aprobación registrada. Se creó una cotización DRAFT; ahora puedes entregar la respuesta dentro del simulador.");
      await loadList(approved.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible crear el borrador.");
    } finally {
      setWorking(false);
    }
  }

  async function sendMessage() {
    if (!detail) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const sent = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/messages/send`, {
        method: "POST",
        body: JSON.stringify({ message_id: pendingMessage?.id || "", body: sendBody }),
      });
      setDetail(sent);
      setSendBody("");
      setNotice(
        detail.channel === "WHATSAPP"
          ? "Meta local aceptó la respuesta. El adaptador de desarrollo no contacta ningún teléfono real."
          : detail.channel === "WEB_CHAT"
            ? "Respuesta publicada en la sesión segura del chat web."
            : "Respuesta entregada únicamente dentro del simulador. No se contactó ningún teléfono real.",
      );
      await loadList(sent.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible enviar la respuesta.");
    } finally {
      setWorking(false);
    }
  }

  async function receiveDemo(event: FormEvent) {
    event.preventDefault();
    if (!detail) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const received = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/messages/receive-demo`, {
        method: "POST",
        body: JSON.stringify({ body: incomingBody }),
      });
      setDetail(received);
      setIncomingBody("");
      setIncomingOpen(false);
      setNotice("El cliente respondió en el simulador. El asistente preparó otro borrador pendiente de revisión humana.");
      await loadList(received.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible simular la respuesta del cliente.");
    } finally {
      setWorking(false);
    }
  }

  async function shareQuoteDemo() {
    if (!detail?.proposal?.quote_id) return;
    const rotating = detail.proposal.quote_status === "SENT";
    if (rotating && !window.confirm("Se generará un enlace nuevo y el anterior dejará de funcionar. ¿Continuar?")) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const shared = await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${detail.id}/quote/share-demo`, {
        method: "POST",
        body: JSON.stringify({ body: "" }),
      });
      const delivery = shared.portal_delivery;
      if (!delivery?.public_url) throw new Error("El portal no devolvió su enlace de una sola lectura.");
      window.sessionStorage.setItem(portalSessionKey(shared.id), delivery.public_url);
      setPortalURL(delivery.public_url);
      setPortalCopied(false);
      setDetail(shared);
      setNotice("Cotización enviada al simulador con un enlace seguro disponible solamente en esta sesión del navegador.");
      await loadList(shared.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible compartir la cotización.");
    } finally {
      setWorking(false);
    }
  }

  async function copyPortalURL() {
    if (!portalURL) return;
    try {
      await navigator.clipboard.writeText(portalURL);
      setPortalCopied(true);
    } catch {
      setError("No fue posible copiar el enlace. Ábrelo y cópialo manualmente desde el navegador.");
    }
  }

  async function refreshConversation() {
    if (!detail) return;
    await selectConversation(detail.id);
    setNotice("Estado del portal actualizado.");
  }

  function resetDemo() {
    setSimulation({ ...initialSimulation });
    setDetail(null);
    setPortalURL("");
    setIncomingOpen(false);
    setCustomerCreatorOpen(false);
    setComposerOpen(true);
    setError("");
    setNotice("Demostración preparada desde cero. Completa o conserva los datos y simula una nueva consulta.");
  }

  const metrics = useMemo(() => ({
    review: items.filter((item) => item.status === "HUMAN_REVIEW").length,
    drafted: items.filter((item) => Boolean(item.quote_id)).length,
    demo: items.filter((item) => item.channel === "DEMO").length,
    whatsapp: items.filter((item) => item.channel === "WHATSAPP").length,
    webChat: items.filter((item) => item.channel === "WEB_CHAT").length,
  }), [items]);

  return (
    <div className="page-stack assistant-page">
      <section className="assistant-hero">
        <div>
          <p className="eyebrow">ASISTENTE OMNICANAL · V0.19.0</p>
          <h2>Gestiona cada conversación con una persona al mando.</h2>
          <p>
            Centraliza chat web, Meta local y demostraciones, conserva la revisión
            humana y prepara el camino para nuevos canales.
          </p>
        </div>
        <div className="assistant-hero-actions">
          <span className="assistant-provider-pill">
            <i />
            {detail ? channelLabel(detail.channel).toUpperCase() : "INBOX OMNICANAL"}
          </span>
          {canManage && <button className="button button-primary" onClick={() => setComposerOpen((value) => !value)}>{composerOpen ? "Cerrar simulador" : "Simular consulta"}</button>}
          {canManage && <button className="button button-secondary assistant-reset-button" onClick={resetDemo}>Reiniciar demo</button>}
        </div>
      </section>

      <section className="assistant-boundary-strip">
        <strong>Control humano obligatorio</strong>
        <span>Cada borrador requiere que alguien pulse enviar.</span>
        <span>CHAT WEB publica únicamente en la sesión segura del visitante.</span>
        <span>Nunca reserva inventario automáticamente.</span>
      </section>

      <section className="assistant-boundary-strip">
        <strong>Conector Meta: {readiness?.mode || "cargando"}</strong>
        <span>Webhook firmado: {readiness?.signature_validation_configured ? "listo" : "pendiente"}.</span>
        <span>Salida local: {readiness?.local_delivery_available ? "habilitada" : "bloqueada"}.</span>
        <span>Salida cloud: bloqueada en v0.19.0.</span>
        <Link href="/privacy">Privacidad</Link><Link href="/data-deletion">Eliminar datos</Link><Link href="/support">Soporte</Link>
      </section>

      {composerOpen && canManage && (
        <form className="panel assistant-simulator" onSubmit={simulate}>
          <div className="panel-header"><div><p className="eyebrow">NUEVA CONSULTA DEMO</p><h2>Simular mensaje entrante</h2><p>Los campos estructurados representan lo que un proveedor futuro extraería del mensaje.</p></div><span className="assistant-demo-badge">Sin Meta</span></div>
          <div className="assistant-simulator-grid">
            <label className="field"><span>Nombre</span><input value={simulation.contact_name} onChange={(event) => setSimulation({ ...simulation, contact_name: event.target.value })} required /></label>
            <label className="field"><span>Teléfono</span><input value={simulation.contact_phone} onChange={(event) => setSimulation({ ...simulation, contact_phone: event.target.value })} required /></label>
            <label className="field assistant-message-field"><span>Mensaje del cliente</span><textarea value={simulation.message} onChange={(event) => setSimulation({ ...simulation, message: event.target.value })} required /></label>
            <label className="field"><span>Tipo de evento</span><input value={simulation.event_type} onChange={(event) => setSimulation({ ...simulation, event_type: event.target.value })} required /></label>
            <label className="field"><span>Invitados</span><input type="number" min="1" value={simulation.guest_count} onChange={(event) => setSimulation({ ...simulation, guest_count: event.target.value })} required /></label>
            <label className="field"><span>Ubicación</span><input value={simulation.event_location} onChange={(event) => setSimulation({ ...simulation, event_location: event.target.value })} required /></label>
            <label className="field"><span>Inicio</span><input type="datetime-local" value={simulation.start_at} onChange={(event) => setSimulation({ ...simulation, start_at: event.target.value })} required /></label>
            <label className="field"><span>Fin</span><input type="datetime-local" value={simulation.end_at} onChange={(event) => setSimulation({ ...simulation, end_at: event.target.value })} required /></label>
          </div>
          <div className="form-actions"><button className="button button-primary" disabled={working}>{working ? "Analizando…" : "Analizar con datos reales"}</button></div>
        </form>
      )}

      {(error || notice) && <div className={error ? "inline-error" : "assistant-notice"}>{error || notice}</div>}

      <section className="assistant-metrics">
        <article><span>Conversaciones</span><strong>{items.length}</strong><small>{metrics.webChat} web · {metrics.whatsapp} Meta · {metrics.demo} demo</small></article>
        <article><span>Revisión humana</span><strong>{metrics.review}</strong><small>respuestas esperando aprobación</small></article>
        <article><span>Cotizaciones creadas</span><strong>{metrics.drafted}</strong><small>borrador, portal y decisión trazables</small></article>
      </section>

      <section className="assistant-workspace panel">
        <aside className="assistant-inbox">
          <div className="assistant-column-heading"><div><p className="eyebrow">INBOX</p><h3>Conversaciones</h3></div><span>{items.length}</span></div>
          {loading ? <div className="table-skeleton">Cargando conversaciones…</div> : items.length === 0 ? <div className="assistant-empty">Usa “Simular consulta” para iniciar.</div> : items.map((item) => (
            <button key={item.id} className={`assistant-conversation-row ${detail?.id === item.id ? "active" : ""}`} onClick={() => void selectConversation(item.id)}>
              <span className="assistant-contact-avatar">{item.contact_name.slice(0, 2).toUpperCase()}</span>
              <span><strong>{item.contact_name}</strong><small>{item.last_message}</small><em>{channelLabel(item.channel)} · {statusLabel(item.status)}</em></span>
              <time>{new Date(item.last_message_at).toLocaleDateString("es-SV", { day: "2-digit", month: "short" })}</time>
            </button>
          ))}
        </aside>

        <div className="assistant-chat">
          {!detail ? <div className="assistant-empty large">Selecciona o simula una conversación.</div> : <>
            <header className="assistant-chat-header">
              <div className="assistant-contact-avatar">{detail.contact_name.slice(0, 2).toUpperCase()}</div>
              <div><strong>{detail.contact_name}</strong><span>{detail.channel === "WEB_CHAT" ? detail.contact_email || "visitante web" : detail.contact_phone || "sin teléfono"} · {detail.customer_name || "contacto sin vincular"}</span></div>
              <em className={`assistant-status ${detail.status.toLowerCase()}`}>{statusLabel(detail.status)}</em>
            </header>
            <div className="assistant-chat-scroll" ref={chatScrollRef}>
              <div className="assistant-chat-day">HOY · {channelLabel(detail.channel).toUpperCase()}</div>
              {detail.messages.map((message) => (
                <article key={message.id} className={`assistant-bubble ${message.direction === "OUTBOUND" ? "outbound" : "inbound"} ${message.status === "DRAFT" ? "draft" : ""}`}>
                  <small>{messageLabel(message, detail.channel)}</small>
                  <p>{message.body}</p>
                  {message.metadata.message_kind === "QUOTE_PORTAL" && <div className="assistant-message-portal">
                    <strong>{portalStatusLabel(detail.proposal?.portal_status)}</strong>
                    {portalURL ? <a href={portalURL} target="_blank" rel="noreferrer">Abrir portal como cliente →</a> : <span>El enlace secreto solo se conserva durante esta sesión.</span>}
                  </div>}
                  <time>{formatDateTime(message.created_at)}</time>
                </article>
              ))}
            </div>
            {canManage && <div className="assistant-chat-composer">
              <label><span>{pendingMessage ? "Revisar borrador antes de enviar" : "Responder como miembro del equipo"}</span><textarea value={sendBody} onChange={(event) => setSendBody(event.target.value)} placeholder="Escribe una respuesta para el cliente…" /></label>
              <div className="assistant-composer-actions">
                <small>{detail.channel === "WHATSAPP" ? "La entrega usa el Graph API local y no sale a Internet." : detail.channel === "WEB_CHAT" ? "La respuesta será visible en la sesión segura del visitante." : "La entrega ocurre solamente dentro de este chat demo."}</small>
                <button className="button button-primary" onClick={() => void sendMessage()} disabled={working || !sendBody.trim()}>{working ? "Procesando…" : detail.channel === "WHATSAPP" ? "Enviar por Meta local" : detail.channel === "WEB_CHAT" ? "Publicar en chat web" : "Enviar respuesta demo"}</button>
              </div>
              {detail.channel === "DEMO" && <button className="assistant-incoming-toggle" onClick={() => setIncomingOpen((value) => !value)}>{incomingOpen ? "Cancelar mensaje del cliente" : "+ Simular respuesta del cliente"}</button>}
              {detail.channel === "DEMO" && incomingOpen && <form className="assistant-incoming-form" onSubmit={receiveDemo}>
                <textarea value={incomingBody} onChange={(event) => setIncomingBody(event.target.value)} placeholder="Ej.: ¿Puedo pagar un anticipo?" required />
                <button className="button button-secondary" disabled={working || !incomingBody.trim()}>Recibir en demo</button>
              </form>}
            </div>}
          </>}
        </div>

        <aside className="assistant-review">
          <div className="assistant-column-heading"><div><p className="eyebrow">HUMAN IN THE LOOP</p><h3>Propuesta comercial</h3></div></div>
          {!detail?.proposal ? <div className="assistant-empty">No hay una propuesta seleccionada.</div> : <>
            <div className="assistant-analysis-card">
              <span className="assistant-engine">✨ {detail.proposal.provider}</span>
              <h3>{detail.proposal.package_name}</h3>
              <strong>{formatCurrency(detail.proposal.package_price)}</strong>
              <p>{detail.proposal.recommendation}</p>
              <dl>
                <div><dt>Evento</dt><dd>{detail.proposal.event_type} · {detail.proposal.guest_count} personas</dd></div>
                <div><dt>Ubicación</dt><dd>{detail.proposal.event_location}</dd></div>
                <div><dt>Período</dt><dd>{formatDateTime(detail.proposal.start_at)} — {formatDateTime(detail.proposal.end_at)}</dd></div>
                <div><dt>Disponibilidad</dt><dd className={detail.proposal.available ? "available" : "unavailable"}>{detail.proposal.available ? "Confirmada" : "Requiere ajuste"}</dd></div>
              </dl>
            </div>

            <section className="assistant-customer-panel">
              <div><span>CLIENTE</span><strong>{detail.customer_name || "Contacto todavía no vinculado"}</strong></div>
              <label className="field"><span>Seleccionar cliente existente</span><select value={customerID} onChange={(event) => setCustomerID(event.target.value)} disabled={!canManage}><option value="">Selecciona un cliente</option>{customers.map((customer) => <option key={customer.id} value={customer.id}>{customer.display_name} · {customer.phone || "sin teléfono"}</option>)}</select></label>
              {canManage && <div className="assistant-customer-actions">
                <button className="button button-secondary" onClick={() => void linkCustomer()} disabled={working || !customerID || customerID === detail.customer_id}>Vincular seleccionado</button>
                {canManageCustomers && <button className="assistant-text-button" onClick={() => setCustomerCreatorOpen((value) => !value)}>{customerCreatorOpen ? "Cancelar" : "+ Crear desde el chat"}</button>}
              </div>}
              {customerCreatorOpen && canManageCustomers && <form className="assistant-customer-form" onSubmit={createCustomer}>
                <label className="field"><span>Nombre</span><input value={customerDraft.first_name} onChange={(event) => setCustomerDraft({ ...customerDraft, first_name: event.target.value })} required /></label>
                <label className="field"><span>Apellido</span><input value={customerDraft.last_name} onChange={(event) => setCustomerDraft({ ...customerDraft, last_name: event.target.value })} /></label>
                <label className="field"><span>Teléfono</span><input value={customerDraft.phone} onChange={(event) => setCustomerDraft({ ...customerDraft, phone: event.target.value })} placeholder="+50371234567" /></label>
                <label className="field"><span>Correo</span><input type="email" value={customerDraft.email} onChange={(event) => setCustomerDraft({ ...customerDraft, email: event.target.value })} /></label>
                <label className="field assistant-customer-company"><span>Empresa</span><input value={customerDraft.company_name} onChange={(event) => setCustomerDraft({ ...customerDraft, company_name: event.target.value })} /></label>
                <button className="button button-primary assistant-customer-create" disabled={working}>Crear y vincular</button>
              </form>}
            </section>

            {detail.proposal.quote_id ? <div className="assistant-quote-created">
              <span>✓</span><div><strong>{detail.proposal.quote_number ? formatQuoteNumber(detail.proposal.quote_number) : "Cotización DRAFT"}</strong><p>Creada con aprobación humana. No bloquea inventario.</p><Link href={`/quotes/${detail.proposal.quote_id}`}>Abrir cotización →</Link></div>
            </div> : <>
              <label className="field assistant-response-field"><span>Respuesta editable</span><textarea value={responseBody} onChange={(event) => setResponseBody(event.target.value)} disabled={!canManage} /></label>
              <div className="assistant-approval-note"><strong>Al aprobar:</strong><span>se registra el usuario y se crea un borrador de cotización. Después podrás entregar la respuesta en el simulador.</span></div>
              {canManage ? <button className="button button-primary assistant-approve-button" onClick={() => void approve()} disabled={working || !customerID || !detail.proposal.available}>{working ? "Procesando…" : "Aprobar y crear DRAFT"}</button> : <p className="assistant-readonly">Tu rol permite revisar, pero no aprobar propuestas.</p>}
            </>}

            {detail.proposal.quote_id && <section className="assistant-portal-panel">
              <div className="assistant-portal-heading">
                <div><span>PORTAL DEL CLIENTE</span><strong>{portalStatusLabel(detail.proposal.portal_status)}</strong></div>
                {detail.proposal.portal_status && <em className={`status-${detail.proposal.portal_status.toLowerCase()}`}>{detail.proposal.portal_status}</em>}
              </div>
              {detail.proposal.portal_status ? <dl>
                <div><dt>Vistas</dt><dd>{detail.proposal.portal_view_count}</dd></div>
                <div><dt>Decisión</dt><dd>{detail.proposal.portal_decision_at ? formatDateTime(detail.proposal.portal_decision_at) : "Pendiente"}</dd></div>
                {detail.proposal.reservation_number && <div><dt>Reserva</dt><dd>{formatReservationNumber(detail.proposal.reservation_number)}</dd></div>}
              </dl> : <p>Envía la cotización para generar un enlace bearer de una sola lectura. RentStage solo conservará su hash.</p>}
              {portalURL && <div className="assistant-portal-link-actions">
                <a className="button button-primary" href={portalURL} target="_blank" rel="noreferrer">Abrir como cliente</a>
                <button className="button button-secondary" onClick={() => void copyPortalURL()}>{portalCopied ? "Copiado ✓" : "Copiar enlace"}</button>
              </div>}
              {canManage && canManageQuote && ["DRAFT", "SENT"].includes(detail.proposal.quote_status || "") && <button className="button button-secondary assistant-portal-share" onClick={() => void shareQuoteDemo()} disabled={working}>{detail.proposal.portal_status === "ACTIVE" ? "Rotar enlace seguro" : detail.proposal.quote_status === "SENT" ? "Generar nuevo enlace seguro" : "Enviar cotización y generar portal"}</button>}
              {detail.proposal.portal_status && <button className="assistant-text-button" onClick={() => void refreshConversation()}>↻ Actualizar vistas y decisión</button>}
              <small>Aceptar en el portal es una decisión explícita del cliente. El asistente nunca reserva automáticamente.</small>
            </section>}
          </>}
        </aside>
      </section>
    </div>
  );
}