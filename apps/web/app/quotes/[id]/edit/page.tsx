"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { QuoteEditor } from "@/components/QuoteEditor";
import { api } from "@/lib/api";
import { formatQuoteNumber } from "@/lib/format";
import type { QuoteDetail } from "@/lib/types";

export default function EditQuotePage() {
  const params = useParams<{ id: string }>();
  const [quote, setQuote] = useState<QuoteDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<QuoteDetail>(`/api/v1/quotes/${params.id}`)
      .then(setQuote)
      .catch((reason) => setError(reason instanceof Error ? reason.message : "No fue posible cargar la cotización."));
  }, [params.id]);

  if (error) return <div className="panel inline-error">{error}</div>;
  if (!quote) return <div className="skeleton detail-skeleton" />;
  if (quote.status !== "DRAFT") return <div className="panel inline-error">Solo los borradores pueden editarse.</div>;

  return (
    <div className="page-stack">
      <section className="page-heading quote-page-heading">
        <div><p className="eyebrow">{formatQuoteNumber(quote.quote_number)}</p><h2>Editar cotización</h2><p>Los cambios reemplazan el contenido del borrador y conservan el mismo número.</p></div>
      </section>
      <QuoteEditor initial={quote} />
    </div>
  );
}
