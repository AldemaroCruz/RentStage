package commercialmetrics

import "time"

type Window struct {
	Days    int       `json:"days"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type Overview struct {
	Inquiries                int     `json:"inquiries"`
	PublicRequests           int     `json:"public_requests"`
	AssistantConversations   int     `json:"assistant_conversations"`
	NewCustomers             int     `json:"new_customers"`
	QuotesCreated            int     `json:"quotes_created"`
	QuotesPresented          int     `json:"quotes_presented"`
	QuotesAccepted           int     `json:"quotes_accepted"`
	QuotesRejected           int     `json:"quotes_rejected"`
	ReservationsCreated      int     `json:"reservations_created"`
	QuoteReservationsCreated int     `json:"quote_reservations_created"`
	InvoicesIssued           int     `json:"invoices_issued"`
	QuoteAcceptanceRate      float64 `json:"quote_acceptance_rate"`
	QuoteToReservationRate   float64 `json:"quote_to_reservation_rate"`
	AverageResponseMinutes   float64 `json:"average_response_minutes"`
	ResponseSamples          int     `json:"response_samples"`
	QuotePipelineValue       float64 `json:"quote_pipeline_value"`
	AcceptedQuoteValue       float64 `json:"accepted_quote_value"`
	ReservationValue         float64 `json:"reservation_value"`
	IssuedValue              float64 `json:"issued_value"`
	CollectedValue           float64 `json:"collected_value"`
	OutstandingValue         float64 `json:"outstanding_value"`
	AuditEvents              int     `json:"audit_events"`
	HumanApprovedMessages    int     `json:"human_approved_messages"`
	CustomerPortalDecisions  int     `json:"customer_portal_decisions"`
}

type FunnelStage struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}

type ReservationOutcomes struct {
	Active           int     `json:"active"`
	Completed        int     `json:"completed"`
	Cancelled        int     `json:"cancelled"`
	CancellationRate float64 `json:"cancellation_rate"`
}

type SourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type MonthlyActivity struct {
	Month            string  `json:"month"`
	QuoteValue       float64 `json:"quote_value"`
	ReservationValue float64 `json:"reservation_value"`
	CollectedValue   float64 `json:"collected_value"`
}

type Report struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Currency    string              `json:"currency"`
	Window      Window              `json:"window"`
	Overview    Overview            `json:"overview"`
	Funnel      []FunnelStage       `json:"funnel"`
	Outcomes    ReservationOutcomes `json:"reservation_outcomes"`
	Sources     []SourceCount       `json:"customer_sources"`
	Monthly     []MonthlyActivity   `json:"monthly_activity"`
}

func percentage(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	value := float64(numerator) / float64(denominator) * 100
	return float64(int(value*10+0.5)) / 10
}

func (report *Report) finalize() {
	report.Overview.Inquiries = report.Overview.PublicRequests + report.Overview.AssistantConversations
	report.Overview.QuoteAcceptanceRate = percentage(
		report.Overview.QuotesAccepted,
		report.Overview.QuotesAccepted+report.Overview.QuotesRejected,
	)
	report.Overview.QuoteToReservationRate = percentage(
		report.Overview.QuoteReservationsCreated,
		report.Overview.QuotesAccepted,
	)
	report.Outcomes.CancellationRate = percentage(
		report.Outcomes.Cancelled,
		report.Outcomes.Completed+report.Outcomes.Cancelled,
	)
	report.Funnel = []FunnelStage{
		{Key: "inquiries", Label: "Consultas", Count: report.Overview.Inquiries, Description: "Solicitudes web y conversaciones nuevas"},
		{Key: "quotes", Label: "Cotizaciones", Count: report.Overview.QuotesCreated, Description: "Propuestas creadas durante el período"},
		{Key: "accepted", Label: "Aceptadas", Count: report.Overview.QuotesAccepted, Description: "Cotizaciones con decisión positiva"},
		{Key: "reservations", Label: "Reservas", Count: report.Overview.ReservationsCreated, Description: "Compromisos operativos creados"},
		{Key: "invoices", Label: "Facturas", Count: report.Overview.InvoicesIssued, Description: "Documentos emitidos"},
	}
}
