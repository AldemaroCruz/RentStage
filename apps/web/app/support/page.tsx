import { LegalPage } from "@/components/LegalPage";

export default function SupportPage() { return <LegalPage eyebrow="AYUDA" title="Soporte de RentStage" summary="Canal para consultas técnicas, privacidad, seguridad y preparación de la integración de WhatsApp." sections={[
  { title: "Qué incluir", paragraphs: ["Describe el problema, la sección afectada, la hora aproximada y el identificador de solicitud visible. No compartas contraseñas, access tokens, app secrets ni credenciales de servicio."] },
  { title: "Incidentes de seguridad", paragraphs: ["Indica claramente que se trata de un reporte de seguridad. Revocaremos credenciales expuestas y coordinaremos la investigación por un canal verificado."] },
]}/>; }
