package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"businessapp/backend/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("record not found")

type Store struct {
	client    *mongo.Client
	users     *mongo.Collection
	customers *mongo.Collection
	documents *mongo.Collection
	revenues  *mongo.Collection
}

func New(ctx context.Context, uri, database string) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	db := client.Database(database)
	store := &Store{
		client:    client,
		users:     db.Collection("users"),
		customers: db.Collection("customers"),
		documents: db.Collection("documents"),
		revenues:  db.Collection("revenues"),
	}
	if err := store.ensureIndexes(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return store, nil
}

func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Store) ensureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection *mongo.Collection
		models     []mongo.IndexModel
	}{
		{s.users, []mongo.IndexModel{{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{s.users, []mongo.IndexModel{{Keys: bson.D{{Key: "accountId", Value: 1}, {Key: "role", Value: 1}}}}},
		{s.customers, []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "name", Value: 1}}},
		}},
		{s.documents, []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "number", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "customerId", Value: 1}, {Key: "createdAt", Value: -1}}},
		}},
		{s.revenues, []mongo.IndexModel{
			{Keys: bson.D{{Key: "receiptId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "occurredAt", Value: -1}}},
		}},
	}
	for _, entry := range indexes {
		if _, err := entry.collection.Indexes().CreateMany(ctx, entry.models); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	user.ID = primitive.NewObjectID()
	if user.AccountID.IsZero() {
		user.AccountID = user.ID
	}
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt
	_, err := s.users.InsertOne(ctx, user)
	return err
}

func (s *Store) Staff(ctx context.Context, accountID primitive.ObjectID) ([]model.User, error) {
	cursor, err := s.users.Find(ctx, bson.M{"accountId": accountID, "role": "staff"}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	staff := make([]model.User, 0)
	err = cursor.All(ctx, &staff)
	return staff, err
}

func (s *Store) UpdateStaffDisabled(ctx context.Context, accountID, staffID primitive.ObjectID, disabled bool) error {
	result, err := s.users.UpdateOne(ctx, bson.M{
		"_id": staffID, "accountId": accountID, "role": "staff",
	}, bson.M{"$set": bson.M{"disabled": disabled, "updatedAt": time.Now().UTC()}})
	if err == nil && result.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	return &user, normalize(err)
}

func (s *Store) UserByID(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	var user model.User
	err := s.users.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	return &user, normalize(err)
}

func (s *Store) UpdateProfile(ctx context.Context, userID primitive.ObjectID, profile model.BusinessProfile) error {
	result, err := s.users.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{
		"business": profile, "updatedAt": time.Now().UTC(),
	}})
	if err == nil && result.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) CreateCustomer(ctx context.Context, customer *model.Customer) error {
	customer.ID = primitive.NewObjectID()
	customer.CreatedAt = time.Now().UTC()
	customer.UpdatedAt = customer.CreatedAt
	_, err := s.customers.InsertOne(ctx, customer)
	return err
}

func (s *Store) Customers(ctx context.Context, userID primitive.ObjectID, search string) ([]model.Customer, error) {
	filter := bson.M{"userId": userID}
	if search != "" {
		phoneSearch := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(search)
		filter["$or"] = bson.A{
			bson.M{"name": primitive.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}},
			bson.M{"phone": primitive.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}},
			bson.M{"phone": primitive.Regex{Pattern: regexp.QuoteMeta(phoneSearch), Options: "i"}},
		}
	}
	cursor, err := s.customers.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	customers := make([]model.Customer, 0)
	err = cursor.All(ctx, &customers)
	return customers, err
}

func (s *Store) CustomerByID(ctx context.Context, userID, id primitive.ObjectID) (*model.Customer, error) {
	var customer model.Customer
	err := s.customers.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&customer)
	return &customer, normalize(err)
}

func (s *Store) UpdateCustomer(ctx context.Context, customer model.Customer) error {
	result, err := s.customers.UpdateOne(ctx, bson.M{"_id": customer.ID, "userId": customer.UserID}, bson.M{"$set": bson.M{
		"name": customer.Name, "phone": customer.Phone, "email": customer.Email,
		"address": customer.Address, "updatedAt": time.Now().UTC(),
	}})
	if err == nil && result.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) CreateDocument(ctx context.Context, document *model.Document) error {
	document.ID = primitive.NewObjectID()
	document.CreatedAt = time.Now().UTC()
	document.UpdatedAt = document.CreatedAt
	if document.Type != "receipt" {
		_, err := s.documents.InsertOne(ctx, document)
		return err
	}

	session, err := s.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sessionContext mongo.SessionContext) (any, error) {
		if _, insertErr := s.documents.InsertOne(sessionContext, document); insertErr != nil {
			return nil, insertErr
		}
		revenue := model.Revenue{
			ID: primitive.NewObjectID(), UserID: document.UserID, ReceiptID: document.ID,
			CustomerID: document.CustomerID, Amount: document.Total,
			OccurredAt: document.IssueDate, CreatedAt: document.CreatedAt,
		}
		_, insertErr := s.revenues.InsertOne(sessionContext, revenue)
		return nil, insertErr
	})
	return err
}

func (s *Store) Documents(ctx context.Context, userID primitive.ObjectID, kind string, customerID *primitive.ObjectID) ([]model.Document, error) {
	filter := bson.M{"userId": userID}
	if kind != "" {
		filter["type"] = kind
	}
	if customerID != nil {
		filter["customerId"] = *customerID
	}
	cursor, err := s.documents.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	documents := make([]model.Document, 0)
	err = cursor.All(ctx, &documents)
	return documents, err
}

func (s *Store) DocumentByID(ctx context.Context, userID, id primitive.ObjectID) (*model.Document, error) {
	var document model.Document
	err := s.documents.FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&document)
	return &document, normalize(err)
}

func (s *Store) RevenueSummary(ctx context.Context, userID primitive.ObjectID, from, to time.Time) (model.RevenueSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"userId": userID, "occurredAt": bson.M{"$gte": from, "$lte": to}}}},
		{{Key: "$group", Value: bson.M{
			"_id":    bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$occurredAt"}},
			"amount": bson.M{"$sum": "$amount"}, "count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}
	cursor, err := s.revenues.Aggregate(ctx, pipeline)
	if err != nil {
		return model.RevenueSummary{}, err
	}
	defer cursor.Close(ctx)
	points := make([]model.RevenuePoint, 0)
	if err := cursor.All(ctx, &points); err != nil {
		return model.RevenueSummary{}, err
	}
	summary := model.RevenueSummary{Items: points}
	for _, point := range points {
		summary.Total += point.Amount
		summary.Count += point.Count
	}
	return summary, nil
}

func NextDocumentNumber(kind string) string {
	prefix := "INV"
	if kind == "receipt" {
		prefix = "RCT"
	}
	return fmt.Sprintf("%s-%s", prefix, primitive.NewObjectID().Hex()[12:])
}

func normalize(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}
