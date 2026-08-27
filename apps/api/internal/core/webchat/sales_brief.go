package webchat

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type DraftSalesSignalKind string

type DraftSalesMissingField string

const (
	DraftSalesSignalEventType  DraftSalesSignalKind = "EVENT_TYPE"
	DraftSalesSignalEventDate  DraftSalesSignalKind = "EVENT_DATE"
	DraftSalesSignalLocation   DraftSalesSignalKind = "LOCATION"
	DraftSalesSignalGuestCount DraftSalesSignalKind = "GUEST_COUNT"
	DraftSalesSignalBudget     DraftSalesSignalKind = "BUDGET"

	DraftSalesMissingEventType  DraftSalesMissingField = "EVENT_TYPE"
	DraftSalesMissingEventDate  DraftSalesMissingField = "EVENT_DATE"
	DraftSalesMissingLocation   DraftSalesMissingField = "LOCATION"
	DraftSalesMissingGuestCount DraftSalesMissingField = "GUEST_COUNT"
	DraftSalesMissingBudget     DraftSalesMissingField = "BUDGET"

	MaximumDraftSalesSignals       = 5
	MaximumDraftSalesMissingFields = 5
	MaximumDraftSalesSignalRunes   = 180
	MaximumDraftNextQuestionRunes  = 300
)

type DraftSalesSignal struct {
	Kind  DraftSalesSignalKind `json:"kind"`
	Value string               `json:"value"`
}

type DraftSalesBrief struct {
	Signals       []DraftSalesSignal       `json:"signals"`
	MissingFields []DraftSalesMissingField `json:"missing_fields"`
	NextQuestion  string                   `json:"next_question"`
}

func emptyDraftSalesBrief() DraftSalesBrief {
	return DraftSalesBrief{
		Signals:       []DraftSalesSignal{},
		MissingFields: []DraftSalesMissingField{},
	}
}

func NormalizeDraftSalesBrief(
	brief DraftSalesBrief,
	request DraftRequest,
) (DraftSalesBrief, error) {
	if len(brief.Signals) > MaximumDraftSalesSignals {
		return DraftSalesBrief{}, fmt.Errorf(
			"%w: sales brief exceeds %d signals",
			ErrInvalidDraft,
			MaximumDraftSalesSignals,
		)
	}
	if len(brief.MissingFields) > MaximumDraftSalesMissingFields {
		return DraftSalesBrief{}, fmt.Errorf(
			"%w: sales brief exceeds %d missing fields",
			ErrInvalidDraft,
			MaximumDraftSalesMissingFields,
		)
	}

	previousMessages, err := NormalizeDraftConversation(
		request.PreviousMessages,
	)
	if err != nil {
		return DraftSalesBrief{}, err
	}

	customerMessages := make([]string, 0, len(previousMessages)+1)
	for _, message := range previousMessages {
		if message.Role == DraftMessageRoleCustomer {
			customerMessages = append(customerMessages, message.Body)
		}
	}
	if current := strings.TrimSpace(request.CustomerMessage); current != "" {
		customerMessages = append(customerMessages, current)
	}

	normalizedSignals := make(
		[]DraftSalesSignal,
		0,
		len(brief.Signals),
	)
	seenSignals := make(map[DraftSalesSignalKind]struct{}, len(brief.Signals))

	for _, signal := range brief.Signals {
		signal.Value = strings.TrimSpace(signal.Value)
		if !validDraftSalesSignalKind(signal.Kind) ||
			signal.Value == "" ||
			utf8.RuneCountInString(signal.Value) >
				MaximumDraftSalesSignalRunes {
			return DraftSalesBrief{}, fmt.Errorf(
				"%w: invalid sales brief signal",
				ErrInvalidDraft,
			)
		}
		if _, duplicate := seenSignals[signal.Kind]; duplicate {
			return DraftSalesBrief{}, fmt.Errorf(
				"%w: duplicate sales brief signal",
				ErrInvalidDraft,
			)
		}
		if !draftCustomerMessagesContain(
			customerMessages,
			signal.Value,
		) {
			return DraftSalesBrief{}, fmt.Errorf(
				"%w: sales brief signal is not customer-grounded",
				ErrInvalidDraft,
			)
		}

		seenSignals[signal.Kind] = struct{}{}
		normalizedSignals = append(normalizedSignals, signal)
	}

	normalizedMissing := make(
		[]DraftSalesMissingField,
		0,
		len(brief.MissingFields),
	)
	seenMissing := make(
		map[DraftSalesMissingField]struct{},
		len(brief.MissingFields),
	)
	for _, field := range brief.MissingFields {
		if !validDraftSalesMissingField(field) {
			return DraftSalesBrief{}, fmt.Errorf(
				"%w: invalid sales brief missing field",
				ErrInvalidDraft,
			)
		}
		if _, duplicate := seenMissing[field]; duplicate {
			return DraftSalesBrief{}, fmt.Errorf(
				"%w: duplicate sales brief missing field",
				ErrInvalidDraft,
			)
		}
		seenMissing[field] = struct{}{}
		normalizedMissing = append(normalizedMissing, field)
	}

	brief.NextQuestion = strings.TrimSpace(brief.NextQuestion)
	if utf8.RuneCountInString(brief.NextQuestion) >
		MaximumDraftNextQuestionRunes {
		return DraftSalesBrief{}, fmt.Errorf(
			"%w: sales brief next question is too long",
			ErrInvalidDraft,
		)
	}
	if len(normalizedMissing) > 0 && brief.NextQuestion == "" {
		return DraftSalesBrief{}, fmt.Errorf(
			"%w: sales brief next question is required",
			ErrInvalidDraft,
		)
	}
	if len(normalizedMissing) == 0 && brief.NextQuestion != "" {
		return DraftSalesBrief{}, fmt.Errorf(
			"%w: sales brief next question requires a missing field",
			ErrInvalidDraft,
		)
	}

	brief.Signals = normalizedSignals
	brief.MissingFields = normalizedMissing

	return brief, nil
}

func draftCustomerMessagesContain(
	messages []string,
	value string,
) bool {
	for _, message := range messages {
		if strings.Contains(message, value) {
			return true
		}
	}
	return false
}

func validDraftSalesSignalKind(kind DraftSalesSignalKind) bool {
	switch kind {
	case DraftSalesSignalEventType,
		DraftSalesSignalEventDate,
		DraftSalesSignalLocation,
		DraftSalesSignalGuestCount,
		DraftSalesSignalBudget:
		return true
	default:
		return false
	}
}

func validDraftSalesMissingField(
	field DraftSalesMissingField,
) bool {
	switch field {
	case DraftSalesMissingEventType,
		DraftSalesMissingEventDate,
		DraftSalesMissingLocation,
		DraftSalesMissingGuestCount,
		DraftSalesMissingBudget:
		return true
	default:
		return false
	}
}

func encodeDraftSalesBrief(brief DraftSalesBrief) (string, error) {
	if brief.Signals == nil {
		brief.Signals = []DraftSalesSignal{}
	}
	if brief.MissingFields == nil {
		brief.MissingFields = []DraftSalesMissingField{}
	}

	encoded, err := json.Marshal(brief)
	if err != nil {
		return "", fmt.Errorf(
			"encode web chat sales brief: %w",
			err,
		)
	}
	return string(encoded), nil
}
