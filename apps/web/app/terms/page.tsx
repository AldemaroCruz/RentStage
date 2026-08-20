import { LegalPage } from "@/components/LegalPage";

export default function TermsPage() { return <LegalPage eyebrow="CONDICIONES" title="Términos de uso" summary="Condiciones preliminares para evaluar RentStage antes de una publicación comercial definitiva." sections={[
  { title: "Uso autorizado", paragraphs: ["RentStage debe utilizarse únicamente para operaciones legítimas de alquiler y comunicaciones autorizadas. La empresa usuaria es responsable por los datos, permisos y mensajes que administra."] },
  { title: "WhatsApp y revisión humana", paragraphs: ["La integración no debe emplearse para spam. Los mensajes comerciales requieren consentimiento, respetar las bajas y seguir las políticas vigentes de Meta. RentStage mantiene aprobación humana y bloqueos de entrega durante esta fase de preparación."] },
  { title: "Disponibilidad", paragraphs: ["La versión actual es una plataforma en evolución. Antes de ofrecerla como servicio comercial se publicarán términos definitivos, entidad operadora, jurisdicción y condiciones de soporte."] },
]}/>; }
