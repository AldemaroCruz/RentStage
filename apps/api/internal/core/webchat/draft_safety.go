package webchat

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	draftCurrencyBeforeAmountPattern = regexp.MustCompile(
		`(?i)\b([A-Z]{3})\s*\$?\s*([0-9]+(?:[.,][0-9]{1,2})?)\b`,
	)
	draftCurrencyAfterAmountPattern = regexp.MustCompile(
		`(?i)\b([0-9]+(?:[.,][0-9]{1,2})?)\s*([A-Z]{3})\b`,
	)
	draftDollarAmountPattern = regexp.MustCompile(
		`\$\s*([0-9]+(?:[.,][0-9]{1,2})?)\b`,
	)
	draftDollarWordPattern = regexp.MustCompile(
		`(?i)\b([0-9]+(?:[.,][0-9]{1,2})?)\s*d[oó]lares?\b`,
	)
	draftGuestCountPattern = regexp.MustCompile(
		`(?i)\b([0-9]{1,7})\s*(?:personas|invitados|asistentes)\b`,
	)
	draftForbiddenCommercialClaimPatterns = []*regexp.Regexp{
		regexp.MustCompile(
			`\b(?:disponibilidad\s+(?:esta\s+)?(?:confirmada|garantizada)|` +
				`(?:esta|estan)\s+disponibles?|(?:tenemos|hay)\s+disponibilidad)\b`,
		),
		regexp.MustCompile(
			`\b(?:reserva(?:cion)?\s+(?:esta\s+)?confirmada|` +
				`(?:hemos|ya)\s+reserv(?:ado|amos|o)|` +
				`(?:quedo|queda|esta)\s+reservad[oa])\b`,
		),
		regexp.MustCompile(
			`\b(?:descuento\s+(?:esta\s+)?` +
				`(?:aprobado|confirmado|garantizado)|` +
				`(?:hemos|te)\s+aplicamos\s+un\s+descuento)\b`,
		),
		regexp.MustCompile(
			`\bpago\s+(?:fue\s+|esta\s+)?(?:recibido|confirmado)\b`,
		),
		regexp.MustCompile(
			`\b(?:cotizacion|orden)\s+(?:fue\s+|esta\s+)?` +
				`(?:creada|confirmada|aprobada)\b`,
		),
	}
)

type draftMoneyClaim struct {
	Currency string
	Cents    int64
}

// ValidateDraftCommercialClaims applies deterministic checks to the free-text
// portion of a provider draft. Structured grounding remains the source of
// truth; these checks are defense in depth before mandatory human approval.
func ValidateDraftCommercialClaims(
	result DraftResult,
	request DraftRequest,
) error {
	body := foldDraftSafetyText(result.Body)
	for _, pattern := range draftForbiddenCommercialClaimPatterns {
		if pattern.MatchString(body) {
			return fmt.Errorf(
				"%w: draft contains an unverified commercial action",
				ErrInvalidDraft,
			)
		}
	}

	amounts := draftMoneyClaims(
		result.Body,
		request.SalesContext.Currency,
	)
	guestCounts := draftGuestCounts(result.Body)
	if len(amounts) == 0 && len(guestCounts) == 0 {
		return nil
	}

	context, err := NormalizeDraftSalesContext(request.SalesContext)
	if err != nil {
		return err
	}

	allowedPrices := draftReferencedPrices(
		result.GroundingReferences,
		context,
	)
	for _, amount := range amounts {
		_, allowed := allowedPrices[amount.Cents]
		if amount.Currency != context.Currency || !allowed {
			return fmt.Errorf(
				"%w: draft contains an ungrounded monetary claim",
				ErrInvalidDraft,
			)
		}
	}

	allowedGuestCounts := draftAllowedGuestCounts(
		request,
		result.GroundingReferences,
		context,
	)
	for _, count := range guestCounts {
		if _, allowed := allowedGuestCounts[count]; !allowed {
			return fmt.Errorf(
				"%w: draft contains an ungrounded guest capacity",
				ErrInvalidDraft,
			)
		}
	}

	return nil
}

func foldDraftSafetyText(value string) string {
	value = strings.ToLower(value)
	return strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
	).Replace(value)
}

func draftReferencedPrices(
	references []DraftGroundingReference,
	context DraftSalesContext,
) map[int64]struct{} {
	allowed := make(map[int64]struct{})
	if !context.ShowPrices {
		return allowed
	}

	for _, reference := range references {
		switch reference.Kind {
		case DraftGroundingKindPackage:
			for _, item := range context.Packages {
				if strings.EqualFold(item.Name, reference.Name) &&
					item.Price != nil {
					allowed[draftPriceCents(*item.Price)] = struct{}{}
				}
			}
		case DraftGroundingKindResource:
			for _, item := range context.Resources {
				if strings.EqualFold(item.Name, reference.Name) &&
					item.Price != nil {
					allowed[draftPriceCents(*item.Price)] = struct{}{}
				}
			}
		}
	}

	return allowed
}

func draftMoneyClaims(
	value string,
	expectedCurrency string,
) []draftMoneyClaim {
	claims := make(map[string]draftMoneyClaim)
	expectedCurrency = strings.ToUpper(
		strings.TrimSpace(expectedCurrency),
	)
	addClaim := func(currency string, rawAmount string) {
		originalCurrency := strings.TrimSpace(currency)
		if expectedCurrency != "" &&
			strings.EqualFold(originalCurrency, expectedCurrency) {
			originalCurrency = expectedCurrency
		} else if originalCurrency != strings.ToUpper(originalCurrency) {
			return
		}

		parsed, err := strconv.ParseFloat(
			strings.ReplaceAll(rawAmount, ",", "."),
			64,
		)
		if err != nil || parsed < 0 ||
			math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return
		}
		claim := draftMoneyClaim{
			Currency: strings.ToUpper(originalCurrency),
			Cents:    draftPriceCents(parsed),
		}
		key := claim.Currency + "\x00" + strconv.FormatInt(claim.Cents, 10)
		claims[key] = claim
	}

	for _, match := range draftCurrencyBeforeAmountPattern.FindAllStringSubmatch(
		value,
		-1,
	) {
		addClaim(match[1], match[2])
	}
	for _, match := range draftCurrencyAfterAmountPattern.FindAllStringSubmatch(
		value,
		-1,
	) {
		addClaim(match[2], match[1])
	}
	for _, match := range draftDollarAmountPattern.FindAllStringSubmatch(
		value,
		-1,
	) {
		addClaim("USD", match[1])
	}
	for _, match := range draftDollarWordPattern.FindAllStringSubmatch(
		value,
		-1,
	) {
		addClaim("USD", match[1])
	}

	result := make([]draftMoneyClaim, 0, len(claims))
	for _, claim := range claims {
		result = append(result, claim)
	}
	return result
}

func draftPriceCents(value float64) int64 {
	return int64(math.Round(value * 100))
}

func draftAllowedGuestCounts(
	request DraftRequest,
	references []DraftGroundingReference,
	context DraftSalesContext,
) map[int64]struct{} {
	allowed := make(map[int64]struct{})
	for _, message := range request.PreviousMessages {
		if message.Role == DraftMessageRoleCustomer {
			for _, count := range draftGuestCounts(message.Body) {
				allowed[count] = struct{}{}
			}
		}
	}
	for _, count := range draftGuestCounts(request.CustomerMessage) {
		allowed[count] = struct{}{}
	}

	for _, reference := range references {
		if reference.Kind != DraftGroundingKindPackage {
			continue
		}
		for _, item := range context.Packages {
			if !strings.EqualFold(item.Name, reference.Name) {
				continue
			}
			if item.GuestCapacity != nil {
				allowed[int64(*item.GuestCapacity)] = struct{}{}
			}
			for _, count := range draftGuestCounts(
				item.Name + " " + item.Description,
			) {
				allowed[count] = struct{}{}
			}
		}
	}

	return allowed
}

func draftGuestCounts(value string) []int64 {
	matches := draftGuestCountPattern.FindAllStringSubmatch(value, -1)
	result := make([]int64, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		count, err := strconv.ParseInt(match[1], 10, 64)
		if err == nil {
			result = append(result, count)
		}
	}
	return result
}
