package tickets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v84"
	checkoutsession "github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"
)

type Handler struct {
	repo          *Repository
	frontendURL   string
	webhookSecret string
}

func NewHandler(repo *Repository, frontendURL string, webhookSecret string) *Handler {
	return &Handler{
		repo:          repo,
		frontendURL:   strings.TrimSpace(frontendURL),
		webhookSecret: strings.TrimSpace(webhookSecret),
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/concerts/upcoming", h.listUpcoming)
}

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/tickets/checkout", h.createCheckoutSession)
}

func (h *Handler) RegisterWebhookRoutes(rg *gin.RouterGroup) {
	rg.POST("/stripe/webhook", h.handleWebhook)
}

func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/concerts/:id/ticket-types", h.createTicketType)
}

func (h *Handler) listUpcoming(c *gin.Context) {
	concerts, err := h.repo.ListUpcoming(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load concerts"})
		return
	}
	c.JSON(http.StatusOK, concerts)
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
					Currency: stripe.String(strings.ToLower(info.Currency)),
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
	req := FulfillRequest{
		UserID:       meta["user_id"],
		ConcertID:    meta["concert_id"],
		TicketTypeID: meta["ticket_type_id"],
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

	if err := h.repo.FulfillCheckout(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fulfill checkout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
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

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}
