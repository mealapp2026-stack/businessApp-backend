package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Business     BusinessProfile    `bson:"business" json:"business"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type BusinessProfile struct {
	Name        string `bson:"name" json:"name"`
	LogoURL     string `bson:"logoUrl,omitempty" json:"logoUrl,omitempty"`
	Phone       string `bson:"phone,omitempty" json:"phone,omitempty"`
	Address     string `bson:"address,omitempty" json:"address,omitempty"`
	Currency    string `bson:"currency" json:"currency"`
	InvoiceNote string `bson:"invoiceNote,omitempty" json:"invoiceNote,omitempty"`
	ReceiptNote string `bson:"receiptNote,omitempty" json:"receiptNote,omitempty"`
}

type Customer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"-"`
	Name      string             `bson:"name" json:"name"`
	Phone     string             `bson:"phone" json:"phone"`
	Email     string             `bson:"email,omitempty" json:"email,omitempty"`
	Address   string             `bson:"address,omitempty" json:"address,omitempty"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type LineItem struct {
	Description string  `bson:"description" json:"description" binding:"required"`
	Quantity    float64 `bson:"quantity" json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `bson:"unitPrice" json:"unitPrice" binding:"required,gte=0"`
	Amount      float64 `bson:"amount" json:"amount"`
}

type CustomerSnapshot struct {
	Name    string `bson:"name" json:"name"`
	Phone   string `bson:"phone" json:"phone"`
	Email   string `bson:"email,omitempty" json:"email,omitempty"`
	Address string `bson:"address,omitempty" json:"address,omitempty"`
}

type Document struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID           primitive.ObjectID `bson:"userId" json:"-"`
	CustomerID       primitive.ObjectID `bson:"customerId" json:"customerId"`
	Customer         CustomerSnapshot   `bson:"customer" json:"customer"`
	Type             string             `bson:"type" json:"type"`
	Number           string             `bson:"number" json:"number"`
	Items            []LineItem         `bson:"items" json:"items"`
	Subtotal         float64            `bson:"subtotal" json:"subtotal"`
	Discount         float64            `bson:"discount" json:"discount"`
	Tax              float64            `bson:"tax" json:"tax"`
	Total            float64            `bson:"total" json:"total"`
	Notes            string             `bson:"notes,omitempty" json:"notes,omitempty"`
	IssueDate        time.Time          `bson:"issueDate" json:"issueDate"`
	DueDate          *time.Time         `bson:"dueDate,omitempty" json:"dueDate,omitempty"`
	BusinessSnapshot BusinessProfile    `bson:"businessSnapshot" json:"businessSnapshot"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Revenue struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"userId" json:"-"`
	ReceiptID  primitive.ObjectID `bson:"receiptId" json:"receiptId"`
	CustomerID primitive.ObjectID `bson:"customerId" json:"customerId"`
	Amount     float64            `bson:"amount" json:"amount"`
	OccurredAt time.Time          `bson:"occurredAt" json:"occurredAt"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}

type RevenueSummary struct {
	Total float64        `json:"total"`
	Count int64          `json:"count"`
	Items []RevenuePoint `json:"items"`
}

type RevenuePoint struct {
	Date   string  `bson:"_id" json:"date"`
	Amount float64 `bson:"amount" json:"amount"`
	Count  int64   `bson:"count" json:"count"`
}
