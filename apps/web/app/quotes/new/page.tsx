import { QuoteEditor } from "@/components/QuoteEditor";

type Props = {
  searchParams: Promise<{ customer_id?: string; package_id?: string }>;
};

export default async function NewQuotePage({ searchParams }: Props) {
  const params = await searchParams;
  return (
    <div className="page-stack">
      <section className="page-heading quote-page-heading">
        <div><p className="eyebrow">QUOTE CORE</p><h2>Nueva cotización</h2><p>Selecciona un cliente, define el período y agrega recursos con precios históricos.</p></div>
      </section>
      <QuoteEditor presetCustomerID={params.customer_id} presetPackageID={params.package_id} />
    </div>
  );
}
