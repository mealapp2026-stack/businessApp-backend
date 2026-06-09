package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"businessapp/backend/internal/auth"
	"businessapp/backend/internal/model"
	"businessapp/backend/internal/store"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Server struct {
	store     *store.Store
	jwtSecret string
	origin    string
}

var supportedCurrencies = map[string]struct{}{
	"NGN": {}, "GHS": {}, "KES": {}, "ZAR": {}, "UGX": {}, "TZS": {},
	"RWF": {}, "XOF": {}, "XAF": {}, "EGP": {}, "USD": {}, "GBP": {},
	"EUR": {}, "CAD": {}, "AUD": {}, "AED": {}, "SAR": {}, "INR": {},
	"CNY": {}, "JPY": {},
}

func New(store *store.Store, jwtSecret, origin string) *Server {
	return &Server{store: store, jwtSecret: jwtSecret, origin: origin}
}

func (s *Server) Router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors(s.origin))
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.POST("/api/auth/register", s.register)
	router.POST("/api/auth/login", s.login)

	api := router.Group("/api", authMiddleware(s.jwtSecret))
	api.GET("/me", s.me)
	api.PUT("/profile", s.updateProfile)
	api.GET("/customers", s.listCustomers)
	api.POST("/customers", s.createCustomer)
	api.PUT("/customers/:id", s.updateCustomer)
	api.GET("/documents", s.listDocuments)
	api.POST("/documents", s.createDocument)
	api.GET("/documents/:id", s.getDocument)
	api.GET("/revenue", s.revenue)
	return router
}

type registerRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	BusinessName string `json:"businessName" binding:"required"`
	Phone        string `json:"phone"`
}

func (s *Server) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		internalError(c, err)
		return
	}
	user := model.User{
		Email: strings.ToLower(strings.TrimSpace(req.Email)), PasswordHash: hash,
		Business: model.BusinessProfile{Name: strings.TrimSpace(req.BusinessName), Phone: req.Phone, Currency: "NGN"},
	}
	if err := s.store.CreateUser(c, &user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})
			return
		}
		internalError(c, err)
		return
	}
	s.respondWithSession(c, user)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	user, err := s.store.UserByEmail(c, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	s.respondWithSession(c, *user)
}

func (s *Server) respondWithSession(c *gin.Context, user model.User) {
	token, err := auth.CreateToken(s.jwtSecret, user.ID.Hex())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (s *Server) me(c *gin.Context) {
	user, err := s.store.UserByID(c, currentUserID(c))
	if respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) updateProfile(c *gin.Context) {
	var profile model.BusinessProfile
	if err := c.ShouldBindJSON(&profile); err != nil || strings.TrimSpace(profile.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "business name is required"})
		return
	}
	if profile.LogoURL != "" {
		isImage := strings.HasPrefix(profile.LogoURL, "data:image/jpeg;base64,") ||
			strings.HasPrefix(profile.LogoURL, "data:image/png;base64,") ||
			strings.HasPrefix(profile.LogoURL, "https://")
		if !isImage {
			c.JSON(http.StatusBadRequest, gin.H{"error": "logo must be a JPEG, PNG, or HTTPS image"})
			return
		}
		if len(profile.LogoURL) > 2_000_000 {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "logo image is too large"})
			return
		}
	}
	if profile.Currency == "" {
		profile.Currency = "NGN"
	}
	profile.Currency = strings.ToUpper(profile.Currency)
	if _, ok := supportedCurrencies[profile.Currency]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported currency"})
		return
	}
	if err := s.store.UpdateProfile(c, currentUserID(c), profile); respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (s *Server) listCustomers(c *gin.Context) {
	customers, err := s.store.Customers(c, currentUserID(c), c.Query("search"))
	if respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, customers)
}

type customerRequest struct {
	Name    string `json:"name" binding:"required"`
	Phone   string `json:"phone" binding:"required"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

var phoneSeparators = regexp.MustCompile(`[\s\-().]`)

func normalizePhone(phone string) string {
	return phoneSeparators.ReplaceAllString(strings.TrimSpace(phone), "")
}

func (s *Server) createCustomer(c *gin.Context) {
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	customer := model.Customer{
		UserID: currentUserID(c), Name: strings.TrimSpace(req.Name), Phone: normalizePhone(req.Phone),
		Email: strings.TrimSpace(req.Email), Address: strings.TrimSpace(req.Address),
	}
	if err := s.store.CreateCustomer(c, &customer); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a customer with this phone number already exists"})
			return
		}
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, customer)
}

func (s *Server) updateCustomer(c *gin.Context) {
	id, ok := objectIDParam(c)
	if !ok {
		return
	}
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	customer := model.Customer{
		ID: id, UserID: currentUserID(c), Name: strings.TrimSpace(req.Name), Phone: normalizePhone(req.Phone),
		Email: strings.TrimSpace(req.Email), Address: strings.TrimSpace(req.Address),
	}
	if err := s.store.UpdateCustomer(c, customer); respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, customer)
}

type documentRequest struct {
	Type       string           `json:"type" binding:"required,oneof=invoice receipt"`
	CustomerID string           `json:"customerId" binding:"required"`
	Items      []model.LineItem `json:"items" binding:"required,min=1,dive"`
	Discount   float64          `json:"discount" binding:"gte=0"`
	Tax        float64          `json:"tax" binding:"gte=0"`
	Notes      string           `json:"notes"`
	IssueDate  *time.Time       `json:"issueDate"`
	DueDate    *time.Time       `json:"dueDate"`
}

func (s *Server) createDocument(c *gin.Context) {
	var req documentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	customerID, err := primitive.ObjectIDFromHex(req.CustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customerId"})
		return
	}
	customer, err := s.store.CustomerByID(c, currentUserID(c), customerID)
	if respondStoreError(c, err) {
		return
	}
	user, err := s.store.UserByID(c, currentUserID(c))
	if respondStoreError(c, err) {
		return
	}
	subtotal := 0.0
	for i := range req.Items {
		req.Items[i].Amount = req.Items[i].Quantity * req.Items[i].UnitPrice
		subtotal += req.Items[i].Amount
	}
	total := subtotal - req.Discount + req.Tax
	if total < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discount cannot exceed subtotal and tax"})
		return
	}
	issueDate := time.Now().UTC()
	if req.IssueDate != nil {
		issueDate = req.IssueDate.UTC()
	}
	document := model.Document{
		UserID: currentUserID(c), CustomerID: customerID, Type: req.Type,
		Number: store.NextDocumentNumber(req.Type), Items: req.Items, Subtotal: subtotal,
		Discount: req.Discount, Tax: req.Tax, Total: total, Notes: req.Notes,
		IssueDate: issueDate, DueDate: req.DueDate, BusinessSnapshot: user.Business,
		Customer: model.CustomerSnapshot{Name: customer.Name, Phone: customer.Phone, Email: customer.Email, Address: customer.Address},
	}
	if err := s.store.CreateDocument(c, &document); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, document)
}

func (s *Server) listDocuments(c *gin.Context) {
	var customerID *primitive.ObjectID
	if raw := c.Query("customerId"); raw != "" {
		id, err := primitive.ObjectIDFromHex(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customerId"})
			return
		}
		customerID = &id
	}
	kind := c.Query("type")
	if kind != "" && kind != "invoice" && kind != "receipt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be invoice or receipt"})
		return
	}
	documents, err := s.store.Documents(c, currentUserID(c), kind, customerID)
	if respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, documents)
}

func (s *Server) getDocument(c *gin.Context) {
	id, ok := objectIDParam(c)
	if !ok {
		return
	}
	document, err := s.store.DocumentByID(c, currentUserID(c), id)
	if respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, document)
}

func (s *Server) revenue(c *gin.Context) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := now
	var err error
	if raw := c.Query("from"); raw != "" {
		from, err = time.Parse("2006-01-02", raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must use YYYY-MM-DD"})
			return
		}
	}
	if raw := c.Query("to"); raw != "" {
		to, err = time.Parse("2006-01-02", raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must use YYYY-MM-DD"})
			return
		}
		to = to.Add(24*time.Hour - time.Nanosecond)
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from must be before to"})
		return
	}
	summary, err := s.store.RevenueSummary(c, currentUserID(c), from, to)
	if respondStoreError(c, err) {
		return
	}
	c.JSON(http.StatusOK, summary)
}

func objectIDParam(c *gin.Context) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return primitive.NilObjectID, false
	}
	return id, true
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func internalError(c *gin.Context, err error) {
	_ = err
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func respondStoreError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return true
	}
	internalError(c, err)
	return true
}
