package cart

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	checkoutsession "github.com/stripe/stripe-go/v84/checkout/session"

	"groupie-tracker/internal/middleware"
)

type Handler struct {
	repo        *Repository
	frontendURL string
}

func NewHandler(repo *Repository, frontendURL string) *Handler {
	return &Handler{repo: repo, frontendURL: frontendURL}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("/cart")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("", h.list)
		protected.POST("/items", h.addItem)
		protected.PATCH("/items/:id", h.updateItem)
		protected.DELETE("/items/:id", h.deleteItem)
		protected.POST("/checkout", h.checkout)
	}
}

func (h *Handler) list(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	resp, err := h.repo.ListCart(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load cart"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) addItem(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}

	item, err := h.repo.AddItem(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateItem(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	itemID := strings.TrimSpace(c.Param("id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item id is required"})
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	item, err := h.repo.UpdateItem(c.Request.Context(), userID, itemID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) deleteItem(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	itemID := strings.TrimSpace(c.Param("id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item id is required"})
		return
	}

	if err := h.repo.DeleteItem(c.Request.Context(), userID, itemID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) checkout(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userEmail, ok := userEmailFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user email"})
		return
	}

	items, err := h.repo.GetCheckoutItems(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load cart"})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart is empty"})
		return
	}
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stripe is not configured"})
		return
	}

	currency := items[0].Currency
	var lineItems []*stripe.CheckoutSessionLineItemParams
	itemIDs := make([]string, 0, len(items))

	for _, item := range items {
		if item.Currency != currency {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cart contains multiple currencies"})
			return
		}

		productName := item.Title
		if item.City != "" {
			productName = productName + " - " + item.City
		}

		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(int64(item.Quantity)),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(strings.ToLower(item.Currency)),
				UnitAmount: stripe.Int64(item.PriceCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(productName),
				},
			},
		})

		itemIDs = append(itemIDs, item.ID)
	}

	successURL := joinURL(h.frontendURL, "/tickets?success=1")
	cancelURL := joinURL(h.frontendURL, "/tickets?canceled=1")

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:         lineItems,
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		CustomerEmail:     stripe.String(userEmail),
		ClientReferenceID: stripe.String(userID),
		Metadata: map[string]string{
			"user_id":         userID,
			"cart_item_ids":   strings.Join(itemIDs, ","),
			"cart_item_count": strconv.Itoa(len(itemIDs)),
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

func userIDFromContext(c *gin.Context) (string, bool) {
	userIDValue, _ := c.Get("user_id")
	userID, ok := userIDValue.(string)
	return userID, ok && userID != ""
}

func userEmailFromContext(c *gin.Context) (string, bool) {
	userEmailValue, _ := c.Get("user_email")
	userEmail, ok := userEmailValue.(string)
	return userEmail, ok && userEmail != ""
}

func joinURL(base, suffix string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		return suffix
	}
	if strings.HasPrefix(suffix, "/") {
		return base + suffix
	}
	return base + "/" + suffix
}
