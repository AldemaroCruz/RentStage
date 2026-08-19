"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, formatDateTime, formatQuoteNumber } from "@/lib/format";
import type {
  AssistantConversationDetail,
  AssistantConversationSummary,
  Customer,
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

export default function AssistantPage() {
  const { can } = useAuth();
  const canManage = can("assistant.manage");
  const [items, setItems] = useState<AssistantConversationSummary[]>([]);
  const [detail, setDetail] = useState<AssistantConversationDetail | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [customerID, setCustomerID] = useState("");
  const [responseBody, setResponseBody] = useState("");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [composerOpen, setComposerOpen] = useState(false);
  const [simulation, setSimulation] = useState(initialSimulation);

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

  useEffect(() => {
    Promise.all([
      api<{ items: AssistantConversationSummary[] }>("/api/v1/assistant/conversations"),
      api<{ items: Customer[] }>("/api/v1/customers"),
    ])
      .then(async ([conversationResponse, customerResponse]) => {
        setItems(conversationResponse.items);
        setCustomers(customerResponse.items);
        if (conversationResponse.items[0]) {
          setDetail(await api<AssistantConversationDetail>(`/api/v1/assistant/conversations/${conversationResponse.items[0].id}`));
        }
      })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar el asistente."))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    setResponseBody(detail?.proposal?.response_draft || "");
    setCustomerID(detail?.customer_id || customers[0]?.id || "");
  }, [detail, customers]);

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
      setNotice("Aprobación registrada. Se creó una cotización DRAFT; el inventario continúa sin reservarse.");
      await loadList(approved.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "No fue posible crear el borrador.");
    } finally {
      setWorking(false);
    }
  }

  const metrics = useMemo(() => ({
    review: items.filter((item) => item.status === "HUMAN_REVIEW").length,
    drafted: items.filter((item) => item.status === "QUOTE_DRAFTED").length,
    demo: items.filter((item) => item.channel === "DEMO").length,
  }), [items]);

  return (
    <div className="page-stack assistant-page">
      <section className="assistant-hero">
        <div>
          <p className="eyebrow">WHATSAPP SALES ASSISTANT · V0.15</p>
          <h2>Convierte consultas en cotizaciones, con una persona al mando.</h2>
          <p>El simulador usa clientes, paquetes, precios y disponibilidad reales de RentStage. Meta se conecta después sin cambiar el flujo comercial.</p>
        </div>
        <div className="assistant-hero-actions">
          <span className="assistant-provider-pill"><i /> CANAL DEMO</span>
          {canManage && <button className="button button-primary" onClick={() => setComposerOpen((value) => !value)}>{composerOpen ? "Cerrar simulador" : "Simular consulta"}</button>}
        </div>
      </section>

      <section className="assistant-boundary-strip">
        <strong>Control humano obligatorio</strong>
        <span>La sugerencia no se envía sola.</span>
        <span>La aprobación crea únicamente una cotización DRAFT.</span>
        <span>Nunca reserva inventario automáticamente.</span>
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
        <article><span>Conversaciones</span><strong>{items.length}</strong><small>{metrics.demo} en canal demo</small></article>
        <article><span>Revisión humana</span><strong>{metrics.review}</strong><small>respuestas esperando aprobación</small></article>
        <article><span>Cotizaciones creadas</span><strong>{metrics.drafted}</strong><small>todas permanecen en DRAFT</small></article>
      </section>

      <section className="assistant-workspace panel">
        <aside className="assistant-inbox">
          <div className="assistant-column-heading"><div><p className="eyebrow">INBOX</p><h3>Conversaciones</h3></div><span>{items.length}</span></div>
          {loading ? <div className="table-skeleton">Cargando conversaciones…</div> : items.length === 0 ? <div className="assistant-empty">Usa “Simular consulta” para iniciar.</div> : items.map((item) => (
            <button key={item.id} className={`assistant-conversation-row ${detail?.id === item.id ? "active" : ""}`} onClick={() => void selectConversation(item.id)}>
              <span className="assistant-contact-avatar">{item.contact_name.slice(0, 2).toUpperCase()}</span>
              <span><strong>{item.contact_name}</strong><small>{item.last_message}</small><em>{statusLabel(item.status)}</em></span>
              <time>{new Date(item.last_message_at).toLocaleDateString("es-SV", { day: "2-digit", month: "short" })}</time>
            </button>
          ))}
        </aside>

        <div className="assistant-chat">
          {!detail ? <div className="assistant-empty large">Selecciona o simula una conversación.</div> : <>
            <header className="assistant-chat-header">
              <div className="assistant-contact-avatar">{detail.contact_name.slice(0, 2).toUpperCase()}</div>
              <div><strong>{detail.contact_name}</strong><span>{detail.contact_phone} · {detail.channel === "DEMO" ? "Simulador integrado" : "WhatsApp"}</span></div>
              <em className={`assistant-status ${detail.status.toLowerCase()}`}>{statusLabel(detail.status)}</em>
            </header>
            <div className="assistant-chat-scroll">
              <div className="assistant-chat-day">HOY · DEMOSTRACIÓN</div>
              {detail.messages.map((message) => (
                <article key={message.id} className={`assistant-bubble ${message.direction === "OUTBOUND" ? "outbound" : "inbound"} ${message.status === "DRAFT" ? "draft" : ""}`}>
                  {message.sender_type === "ASSISTANT" && <small>✨ ASISTENTE · {message.status === "DRAFT" ? "BORRADOR" : "APROBADO"}</small>}
                  <p>{message.body}</p>
                  <time>{formatDateTime(message.created_at)}</time>
                </article>
              ))}
            </div>
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

            {detail.proposal.quote_id ? <div className="assistant-quote-created">
              <span>✓</span><div><strong>{detail.proposal.quote_number ? formatQuoteNumber(detail.proposal.quote_number) : "Cotización DRAFT"}</strong><p>Creada con aprobación humana. No bloquea inventario.</p><Link href={`/quotes/${detail.proposal.quote_id}`}>Abrir cotización →</Link></div>
            </div> : <>
              <label className="field"><span>Cliente de la cotización</span><select value={customerID} onChange={(event) => setCustomerID(event.target.value)} disabled={!canManage}>{customers.map((customer) => <option key={customer.id} value={customer.id}>{customer.display_name} · {customer.phone || "sin teléfono"}</option>)}</select></label>
              <label className="field assistant-response-field"><span>Respuesta editable</span><textarea value={responseBody} onChange={(event) => setResponseBody(event.target.value)} disabled={!canManage} /></label>
              <div className="assistant-approval-note"><strong>Al aprobar:</strong><span>se registra el usuario, la evidencia y se crea un borrador de cotización. El mensaje no sale hacia ningún teléfono.</span></div>
              {canManage ? <button className="button button-primary assistant-approve-button" onClick={() => void approve()} disabled={working || !customerID || !detail.proposal.available}>{working ? "Procesando…" : "Aprobar y crear DRAFT"}</button> : <p className="assistant-readonly">Tu rol permite revisar, pero no aprobar propuestas.</p>}
            </>}
          </>}
        </aside>
      </section>
    </div>
  );
}
