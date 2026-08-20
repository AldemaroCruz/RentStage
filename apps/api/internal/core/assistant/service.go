package assistant

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/core/customer"
	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
	"github.com/rentstage/rentstage/apps/api/internal/core/quote"
	"github.com/rentstage/rentstage/apps/api/internal/core/quoteportal"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	metaintegration "github.com/rentstage/rentstage/apps/api/internal/integrations/meta"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type WhatsAppSender interface {
	SendText(context.Context, string, string) (string, error)
}

type Service struct {
	repository         *Repository
	packageRepository  *packages.Repository
	packageService     *packages.Service
	customerRepository *customer.Repository
	quoteService       *quote.Service
	quotePortalService *quoteportal.Service
	audit              *audit.Repository
	whatsAppSender     WhatsAppSender
}

func NewService(
	repository *Repository,
	packageRepository *packages.Repository,
	packageService *packages.Service,
	customerRepository *customer.Repository,
	quoteService *quote.Service,
	quotePortalService *quoteportal.Service,
	auditRepository *audit.Repository,
	whatsAppSender WhatsAppSender,
) *Service {
	return &Service{
		repository: repository, packageRepository: packageRepository,
		packageService: packageService, customerRepository: customerRepository,
		quoteService: quoteService, quotePortalService: quotePortalService,
		audit: auditRepository, whatsAppSender: whatsAppSender,
	}
}

func (s *Service) ProcessMetaWebhook(
	ctx context.Context,
	events metaintegration.WebhookEvents,
) (metaintegration.ProcessResult, error) {
	return s.repository.ApplyMetaWebhook(ctx, events)
}

func (s *Service) Simulate(
	ctx context.Context,
	tenantID string,
	input SimulateInput,
) (ConversationDetail, map[string]string, error) {
	normalized, fields := normalizeSimulation(input)
	if len(fields) > 0 {
		return ConversationDetail{}, fields, nil
	}

	availablePackages, err := s.packageRepository.List(ctx, tenantID, "", "true")
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	candidates := rankPackages(availablePackages, normalized)
	if len(candidates) == 0 {
		return ConversationDetail{}, nil, ErrNoReadyPackage
	}

	selected := candidates[0]
	selectedAvailable := false
	for _, candidate := range candidates {
		availabilityResult, availabilityFields, availabilityErr := s.packageService.Availability(
			ctx,
			tenantID,
			candidate.ID,
			packages.AvailabilityInput{
				StartAt:  normalized.StartAt.Format(time.RFC3339),
				EndAt:    normalized.EndAt.Format(time.RFC3339),
				Quantity: 1,
			},
		)
		if availabilityErr != nil || len(availabilityFields) > 0 {
			continue
		}
		if availabilityResult.Available {
			selected = candidate
			selectedAvailable = true
			break
		}
	}

	template, templateFields, err := s.packageService.QuoteTemplate(ctx, tenantID, selected.ID, 1)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if len(templateFields) > 0 {
		return ConversationDetail{}, templateFields, nil
	}

	recommendation := fmt.Sprintf(
		"%s cubre la solicitud con %d recursos configurados y un precio de %s.",
		selected.Name,
		selected.ItemCount,
		formatUSD(template.EffectivePrice),
	)
	if selected.GuestCapacity != nil {
		recommendation = fmt.Sprintf(
			"%s está dimensionado para hasta %d personas, usa %d recursos configurados y cuesta %s.",
			selected.Name,
			*selected.GuestCapacity,
			selected.ItemCount,
			formatUSD(template.EffectivePrice),
		)
	}

	availabilityCopy := "La disponibilidad está confirmada para el período indicado."
	if !selectedAvailable {
		availabilityCopy = "La disponibilidad no está completa; el equipo debe ajustar la propuesta antes de aprobarla."
	}
	responseDraft := fmt.Sprintf(
		"¡Hola, %s! Gracias por escribirnos. Para tu %s en %s te recomendamos %s por %s. %s Si te parece bien, nuestro equipo puede preparar la cotización formal.",
		normalized.ContactName,
		strings.ToLower(normalized.EventType),
		normalized.EventLocation,
		selected.Name,
		formatUSD(template.EffectivePrice),
		availabilityCopy,
	)

	itemNames := make([]string, 0, len(template.Items))
	for _, item := range template.Items {
		itemNames = append(itemNames, fmt.Sprintf("%d × %s", item.Quantity, item.ResourceName))
	}
	detail, err := s.repository.CreateDemo(ctx, tenantID, normalized, proposalRecord{
		EventType: normalized.EventType, StartAt: normalized.StartAt,
		EndAt: normalized.EndAt, EventLocation: normalized.EventLocation,
		GuestCount: normalized.GuestCount, PackageID: selected.ID,
		PackageQuantity: 1, PackageName: selected.Name,
		PackagePrice: template.EffectivePrice, Available: selectedAvailable,
		Recommendation: recommendation, ResponseDraft: responseDraft,
		Evidence: map[string]any{
			"engine": "DEMO_RULES", "human_approval_required": true,
			"availability_checked": true, "available": selectedAvailable,
			"package_items": itemNames, "candidate_count": len(candidates),
		},
	})
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_DEMO_SIMULATED", "assistant_conversation", &detail.ID, map[string]any{
		"channel": "DEMO", "provider": "DEMO_RULES", "package_id": selected.ID,
		"available": selectedAvailable, "human_approval_required": true,
	})
	return detail, nil, nil
}

func (s *Service) Approve(
	ctx context.Context,
	tenantID, conversationID string,
	input ApproveInput,
) (ConversationDetail, map[string]string, error) {
	fields := map[string]string{}
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	input.ResponseBody = strings.TrimSpace(input.ResponseBody)
	if !idutil.IsUUID(input.CustomerID) {
		fields["customer_id"] = "Customer ID is invalid."
	}
	if len(input.ResponseBody) < 20 || len(input.ResponseBody) > 2000 {
		fields["response_body"] = "The approved response must contain between 20 and 2,000 characters."
	}
	if len(fields) > 0 {
		return ConversationDetail{}, fields, nil
	}

	detail, err := s.repository.Get(ctx, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if detail.Proposal == nil {
		return ConversationDetail{}, nil, ErrNotFound
	}
	if detail.Proposal.QuoteID != nil || detail.Proposal.Status != "PROPOSED" {
		return ConversationDetail{}, nil, ErrAlreadyApproved
	}
	if !detail.Proposal.Available {
		return ConversationDetail{}, nil, ErrUnavailable
	}
	if _, err := s.customerRepository.Get(ctx, tenantID, input.CustomerID); err != nil {
		if err == customer.ErrNotFound {
			return ConversationDetail{}, nil, ErrCustomerMissing
		}
		return ConversationDetail{}, nil, err
	}

	availabilityResult, availabilityFields, err := s.packageService.Availability(ctx, tenantID, detail.Proposal.PackageID, packages.AvailabilityInput{
		StartAt:  detail.Proposal.StartAt.Format(time.RFC3339),
		EndAt:    detail.Proposal.EndAt.Format(time.RFC3339),
		Quantity: detail.Proposal.PackageQuantity,
	})
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if len(availabilityFields) > 0 {
		return ConversationDetail{}, availabilityFields, nil
	}
	if !availabilityResult.Available {
		return ConversationDetail{}, nil, ErrUnavailable
	}

	template, templateFields, err := s.packageService.QuoteTemplate(
		ctx, tenantID, detail.Proposal.PackageID, detail.Proposal.PackageQuantity,
	)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if len(templateFields) > 0 {
		return ConversationDetail{}, templateFields, nil
	}
	quoteItems := make([]quote.ItemInput, 0, len(template.Items))
	for _, item := range template.Items {
		quoteItems = append(quoteItems, quote.ItemInput{
			ResourceID: item.ResourceID, Description: item.Description,
			Quantity: item.Quantity, UnitPrice: item.UnitPrice,
			DiscountAmount: item.DiscountAmount,
		})
	}
	eventType := detail.Proposal.EventType
	eventLocation := detail.Proposal.EventLocation
	expiresAt := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	createdQuote, quoteFields, err := s.quoteService.Create(ctx, tenantID, quote.CreateInput{
		CustomerID: input.CustomerID,
		StartAt:    detail.Proposal.StartAt.Format(time.RFC3339),
		EndAt:      detail.Proposal.EndAt.Format(time.RFC3339),
		EventType:  &eventType, EventLocation: &eventLocation,
		ExtraCharges: template.ExtraCharges,
		Notes:        fmt.Sprintf("Borrador creado con aprobación humana desde el asistente comercial. Conversación %s. No reserva inventario.", conversationID),
		ExpiresAt:    &expiresAt, Items: quoteItems,
	})
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if len(quoteFields) > 0 {
		return ConversationDetail{}, quoteFields, nil
	}

	approved, err := s.repository.Approve(
		ctx, tenantID, conversationID, input.CustomerID, createdQuote.ID,
		input.ResponseBody, webutil.ActorID(ctx),
	)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_PROPOSAL_APPROVED", "assistant_conversation", &conversationID, map[string]any{
		"quote_id": createdQuote.ID, "quote_number": createdQuote.QuoteNumber,
		"quote_status": createdQuote.Status, "response_sent": false,
	})
	return approved, nil, nil
}

func (s *Service) LinkCustomer(
	ctx context.Context,
	tenantID, conversationID string,
	input LinkCustomerInput,
) (ConversationDetail, map[string]string, error) {
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	if !idutil.IsUUID(input.CustomerID) {
		return ConversationDetail{}, map[string]string{
			"customer_id": "Customer ID is invalid.",
		}, nil
	}
	if _, err := s.customerRepository.Get(ctx, tenantID, input.CustomerID); err != nil {
		if err == customer.ErrNotFound {
			return ConversationDetail{}, nil, ErrCustomerMissing
		}
		return ConversationDetail{}, nil, err
	}
	linked, err := s.repository.LinkCustomer(ctx, tenantID, conversationID, input.CustomerID)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_CUSTOMER_LINKED", "assistant_conversation", &conversationID, map[string]any{
		"customer_id": input.CustomerID,
		"channel":     linked.Channel,
	})
	return linked, nil, nil
}

func (s *Service) SendDemo(
	ctx context.Context,
	tenantID, conversationID string,
	input SendDemoInput,
) (ConversationDetail, map[string]string, error) {
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Body = strings.TrimSpace(input.Body)
	fields := map[string]string{}
	if input.MessageID != "" && !idutil.IsUUID(input.MessageID) {
		fields["message_id"] = "Message ID is invalid."
	}
	if input.Body == "" || len(input.Body) > 2000 {
		fields["body"] = "Message must contain between 1 and 2,000 characters."
	}
	if len(fields) > 0 {
		return ConversationDetail{}, fields, nil
	}

	detail, err := s.repository.SendDemo(
		ctx,
		tenantID,
		conversationID,
		input.MessageID,
		input.Body,
		webutil.ActorID(ctx),
	)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_DEMO_MESSAGE_SENT", "assistant_conversation", &conversationID, map[string]any{
		"channel":             "DEMO",
		"message_id":          input.MessageID,
		"simulated_delivery":  true,
		"real_phone_delivery": false,
	})
	return detail, nil, nil
}

func (s *Service) Send(
	ctx context.Context,
	tenantID, conversationID string,
	input SendInput,
) (ConversationDetail, map[string]string, error) {
	input.MessageID = strings.TrimSpace(input.MessageID)
	input.Body = strings.TrimSpace(input.Body)
	fields := map[string]string{}
	if input.MessageID != "" && !idutil.IsUUID(input.MessageID) {
		fields["message_id"] = "Message ID is invalid."
	}
	if input.Body == "" || len(input.Body) > 2000 {
		fields["body"] = "Message must contain between 1 and 2,000 characters."
	}
	if len(fields) > 0 {
		return ConversationDetail{}, fields, nil
	}
	detail, err := s.repository.Get(ctx, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if detail.Channel == "DEMO" {
		return s.SendDemo(ctx, tenantID, conversationID, input)
	}
	if detail.ConsentStatus == "OPTED_OUT" {
		return ConversationDetail{}, nil, ErrConsentRevoked
	}
	if s.whatsAppSender == nil {
		return ConversationDetail{}, nil, ErrProviderDisabled
	}
	if detail.ServiceWindowExpiresAt == nil || !time.Now().UTC().Before(*detail.ServiceWindowExpiresAt) {
		return ConversationDetail{}, nil, ErrServiceWindowClosed
	}
	externalMessageID, err := s.whatsAppSender.SendText(ctx, detail.ContactPhone, input.Body)
	if err != nil {
		return ConversationDetail{}, nil, fmt.Errorf("%w: %v", ErrProviderDelivery, err)
	}
	sent, err := s.repository.RecordWhatsAppSent(
		ctx, tenantID, conversationID, input.MessageID,
		externalMessageID, input.Body, webutil.ActorID(ctx),
	)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_WHATSAPP_MESSAGE_SENT", "assistant_conversation", &conversationID, map[string]any{
		"channel": "WHATSAPP", "provider_message_id": externalMessageID,
		"human_approved": true,
	})
	return sent, nil, nil
}

func (s *Service) ReceiveDemo(
	ctx context.Context,
	tenantID, conversationID string,
	input ReceiveDemoInput,
) (ConversationDetail, map[string]string, error) {
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" || len(input.Body) > 2000 {
		return ConversationDetail{}, map[string]string{
			"body": "Message must contain between 1 and 2,000 characters.",
		}, nil
	}
	detail, err := s.repository.Get(ctx, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if detail.Channel != "DEMO" {
		return ConversationDetail{}, nil, ErrDemoOnly
	}

	responseDraft := draftDemoReply(detail, input.Body)
	received, err := s.repository.ReceiveDemo(
		ctx,
		tenantID,
		conversationID,
		input.Body,
		responseDraft,
	)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_DEMO_REPLY_SIMULATED", "assistant_conversation", &conversationID, map[string]any{
		"channel":                 "DEMO",
		"draft_created":           true,
		"human_approval_required": true,
	})
	return received, nil, nil
}

func (s *Service) ShareQuoteDemo(
	ctx context.Context,
	tenantID, conversationID string,
	input ShareQuoteDemoInput,
) (ConversationDetail, map[string]string, error) {
	input.Body = strings.TrimSpace(input.Body)
	if len(input.Body) > 2000 {
		return ConversationDetail{}, map[string]string{
			"body": "Message must contain 2,000 characters or fewer.",
		}, nil
	}

	detail, err := s.repository.Get(ctx, tenantID, conversationID)
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if detail.Channel != "DEMO" {
		return ConversationDetail{}, nil, ErrDemoOnly
	}
	if detail.Proposal == nil || detail.Proposal.QuoteID == nil || detail.Proposal.QuoteNumber == nil {
		return ConversationDetail{}, nil, ErrQuoteMissing
	}

	actorID := webutil.ActorID(ctx)
	var issued quote.Detail
	if detail.Proposal.QuoteStatus != nil && *detail.Proposal.QuoteStatus == "SENT" {
		issued, err = s.quotePortalService.Reissue(ctx, tenantID, *detail.Proposal.QuoteID, actorID)
	} else {
		issued, err = s.quotePortalService.Send(ctx, tenantID, *detail.Proposal.QuoteID, actorID)
	}
	if err != nil {
		return ConversationDetail{}, nil, err
	}
	if issued.Portal == nil || issued.Portal.PublicURL == "" {
		return ConversationDetail{}, nil, ErrPortalDeliveryMissing
	}

	body := input.Body
	if body == "" {
		body = defaultPortalMessage(issued.QuoteNumber)
	}
	recorded, recordErr := s.repository.RecordQuoteShared(
		ctx, tenantID, conversationID, issued.QuoteNumber,
		issued.Portal.Revision, body, actorID,
	)
	if recordErr != nil {
		// Issuing the portal already invalidated any previous raw token. Keep
		// returning the new one-time URL even if the optional chat transcript
		// could not be refreshed; otherwise the only usable token would be lost.
		recorded = detail
		recorded.Proposal.QuoteStatus = &issued.Status
		recorded.Proposal.PortalStatus = &issued.Portal.Status
		recorded.Proposal.PortalViewCount = issued.Portal.ViewCount
		recorded.Proposal.PortalViewedAt = issued.Portal.LastViewedAt
		recorded.Proposal.PortalDecisionAt = issued.Portal.DecisionAt
	}
	recorded.PortalDelivery = &PortalDelivery{
		QuoteID: issued.ID, QuoteNumber: issued.QuoteNumber,
		PublicURL: issued.Portal.PublicURL, ExpiresAt: issued.Portal.ExpiresAt,
	}
	_ = s.audit.Record(ctx, tenantID, "ASSISTANT_QUOTE_PORTAL_SHARED_DEMO", "assistant_conversation", &conversationID, map[string]any{
		"quote_id": issued.ID, "quote_number": issued.QuoteNumber,
		"portal_revision":        issued.Portal.Revision,
		"chat_evidence_recorded": recordErr == nil,
		"raw_token_persisted":    false, "real_phone_delivery": false,
	})
	return recorded, nil, nil
}

func defaultPortalMessage(quoteNumber int64) string {
	return fmt.Sprintf(
		"Te compartimos la cotización COT-%06d para que revises fechas, precios y condiciones en el portal seguro de RentStage.",
		quoteNumber,
	)
}

func draftDemoReply(detail ConversationDetail, inboundBody string) string {
	contactName := strings.TrimSpace(detail.ContactName)
	if contactName == "" {
		contactName = "cliente"
	}
	context := strings.ToLower(inboundBody)
	prefix := fmt.Sprintf("Gracias por el seguimiento, %s. ", contactName)

	var response string
	switch {
	case strings.Contains(context, "descuento"), strings.Contains(context, "precio"), strings.Contains(context, "costo"):
		response = "Podemos revisar las condiciones comerciales antes de enviarte la versión final; por ahora la cotización permanece como borrador y no reserva inventario."
	case strings.Contains(context, "disponib"), strings.Contains(context, "fecha"), strings.Contains(context, "hora"):
		response = "El período propuesto sigue registrado, pero nuestro equipo confirmará nuevamente la disponibilidad antes de convertir la cotización en reserva."
	case strings.Contains(context, "pago"), strings.Contains(context, "depósito"), strings.Contains(context, "deposito"), strings.Contains(context, "anticipo"):
		response = "Podemos acordar el anticipo y la forma de pago dentro del flujo formal de cotización; nada se cobrará ni reservará desde este chat."
	case strings.Contains(context, "incluye"), strings.Contains(context, "equipo"), strings.Contains(context, "paquete"):
		response = "La propuesta conserva el paquete y sus recursos configurados; nuestro equipo puede detallar cada componente antes de enviarte la cotización."
	case strings.Contains(context, "gracias"), strings.Contains(context, "listo"), strings.Contains(context, "perfecto"):
		response = "Con gusto. Dejaremos el seguimiento preparado para que una persona del equipo confirme el siguiente paso contigo."
	default:
		response = "Una persona del equipo revisará tu mensaje y podrá ajustar la propuesta antes de responderte."
	}

	if detail.Proposal != nil && detail.Proposal.QuoteNumber != nil {
		response = fmt.Sprintf(
			"La cotización COT-%06d continúa en borrador. %s",
			*detail.Proposal.QuoteNumber,
			response,
		)
	}
	return prefix + response
}

type rankedPackage struct {
	item  packages.Summary
	score int
	gap   int
}

func rankPackages(items []packages.Summary, input normalizedSimulation) []packages.Summary {
	searchText := strings.ToLower(input.Message + " " + input.EventType)
	ranked := make([]rankedPackage, 0, len(items))
	for _, item := range items {
		if !item.Active || !item.Ready {
			continue
		}
		score := 0
		combined := strings.ToLower(item.Name + " " + item.Description)
		for _, keyword := range []string{"boda", "fiesta", "corporativo", "concierto", "sonido", "audio", "evento"} {
			if strings.Contains(searchText, keyword) && strings.Contains(combined, keyword) {
				score += 40
			}
		}
		gap := 1_000_000
		if item.GuestCapacity != nil {
			if *item.GuestCapacity >= input.GuestCount {
				score += 100
				gap = *item.GuestCapacity - input.GuestCount
			} else {
				score -= 100
				gap = input.GuestCount - *item.GuestCapacity
			}
		}
		ranked = append(ranked, rankedPackage{item: item, score: score, gap: gap})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].gap != ranked[j].gap {
			return ranked[i].gap < ranked[j].gap
		}
		return ranked[i].item.Name < ranked[j].item.Name
	})
	result := make([]packages.Summary, 0, len(ranked))
	for _, candidate := range ranked {
		result = append(result, candidate.item)
	}
	return result
}

func normalizeSimulation(input SimulateInput) (normalizedSimulation, map[string]string) {
	result := normalizedSimulation{
		ContactName:   strings.TrimSpace(input.ContactName),
		ContactPhone:  strings.TrimSpace(input.ContactPhone),
		Message:       strings.TrimSpace(input.Message),
		EventType:     strings.TrimSpace(input.EventType),
		EventLocation: strings.TrimSpace(input.EventLocation),
		GuestCount:    input.GuestCount,
	}
	fields := map[string]string{}
	if result.ContactName == "" || len(result.ContactName) > 240 {
		fields["contact_name"] = "Contact name is required and must be 240 characters or fewer."
	}
	if !strings.HasPrefix(result.ContactPhone, "+") || len(result.ContactPhone) < 9 || len(result.ContactPhone) > 16 {
		fields["contact_phone"] = "Use international format, for example +50371234567."
	}
	if len(result.Message) < 10 || len(result.Message) > 2000 {
		fields["message"] = "Message must contain between 10 and 2,000 characters."
	}
	if result.EventType == "" || len(result.EventType) > 120 {
		fields["event_type"] = "Event type is required and must be 120 characters or fewer."
	}
	if result.EventLocation == "" || len(result.EventLocation) > 500 {
		fields["event_location"] = "Event location is required and must be 500 characters or fewer."
	}
	if result.GuestCount <= 0 || result.GuestCount > 1_000_000 {
		fields["guest_count"] = "Guest count must be between 1 and 1,000,000."
	}
	start, startErr := time.Parse(time.RFC3339, strings.TrimSpace(input.StartAt))
	if startErr != nil {
		fields["start_at"] = "Use an RFC3339 timestamp."
	} else {
		result.StartAt = start
	}
	end, endErr := time.Parse(time.RFC3339, strings.TrimSpace(input.EndAt))
	if endErr != nil {
		fields["end_at"] = "Use an RFC3339 timestamp."
	} else {
		result.EndAt = end
	}
	if startErr == nil && endErr == nil && !result.EndAt.After(result.StartAt) {
		fields["end_at"] = "End must be after start."
	}
	return result, fields
}

func formatUSD(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}
