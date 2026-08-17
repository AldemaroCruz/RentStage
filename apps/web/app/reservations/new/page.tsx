import { ReservationEditor } from "@/components/ReservationEditor";

export default function NewReservationPage() {
  return (
    <div className="page-stack">
      <section className="quote-page-heading">
        <div>
          <p className="eyebrow">BOOKING CORE</p>
          <h2>Nueva reserva manual</h2>
          <p>Crea un compromiso inmediato de inventario con disponibilidad transaccional.</p>
        </div>
      </section>
      <ReservationEditor />
    </div>
  );
}
