import { PackageEditor } from "@/components/PackageEditor";

export default function NewPackagePage() {
  return (
    <div className="page-stack">
      <section className="page-heading quote-page-heading">
        <div><p className="eyebrow">PACKAGES CORE</p><h2>Nuevo paquete</h2><p>Define una combinación reusable de recursos, cantidades y precio comercial.</p></div>
      </section>
      <PackageEditor />
    </div>
  );
}
