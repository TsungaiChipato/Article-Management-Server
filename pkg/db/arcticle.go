package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ArticleDbHandler struct {
	coll *mongo.Collection
}

type ArticleDbHandlerInterface interface {
	New(database *mongo.Database) error
	InsertOne(new ArticleDb) (primitive.ObjectID, error)
	AppendImage(id primitive.ObjectID, path string) error
	FindOneById(id primitive.ObjectID) (*ArticleDb, error)
	FindAllTitles() ([]string, error)
	FindTitlesByHasImage(withImage bool) ([]string, error)
}

// ArticleDbHandler implements ArticleDbHandlerInterface.
type ArticleDb struct {
	Id             primitive.ObjectID `bson:"_id,omitempty"`
	Title          string             `bson:"title,omitempty"`
	ExpirationDate time.Time          `bson:"expirationDate,omitempty"`
	Description    string             `bson:"description,omitempty"`
	ImageFilePaths []string           `bson:"imagePaths,omitempty"`
}

// Creates a new articles collection and adds indexes for ttl
func (h *ArticleDbHandler) New(database *mongo.Database) error {
	h.coll = database.Collection("articles")

	// makes the expirationDate the TTL of the document
	_, err := h.coll.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys:    bson.D{{Key: "expirationDate", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

// Inserts one article in the db
func (h *ArticleDbHandler) InsertOne(new ArticleDb) (primitive.ObjectID, error) {
	result, err := h.coll.InsertOne(context.TODO(), new)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return result.InsertedID.(primitive.ObjectID), nil
}

// Appends an image path to an article in the db
func (h *ArticleDbHandler) AppendImage(id primitive.ObjectID, path string) error {
	update := bson.M{"$addToSet": bson.M{"imagePaths": path}} // should not have duplicate paths
	_, err := h.coll.UpdateByID(context.TODO(), id, update)
	return err
}

// Finds one article in the db using the indexed id
func (h *ArticleDbHandler) FindOneById(id primitive.ObjectID) (*ArticleDb, error) {
	filter := bson.D{{Key: "_id", Value: id}}
	var article ArticleDb
	err := h.coll.FindOne(context.TODO(), filter).Decode(&article)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &article, nil
}

// Helper function to consumes an article cursor and returns an title array
func consumeArticleCursorForTitles(cur *mongo.Cursor) ([]string, error) {
	titles := make([]string, 0)

	for cur.Next(context.TODO()) {
		var a ArticleDb
		if err := cur.Decode(&a); err != nil {
			return nil, err
		}

		titles = append(titles, a.Title)
	}

	return titles, nil
}

// Finds all titles of the articles in the db
func (h *ArticleDbHandler) FindAllTitles() ([]string, error) {
	cur, err := h.coll.Find(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}

	titles, err := consumeArticleCursorForTitles(cur)
	return titles, err
}

// Finds all titles of the articles if they have an image or not in the db
func (h *ArticleDbHandler) FindTitlesByHasImage(withImage bool) ([]string, error) {
	filter := bson.M{"imagePaths.0": bson.M{"$exists": withImage}}
	cur, err := h.coll.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	titles, err := consumeArticleCursorForTitles(cur)
	return titles, err
}
