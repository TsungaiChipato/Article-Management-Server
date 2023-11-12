package mocks

import (
	"article-management-service/pkg/db"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockArticleDbHandler struct {
	NewFunc                  func(database *mongo.Database) error
	InsertOneFunc            func(new db.ArticleDb) (primitive.ObjectID, error)
	AppendImageFunc          func(id primitive.ObjectID, path string) error
	FindOneByIdFunc          func(id primitive.ObjectID) (*db.ArticleDb, error)
	FindAllTitlesFunc        func() ([]string, error)
	FindTitlesByHasImageFunc func(withImage bool) ([]string, error)
}

func (m *MockArticleDbHandler) New(database *mongo.Database) error {
	if m.NewFunc != nil {
		return m.NewFunc(database)
	}
	return nil
}

func (m *MockArticleDbHandler) InsertOne(new db.ArticleDb) (primitive.ObjectID, error) {
	if m.InsertOneFunc != nil {
		return m.InsertOneFunc(new)
	}
	return primitive.NilObjectID, nil
}

func (m *MockArticleDbHandler) AppendImage(id primitive.ObjectID, path string) error {
	if m.AppendImageFunc != nil {
		return m.AppendImageFunc(id, path)
	}
	return nil
}

func (m *MockArticleDbHandler) FindOneById(id primitive.ObjectID) (*db.ArticleDb, error) {
	if m.FindOneByIdFunc != nil {
		return m.FindOneByIdFunc(id)
	}
	return nil, nil
}

func (m *MockArticleDbHandler) FindAllTitles() ([]string, error) {
	if m.FindAllTitlesFunc != nil {
		return m.FindAllTitlesFunc()
	}
	return nil, nil
}

func (m *MockArticleDbHandler) FindTitlesByHasImage(withImage bool) ([]string, error) {
	if m.FindTitlesByHasImageFunc != nil {
		return m.FindTitlesByHasImageFunc(withImage)
	}
	return nil, nil
}
