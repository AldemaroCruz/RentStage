import { LegalPage } from "@/components/LegalPage";

export default function DataDeletionPage() { return <LegalPage eyebrow="CONTROL DE DATOS" title="Solicitud de eliminación de datos" summary="Puedes solicitar acceso, corrección o eliminación de la información asociada a RentStage." sections={[
  { title: "Cómo solicitarla", paragraphs: ["Envía un correo desde la dirección asociada a tu cuenta e incluye el workspace, correo de usuario y, si corresponde, el número de WhatsApp en formato internacional. No envíes contraseñas, tokens ni documentos secretos."] },
  { title: "Verificación y plazo", paragraphs: ["Verificaremos identidad y autoridad sobre el workspace antes de ejecutar cambios. Confirmaremos la recepción y comunicaremos el resultado o cualquier retención legal aplicable."] },
  { title: "Alcance", paragraphs: ["La eliminación puede incluir perfil, conversaciones, clientes y datos operativos. Ciertos registros mínimos podrían conservarse por seguridad, facturación, auditoría o cumplimiento legal y luego eliminarse según el plazo aplicable."] },
]}/>; }
