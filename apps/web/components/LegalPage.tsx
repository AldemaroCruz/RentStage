import Link from "next/link";
import { isPlaceholderLegalContact, legalContactEmail } from "@/lib/legal";

type Section = { title: string; paragraphs: string[] };

export function LegalPage({ eyebrow, title, summary, sections }: { eyebrow: string; title: string; summary: string; sections: Section[] }) {
  const email = legalContactEmail();
  const placeholder = isPlaceholderLegalContact(email);
  return <main className="legal-page"><article className="legal-document panel"><header><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p>{summary}</p><small>Última actualización: 20 de agosto de 2026</small></header>{placeholder && <div className="legal-warning">Antes de publicar o enviar la aplicación a revisión, configura <code>NEXT_PUBLIC_SUPPORT_EMAIL</code> con un correo real y monitoreado.</div>}{sections.map((section) => <section key={section.title}><h2>{section.title}</h2>{section.paragraphs.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}</section>)}<section><h2>Contacto</h2><p>Consultas de privacidad, soporte o eliminación: <a href={`mailto:${email}`}>{email}</a>.</p></section><footer><Link href="/privacy">Privacidad</Link><Link href="/terms">Términos</Link><Link href="/data-deletion">Eliminar datos</Link><Link href="/support">Soporte</Link><Link href="/login">Volver a RentStage</Link></footer></article></main>;
}
