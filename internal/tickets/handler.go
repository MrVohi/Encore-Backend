package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v84"
	checkoutsession "github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"

	"groupie-tracker/internal/authentification/database"
	"groupie-tracker/internal/notifications"
)

type Handler struct {
	repo          *Repository
	frontendURL   string
	webhookSecret string
	mailer        notifications.EmailSender
}

func NewHandler(repo *Repository, frontendURL string, webhookSecret string, mailer notifications.EmailSender) *Handler {
	return &Handler{
		repo:          repo,
		frontendURL:   strings.TrimSpace(frontendURL),
		webhookSecret: strings.TrimSpace(webhookSecret),
		mailer:        mailer,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/concerts/upcoming", h.listUpcoming)
	rg.GET("/concerts/:id/ticket-types", h.listTicketTypes)
}

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/tickets/checkout", h.createCheckoutSession)
	rg.GET("/tickets", h.listUserTickets)
}

func (h *Handler) RegisterWebhookRoutes(rg *gin.RouterGroup) {
	rg.POST("/stripe/webhook", h.handleWebhook)
}

func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/concerts/:id/ticket-types", h.createTicketType)
	rg.PATCH("/ticket-types/:id", h.updateTicketType)
	rg.DELETE("/ticket-types/:id", h.deleteTicketType)
	rg.GET("/tickets/admin", h.listAdminTickets)
	rg.POST("/tickets/admin", h.createAdminTicket)
	rg.PATCH("/tickets/admin/:id", h.updateAdminTicket)
	rg.DELETE("/tickets/admin/:id", h.deleteAdminTicket)
	rg.GET("/tickets/admin/stats", h.ticketStats)
}

func (h *Handler) listUpcoming(c *gin.Context) {
	concerts, err := h.repo.ListUpcoming(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concerts"})
		return
	}
	c.JSON(http.StatusOK, concerts)
}

func (h *Handler) listTicketTypes(c *gin.Context) {
	concertID := strings.TrimSpace(c.Param("id"))
	if concertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert id is required"})
		return
	}

	types, err := h.repo.ListTicketTypes(c.Request.Context(), concertID)
	if err != nil {
		log.Printf("ticket types list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket types"})
		return
	}
	c.JSON(http.StatusOK, types)
}

func (h *Handler) listUserTickets(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	resp, err := h.repo.ListUserTickets(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) createCheckoutSession(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.ConcertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert_id is required"})
		return
	}
	if req.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be >= 1"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	userEmail, ok := c.Get("user_email")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user email"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	userEmailStr, ok := userEmail.(string)
	if !ok || userEmailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user email"})
		return
	}

	info, err := h.repo.GetCheckoutInfo(c.Request.Context(), req.ConcertID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "concert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concert"})
		return
	}

	if req.Quantity > info.Available {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough tickets available"})
		return
	}

	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stripe is not configured"})
		return
	}

	productName := info.Title
	if info.City != "" {
		productName = fmt.Sprintf("%s - %s", info.Title, info.City)
	}

	successURL := joinURL(h.frontendURL, "/tickets?success=1")
	cancelURL := joinURL(h.frontendURL, "/tickets?canceled=1")

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(int64(req.Quantity)),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(strings.ToLower(info.Currency)),
					UnitAmount: stripe.Int64(info.PriceCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(productName),
					},
				},
			},
		},
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		CustomerEmail:     stripe.String(userEmailStr),
		ClientReferenceID: stripe.String(userIDStr),
		Metadata: map[string]string{
			"concert_id":     info.ConcertID,
			"ticket_type_id": info.TicketTypeID,
			"ticket_type":    info.TicketType,
			"user_id":        userIDStr,
			"quantity":       strconv.Itoa(req.Quantity),
		},
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}
	if session.URL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stripe checkout url missing"})
		return
	}

	c.JSON(http.StatusOK, CheckoutResponse{
		URL:       session.URL,
		SessionID: session.ID,
	})
}

func (h *Handler) handleWebhook(c *gin.Context) {
	if h.webhookSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stripe webhook secret not configured"})
		return
	}

	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	sig := c.GetHeader("Stripe-Signature")
	if sig == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing stripe signature"})
		return
	}

	event, err := webhook.ConstructEvent(payload, sig, h.webhookSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	if event.Type != "checkout.session.completed" {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session payload"})
		return
	}

	meta := session.Metadata
	cartIDsRaw := strings.TrimSpace(meta["cart_item_ids"])
	if cartIDsRaw != "" {
		userID := strings.TrimSpace(meta["user_id"])
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing checkout metadata"})
			return
		}

		itemIDs := splitCSV(cartIDsRaw)
		if len(itemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing checkout metadata"})
			return
		}

		paymentIntentID := ""
		if session.PaymentIntent != nil {
			paymentIntentID = session.PaymentIntent.ID
		}

		created, items, err := h.repo.FulfillCartCheckout(c.Request.Context(), userID, session.ID, paymentIntentID, itemIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fulfill checkout"})
			return
		}

		if created {
			for _, item := range items {
				req := FulfillRequest{
					UserID:          userID,
					ConcertID:       item.ConcertID,
					TicketTypeID:    item.TicketTypeID,
					TicketType:      item.TicketType,
					SessionID:       session.ID,
					Quantity:        item.Quantity,
					PaymentIntentID: paymentIntentID,
				}
				if err := h.sendTicketEmail(c.Request.Context(), req, session); err != nil {
					log.Printf("ticket email send failed: %v", err)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	req := FulfillRequest{
		UserID:       meta["user_id"],
		ConcertID:    meta["concert_id"],
		TicketTypeID: meta["ticket_type_id"],
		TicketType:   meta["ticket_type"],
		SessionID:    session.ID,
		Quantity:     0,
	}

	if qty, err := strconv.Atoi(meta["quantity"]); err == nil {
		req.Quantity = qty
	}
	if session.PaymentIntent != nil {
		req.PaymentIntentID = session.PaymentIntent.ID
	}

	if req.UserID == "" || req.TicketTypeID == "" || req.SessionID == "" || req.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing checkout metadata"})
		return
	}

	created, err := h.repo.FulfillCheckout(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fulfill checkout"})
		return
	}

	if created {
		if err := h.sendTicketEmail(c.Request.Context(), req, session); err != nil {
			log.Printf("ticket email send failed: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *Handler) sendTicketEmail(ctx context.Context, req FulfillRequest, session stripe.CheckoutSession) error {
	if h.mailer == nil {
		return nil
	}

	email := strings.TrimSpace(session.CustomerEmail)
	if email == "" && session.CustomerDetails != nil {
		email = strings.TrimSpace(session.CustomerDetails.Email)
	}
	if email == "" {
		email = lookupUserEmail(req.UserID)
	}
	if email == "" {
		return fmt.Errorf("missing buyer email")
	}

	info, err := h.repo.GetConcertEmailInfo(ctx, req.ConcertID)
	if err != nil {
		return err
	}

	ticketType := req.TicketType
	if ticketType == "" {
		ticketType = req.TicketTypeID
	}
	qrPayload := fmt.Sprintf("encore:ticket:%s:%s:%s", req.SessionID, req.UserID, ticketType)
	subject, textBody, htmlBody, err := buildTicketEmail(TicketEmailInfo{
		ConcertTitle: info.ConcertTitle,
		When:         info.When,
		City:         info.City,
		Country:      info.Country,
	}, req.Quantity, qrPayload)
	if err != nil {
		return err
	}

	if err := h.mailer.Send(email, subject, textBody, htmlBody); err != nil {
		return err
	}

	log.Printf("notifications: ticket email sent to %s", email)
	return nil
}

func splitCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func lookupUserEmail(userID string) string {
	if userID == "" {
		return ""
	}
	db := database.GetDB()
	if db == nil {
		return ""
	}
	var email string
	if err := db.Raw("SELECT email FROM users WHERE id = ?", userID).Scan(&email).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(email)
}

func (h *Handler) createTicketType(c *gin.Context) {
	concertID := strings.TrimSpace(c.Param("id"))
	if concertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "concert id is required"})
		return
	}

	var req CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.PriceCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price_cents must be > 0"})
		return
	}
	if req.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be >= 0"})
		return
	}
	if strings.TrimSpace(req.Currency) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currency is required"})
		return
	}
	if strings.TrimSpace(req.Starts) == "" || strings.TrimSpace(req.Ends) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "starts and ends are required"})
		return
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))

	tt, err := h.repo.CreateTicketType(c.Request.Context(), concertID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket type"})
		return
	}
	c.JSON(http.StatusCreated, tt)
}

func (h *Handler) deleteTicketType(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket type id is required"})
		return
	}
	if err := h.repo.DeleteTicketType(c.Request.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket type"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) updateTicketType(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket type id is required"})
		return
	}

	var req UpdateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if req.Currency != nil {
		val := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if val == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "currency is required"})
			return
		}
		req.Currency = &val
	}

	updated, err := h.repo.UpdateTicketType(c.Request.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket type not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket type"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) listAdminTickets(c *gin.Context) {
	tickets, err := h.repo.ListAdminTickets(c.Request.Context())
	if err != nil {
		log.Printf("tickets admin list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}
	c.JSON(http.StatusOK, tickets)
}

func (h *Handler) createAdminTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.UserID == "" || req.ConcertID == "" || strings.TrimSpace(req.TicketType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id, concert_id, ticket_type are required"})
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "issued"
	}

	created, err := h.repo.CreateAdminTicket(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) updateAdminTicket(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket id is required"})
		return
	}
	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	updated, err := h.repo.UpdateAdminTicket(c.Request.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteAdminTicket(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket id is required"})
		return
	}
	if err := h.repo.DeleteAdminTicket(c.Request.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ticketStats(c *gin.Context) {
	stats, err := h.repo.GetTicketStats(c.Request.Context())
	if err != nil {
		log.Printf("ticket stats failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ticket stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}
