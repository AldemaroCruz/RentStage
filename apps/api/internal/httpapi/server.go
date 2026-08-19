package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/authn"
	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/assistant"
	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
	"github.com/rentstage/rentstage/apps/api/internal/core/billing"
	"github.com/rentstage/rentstage/apps/api/internal/core/catalog"
	"github.com/rentstage/rentstage/apps/api/internal/core/customer"
	"github.com/rentstage/rentstage/apps/api/internal/core/dashboard"
	"github.com/rentstage/rentstage/apps/api/internal/core/dte"
	"github.com/rentstage/rentstage/apps/api/internal/core/identity"
	"github.com/rentstage/rentstage/apps/api/internal/core/inventory"
	"github.com/rentstage/rentstage/apps/api/internal/core/operations"
	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
	"github.com/rentstage/rentstage/apps/api/internal/core/publiccatalog"
	"github.com/rentstage/rentstage/apps/api/internal/core/quote"
	"github.com/rentstage/rentstage/apps/api/internal/core/quoteportal"
	"github.com/rentstage/rentstage/apps/api/internal/core/reservation"
	"github.com/rentstage/rentstage/apps/api/internal/core/tenant"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Server struct {
	httpServer *http.Server
}

func New(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*Server, error) {
	auditRepository := audit.NewRepository(pool)
	identityRepository := identity.NewRepository(pool)
	identityService := identity.NewService(identityRepository, auditRepository, cfg.WebBaseURL)
	authService, err := authn.New(ctx, cfg, identityRepository)
	if err != nil {
		return nil, fmt.Errorf("initialize authentication: %w", err)
	}
	if err := authService.BootstrapLocalOwner(ctx); err != nil {
		return nil, fmt.Errorf("bootstrap local owner: %w", err)
	}

	catalogRepository := catalog.NewRepository(pool)
	catalogService := catalog.NewService(catalogRepository, auditRepository)
	inventoryRepository := inventory.NewRepository(pool)
	inventoryService := inventory.NewService(inventoryRepository, auditRepository)
	customerRepository := customer.NewRepository(pool)
	customerService := customer.NewService(customerRepository, auditRepository)
	quoteRepository := quote.NewRepository(pool)
	quoteService := quote.NewService(quoteRepository, auditRepository)
	availabilityRepository := availability.NewRepository(pool)
	availabilityService := availability.NewService(availabilityRepository)
	packageRepository := packages.NewRepository(pool)
	packageService := packages.NewService(packageRepository, availabilityService, auditRepository)
	assistantRepository := assistant.NewRepository(pool)
	assistantService := assistant.NewService(
		assistantRepository, packageRepository, packageService,
		customerRepository, quoteService, auditRepository,
	)
	publicCatalogRepository := publiccatalog.NewRepository(pool)
	publicCatalogService := publiccatalog.NewService(
		publicCatalogRepository, packageService, availabilityService, auditRepository,
		cfg.WebBaseURL, cfg.PublicRequestFingerprintSalt,
	)
	reservationRepository := reservation.NewRepository(pool)
	reservationService := reservation.NewService(reservationRepository, auditRepository)
	quotePortalRepository := quoteportal.NewRepository(pool, reservationRepository)
	quotePortalService := quoteportal.NewService(
		quotePortalRepository, quoteRepository, auditRepository,
		cfg.WebBaseURL, cfg.PublicRequestFingerprintSalt,
	)
	operationsRepository := operations.NewRepository(pool)
	billingRepository := billing.NewRepository(pool)
	billingService := billing.NewService(billingRepository, auditRepository)
	dteRepository := dte.NewRepository(pool)
	dteService := dte.NewService(dteRepository, auditRepository)

	authHandler := authn.NewHandler(authService, cfg)
	identityHandler := identity.NewHandler(identityRepository, identityService)
	tenantHandler := tenant.NewHandler(pool)
	dashboardHandler := dashboard.NewHandler(pool)
	catalogHandler := catalog.NewHandler(catalogRepository, catalogService)
	inventoryHandler := inventory.NewHandler(inventoryRepository, inventoryService)
	availabilityHandler := availability.NewHandler(availabilityService)
	packageHandler := packages.NewHandler(packageRepository, packageService)
	assistantHandler := assistant.NewHandler(assistantRepository, assistantService)
	publicCatalogHandler := publiccatalog.NewHandler(publicCatalogService)
	auditHandler := audit.NewHandler(auditRepository)
	customerHandler := customer.NewHandler(customerRepository, customerService)
	quoteHandler := quote.NewHandler(quoteRepository, quoteService)
	quotePortalHandler := quoteportal.NewHandler(quotePortalService)
	reservationHandler := reservation.NewHandler(reservationRepository, reservationService)
	operationsHandler := operations.NewHandler(operationsRepository)
	billingHandler := billing.NewHandler(billingRepository, billingService)
	dteHandler := dte.NewHandler(dteRepository, dteService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		webutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			webutil.WriteError(w, r, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable.")
			return
		}
		webutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.HandleFunc("GET /api/v1/auth/csrf", authHandler.CSRF)
	mux.HandleFunc("POST /api/v1/auth/session", authHandler.Session)
	mux.HandleFunc("DELETE /api/v1/auth/session", authHandler.Logout)
	mux.HandleFunc("GET /api/v1/invitations/{token}", identityHandler.InvitationPreview)

	// Anonymous public catalog routes. Unsafe requests still pass the global
	// CSRF middleware, and the service validates publication, rate limits, and
	// tenant boundaries independently of authentication cookies.
	mux.HandleFunc("GET /api/v1/public/catalogs/{tenantSlug}", publicCatalogHandler.PublicCatalog)
	mux.HandleFunc("GET /api/v1/public/catalogs/{tenantSlug}/packages/{packageSlug}", publicCatalogHandler.PublicPackage)
	mux.HandleFunc("GET /api/v1/public/catalogs/{tenantSlug}/resources/{resourceSlug}", publicCatalogHandler.PublicResource)
	mux.HandleFunc("POST /api/v1/public/catalogs/{tenantSlug}/availability", publicCatalogHandler.PublicAvailability)
	mux.HandleFunc("POST /api/v1/public/catalogs/{tenantSlug}/quote-requests", publicCatalogHandler.SubmitQuoteRequest)
	mux.HandleFunc("GET /api/v1/public/quote-portal", quotePortalHandler.PublicView)
	mux.HandleFunc("POST /api/v1/public/quote-portal/accept", quotePortalHandler.Accept)
	mux.HandleFunc("POST /api/v1/public/quote-portal/reject", quotePortalHandler.Reject)

	authenticated := authenticationMiddleware(authService, cfg)
	tenantContext := tenantMiddleware(identityRepository, cfg)

	registerAuth := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, authenticated(handler))
	}
	registerTenant := func(pattern string, permission identity.Permission, handler http.HandlerFunc) {
		mux.Handle(pattern, chain(handler, authenticated, tenantContext, permissionMiddleware(permission)))
	}

	registerAuth("GET /api/v1/auth/me", authHandler.Me)
	registerAuth("POST /api/v1/auth/select-tenant", authHandler.SelectTenant)
	registerAuth("POST /api/v1/organizations", identityHandler.CreateOrganization)
	registerAuth("POST /api/v1/invitations/{token}/accept", identityHandler.AcceptInvitation)

	registerTenant("GET /api/v1/tenant", identity.PermissionTenantRead, tenantHandler.Get)
	registerTenant("PATCH /api/v1/tenant", identity.PermissionTenantManage, identityHandler.UpdateOrganization)
	registerTenant("GET /api/v1/team", identity.PermissionTeamManage, identityHandler.ListTeam)
	registerTenant("POST /api/v1/team/invitations", identity.PermissionTeamManage, identityHandler.CreateInvitation)
	registerTenant("DELETE /api/v1/team/invitations/{invitationID}", identity.PermissionTeamManage, identityHandler.RevokeInvitation)
	registerTenant("PATCH /api/v1/team/members/{userID}", identity.PermissionTeamManage, identityHandler.UpdateMember)

	registerTenant("GET /api/v1/dashboard", identity.PermissionOperationsRead, dashboardHandler.Get)
	registerTenant("GET /api/v1/assistant/conversations", identity.PermissionAssistantRead, assistantHandler.List)
	registerTenant("POST /api/v1/assistant/conversations/simulate", identity.PermissionAssistantManage, assistantHandler.Simulate)
	registerTenant("GET /api/v1/assistant/conversations/{conversationID}", identity.PermissionAssistantRead, assistantHandler.Get)
	registerTenant("POST /api/v1/assistant/conversations/{conversationID}/approve", identity.PermissionAssistantManage, assistantHandler.Approve)
	registerTenant("POST /api/v1/assistant/conversations/{conversationID}/customer", identity.PermissionAssistantManage, assistantHandler.LinkCustomer)
	registerTenant("POST /api/v1/assistant/conversations/{conversationID}/messages/send-demo", identity.PermissionAssistantManage, assistantHandler.SendDemo)
	registerTenant("POST /api/v1/assistant/conversations/{conversationID}/messages/receive-demo", identity.PermissionAssistantManage, assistantHandler.ReceiveDemo)

	registerTenant("GET /api/v1/categories", identity.PermissionCatalogRead, catalogHandler.ListCategories)
	registerTenant("POST /api/v1/categories", identity.PermissionCatalogManage, catalogHandler.CreateCategory)
	registerTenant("DELETE /api/v1/categories/{categoryID}", identity.PermissionCatalogManage, catalogHandler.DeleteCategory)

	registerTenant("GET /api/v1/resources", identity.PermissionCatalogRead, catalogHandler.ListResources)
	registerTenant("POST /api/v1/resources", identity.PermissionCatalogManage, catalogHandler.CreateResource)
	registerTenant("GET /api/v1/resources/{resourceID}", identity.PermissionCatalogRead, catalogHandler.GetResource)
	registerTenant("PATCH /api/v1/resources/{resourceID}", identity.PermissionCatalogManage, catalogHandler.UpdateResource)
	registerTenant("DELETE /api/v1/resources/{resourceID}", identity.PermissionCatalogManage, catalogHandler.ArchiveResource)
	registerTenant("GET /api/v1/resources/{resourceID}/assets", identity.PermissionInventoryRead, inventoryHandler.ListAssets)
	registerTenant("POST /api/v1/resources/{resourceID}/assets", identity.PermissionInventoryManage, inventoryHandler.CreateAsset)
	registerTenant("GET /api/v1/resources/{resourceID}/availability", identity.PermissionCatalogRead, availabilityHandler.Get)
	registerTenant("POST /api/v1/availability/check", identity.PermissionCatalogRead, availabilityHandler.Check)

	registerTenant("GET /api/v1/packages", identity.PermissionPackageRead, packageHandler.List)
	registerTenant("POST /api/v1/packages", identity.PermissionPackageManage, packageHandler.Create)
	registerTenant("GET /api/v1/packages/{packageID}", identity.PermissionPackageRead, packageHandler.Get)
	registerTenant("PATCH /api/v1/packages/{packageID}", identity.PermissionPackageManage, packageHandler.Update)
	registerTenant("DELETE /api/v1/packages/{packageID}", identity.PermissionPackageManage, packageHandler.Archive)
	registerTenant("GET /api/v1/packages/{packageID}/quote-template", identity.PermissionPackageRead, packageHandler.QuoteTemplate)
	registerTenant("POST /api/v1/packages/{packageID}/availability", identity.PermissionPackageRead, packageHandler.Availability)

	registerTenant("GET /api/v1/public-catalog", identity.PermissionPublicCatalogRead, publicCatalogHandler.AdminCatalog)
	registerTenant("PATCH /api/v1/public-catalog", identity.PermissionPublicCatalogManage, publicCatalogHandler.UpdateSettings)
	registerTenant("PATCH /api/v1/public-catalog/packages/{packageID}", identity.PermissionPublicCatalogManage, publicCatalogHandler.UpdatePackagePublication)
	registerTenant("PATCH /api/v1/public-catalog/resources/{resourceID}", identity.PermissionPublicCatalogManage, publicCatalogHandler.UpdateResourcePublication)
	registerTenant("GET /api/v1/quote-requests", identity.PermissionQuoteRequestRead, publicCatalogHandler.ListQuoteRequests)
	registerTenant("GET /api/v1/quote-requests/{requestID}", identity.PermissionQuoteRequestRead, publicCatalogHandler.GetQuoteRequest)
	registerTenant("PATCH /api/v1/quote-requests/{requestID}", identity.PermissionQuoteRequestManage, publicCatalogHandler.UpdateQuoteRequestStatus)
	registerTenant("POST /api/v1/quote-requests/{requestID}/convert", identity.PermissionQuoteRequestManage, publicCatalogHandler.ConvertQuoteRequest)

	registerTenant("PATCH /api/v1/assets/{assetID}", identity.PermissionInventoryManage, inventoryHandler.UpdateAsset)
	registerTenant("DELETE /api/v1/assets/{assetID}", identity.PermissionInventoryManage, inventoryHandler.RetireAsset)

	registerTenant("GET /api/v1/customers", identity.PermissionCustomerRead, customerHandler.List)
	registerTenant("POST /api/v1/customers", identity.PermissionCustomerManage, customerHandler.Create)
	registerTenant("GET /api/v1/customers/{customerID}", identity.PermissionCustomerRead, customerHandler.Get)
	registerTenant("PATCH /api/v1/customers/{customerID}", identity.PermissionCustomerManage, customerHandler.Update)

	registerTenant("GET /api/v1/quote-portal-settings", identity.PermissionQuoteRead, quotePortalHandler.Settings)
	registerTenant("PATCH /api/v1/quote-portal-settings", identity.PermissionQuoteManage, quotePortalHandler.UpdateSettings)

	registerTenant("GET /api/v1/quotes", identity.PermissionQuoteRead, quoteHandler.List)
	registerTenant("POST /api/v1/quotes", identity.PermissionQuoteManage, quoteHandler.Create)
	registerTenant("GET /api/v1/quotes/{quoteID}", identity.PermissionQuoteRead, quoteHandler.Get)
	registerTenant("PATCH /api/v1/quotes/{quoteID}", identity.PermissionQuoteManage, quoteHandler.Update)
	registerTenant("POST /api/v1/quotes/{quoteID}/send", identity.PermissionQuoteManage, quotePortalHandler.Send)
	registerTenant("POST /api/v1/quotes/{quoteID}/portal/reissue", identity.PermissionQuoteManage, quotePortalHandler.Reissue)
	registerTenant("DELETE /api/v1/quotes/{quoteID}/portal", identity.PermissionQuoteManage, quotePortalHandler.Revoke)
	registerTenant("POST /api/v1/quotes/{quoteID}/accept", identity.PermissionQuoteManage, quoteHandler.Accept)
	registerTenant("POST /api/v1/quotes/{quoteID}/reject", identity.PermissionQuoteManage, quoteHandler.Reject)
	registerTenant("POST /api/v1/quotes/{quoteID}/cancel", identity.PermissionQuoteManage, quoteHandler.Cancel)
	registerTenant("POST /api/v1/quotes/{quoteID}/convert-to-reservation", identity.PermissionReservationManage, reservationHandler.ConvertQuote)

	registerTenant("GET /api/v1/billing/settings", identity.PermissionBillingRead, billingHandler.Settings)
	registerTenant("PATCH /api/v1/billing/settings", identity.PermissionBillingManage, billingHandler.UpdateSettings)
	registerTenant("GET /api/v1/billing/tax-rules", identity.PermissionBillingRead, billingHandler.TaxRules)
	registerTenant("GET /api/v1/billing/dashboard", identity.PermissionBillingRead, billingHandler.Dashboard)

	registerTenant("GET /api/v1/invoices", identity.PermissionBillingRead, billingHandler.ListInvoices)
	registerTenant("POST /api/v1/invoices", identity.PermissionBillingManage, billingHandler.CreateInvoice)
	registerTenant("GET /api/v1/invoices/{invoiceID}", identity.PermissionBillingRead, billingHandler.GetInvoice)
	registerTenant("PATCH /api/v1/invoices/{invoiceID}", identity.PermissionBillingManage, billingHandler.UpdateInvoice)
	registerTenant("POST /api/v1/invoices/{invoiceID}/issue", identity.PermissionBillingManage, billingHandler.IssueInvoice)
	registerTenant("POST /api/v1/invoices/{invoiceID}/void", identity.PermissionBillingManage, billingHandler.VoidInvoice)

	registerTenant("GET /api/v1/dte-settings", identity.PermissionFiscalRead, dteHandler.Settings)
	registerTenant("PATCH /api/v1/dte-settings", identity.PermissionFiscalManage, dteHandler.UpdateSettings)
	registerTenant("GET /api/v1/dte", identity.PermissionFiscalRead, dteHandler.List)
	registerTenant("GET /api/v1/dte/{documentID}", identity.PermissionFiscalRead, dteHandler.Get)
	registerTenant("GET /api/v1/invoices/{invoiceID}/dte", identity.PermissionFiscalRead, dteHandler.GetByInvoice)
	registerTenant("POST /api/v1/invoices/{invoiceID}/dte", identity.PermissionFiscalManage, dteHandler.Prepare)
	registerTenant("POST /api/v1/dte/{documentID}/submit", identity.PermissionFiscalManage, dteHandler.Submit)
	registerTenant("POST /api/v1/dte/{documentID}/retry", identity.PermissionFiscalManage, dteHandler.Submit)
	registerTenant("POST /api/v1/dte/{documentID}/cancel", identity.PermissionFiscalManage, dteHandler.Cancel)
	registerTenant("POST /api/v1/dte/{documentID}/invalidate", identity.PermissionFiscalManage, dteHandler.Invalidate)

	registerTenant("GET /api/v1/payments", identity.PermissionPaymentRead, billingHandler.ListPayments)
	registerTenant("POST /api/v1/payments", identity.PermissionPaymentManage, billingHandler.CreatePayment)
	registerTenant("GET /api/v1/payments/{paymentID}", identity.PermissionPaymentRead, billingHandler.GetPayment)
	registerTenant("POST /api/v1/payments/{paymentID}/void", identity.PermissionPaymentManage, billingHandler.VoidPayment)

	registerTenant("GET /api/v1/security-deposits", identity.PermissionPaymentRead, billingHandler.ListDeposits)
	registerTenant("POST /api/v1/security-deposits", identity.PermissionPaymentManage, billingHandler.CreateDeposit)
	registerTenant("GET /api/v1/security-deposits/{depositID}", identity.PermissionPaymentRead, billingHandler.GetDeposit)
	registerTenant("POST /api/v1/security-deposits/{depositID}/receive", identity.PermissionPaymentManage, billingHandler.ReceiveDeposit)
	registerTenant("POST /api/v1/security-deposits/{depositID}/settle", identity.PermissionPaymentManage, billingHandler.SettleDeposit)

	registerTenant("GET /api/v1/reservations", identity.PermissionReservationRead, reservationHandler.List)
	registerTenant("POST /api/v1/reservations", identity.PermissionReservationManage, reservationHandler.Create)
	registerTenant("GET /api/v1/reservations/{reservationID}", identity.PermissionReservationRead, reservationHandler.Get)
	registerTenant("POST /api/v1/reservations/{reservationID}/reschedule", identity.PermissionReservationManage, reservationHandler.Reschedule)
	registerTenant("GET /api/v1/reservations/{reservationID}/warehouse", identity.PermissionReservationRead, reservationHandler.Warehouse)
	registerTenant("POST /api/v1/reservations/{reservationID}/assets", identity.PermissionWarehouseOperate, reservationHandler.AssignAsset)
	registerTenant("DELETE /api/v1/reservations/{reservationID}/assets/{assetID}", identity.PermissionWarehouseOperate, reservationHandler.UnassignAsset)
	registerTenant("POST /api/v1/reservations/{reservationID}/confirm", identity.PermissionReservationManage, reservationHandler.Confirm)
	registerTenant("POST /api/v1/reservations/{reservationID}/prepare", identity.PermissionWarehouseOperate, reservationHandler.Prepare)
	registerTenant("POST /api/v1/reservations/{reservationID}/mark-ready", identity.PermissionWarehouseOperate, reservationHandler.MarkReady)
	registerTenant("POST /api/v1/reservations/{reservationID}/checkout", identity.PermissionWarehouseOperate, reservationHandler.CheckOut)
	registerTenant("POST /api/v1/reservations/{reservationID}/return", identity.PermissionWarehouseOperate, reservationHandler.Return)
	registerTenant("POST /api/v1/reservations/{reservationID}/complete", identity.PermissionWarehouseOperate, reservationHandler.Complete)
	registerTenant("POST /api/v1/reservations/{reservationID}/cancel", identity.PermissionReservationManage, reservationHandler.Cancel)

	registerTenant("GET /api/v1/calendar", identity.PermissionOperationsRead, operationsHandler.Calendar)
	registerTenant("GET /api/v1/operations/agenda", identity.PermissionOperationsRead, operationsHandler.Agenda)
	registerTenant("GET /api/v1/operations/alerts", identity.PermissionOperationsRead, operationsHandler.Alerts)

	registerTenant("GET /api/v1/audit", identity.PermissionAuditRead, auditHandler.List)

	handler := chain(
		mux,
		requestIDMiddleware,
		corsMiddleware(cfg.CORSAllowedOrigins),
		csrfMiddleware(cfg),
		loggingMiddleware(logger),
		recoveryMiddleware(logger),
	)

	return &Server{httpServer: &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}}, nil
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
