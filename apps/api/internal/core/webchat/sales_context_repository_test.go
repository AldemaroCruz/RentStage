package webchat

import (
	"strings"
	"testing"
)

func TestDraftSalesSettingsQueryIsTenantScoped(t *testing.T) {
	requiredClauses := []string{
		"tenant.id = $1",
		"tenant.status = 'ACTIVE'",
		"settings.enabled = TRUE",
		"settings.web_chat_enabled = TRUE",
		"settings.show_prices",
		"settings.show_resources",
		"settings.quote_requests_enabled",
	}

	assertDraftSalesQueryClauses(
		t,
		draftSalesSettingsQuery,
		requiredClauses,
	)
}

func TestDraftSalesPackagesQueryUsesOnlyPublishedCatalog(
	t *testing.T,
) {
	requiredClauses := []string{
		"pkg.tenant_id = $1",
		"pkg.active = TRUE",
		"pkg.public_visible = TRUE",
		"item_count > 0",
		"inactive_item_count = 0",
		"WHEN $2::boolean = FALSE THEN NULL",
		"LIMIT $4",
	}

	assertDraftSalesQueryClauses(
		t,
		draftSalesPackagesQuery,
		requiredClauses,
	)
	assertDraftSalesQueryExcludesPrivateTables(
		t,
		draftSalesPackagesQuery,
	)
}

func TestDraftSalesResourcesQueryUsesOnlyPublishedCatalog(
	t *testing.T,
) {
	requiredClauses := []string{
		"resource.tenant_id = $1",
		"resource.active = TRUE",
		"resource.public_visible = TRUE",
		"resource.public_slug IS NOT NULL",
		"WHEN $2::boolean THEN resource.base_price::float8",
		"LIMIT $6",
	}

	assertDraftSalesQueryClauses(
		t,
		draftSalesResourcesQuery,
		requiredClauses,
	)
	assertDraftSalesQueryExcludesPrivateTables(
		t,
		draftSalesResourcesQuery,
	)
}

func assertDraftSalesQueryClauses(
	t *testing.T,
	query string,
	clauses []string,
) {
	t.Helper()
	for _, clause := range clauses {
		if !strings.Contains(query, clause) {
			t.Fatalf("sales query is missing %q", clause)
		}
	}
}

func assertDraftSalesQueryExcludesPrivateTables(
	t *testing.T,
	query string,
) {
	t.Helper()
	for _, table := range []string{
		"assets",
		"reservations",
		"reservation_items",
		"customers",
		"quotes",
		"quote_items",
		"assistant_messages",
	} {
		if strings.Contains(query, table) {
			t.Fatalf("sales query exposes private table %q", table)
		}
	}
}
