import { LegalPage } from "@/components/LegalPage";

export default function PrivacyPage() { return <LegalPage eyebrow="PRIVACIDAD" title="Política de privacidad" summary="Esta página describe cómo RentStage procesa información de cuentas, operaciones de alquiler y conversaciones conectadas por una empresa usuaria." sections={[
  { title: "Información procesada", paragraphs: ["RentStage procesa datos de cuenta y workspace, clientes, inventario, cotizaciones, reservas, pagos, auditoría y, cuando la empresa lo habilita, identificadores y contenido de conversaciones de WhatsApp.", "No vendemos información personal ni la utilizamos para crear audiencias publicitarias."] },
  { title: "Finalidad y control", paragraphs: ["La información se usa para prestar la operación solicitada por la empresa usuaria, proteger el servicio, mantener trazabilidad y atender solicitudes de soporte.", "Cada empresa controla sus contactos y debe contar con una base legal o consentimiento válido para comunicarse con ellos."] },
  { title: "Conservación y seguridad", paragraphs: ["Los datos se conservan mientras la cuenta esté activa o durante el plazo necesario para obligaciones legales, seguridad y resolución de disputas. Aplicamos aislamiento por empresa, controles de acceso y registros de auditoría."] },
]}/>; }
