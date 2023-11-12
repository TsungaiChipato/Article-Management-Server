package db

import (
	"article-management-service/pkg/env"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func createDb(t *testing.T) (*mongo.Database, func()) {
	cfg, err := env.Load()
	if err != nil {
		panic(err)
	}

	mm := MockMongo{}
	uri, err := mm.HostMemoryDb(cfg.MongodPath)
	if err != nil {
		t.Error("Failed to launch memory db")
		t.FailNow()
	}

	conn := Connection{}
	err = conn.Connect(uri)
	if err != nil {
		t.Error("Failed to connect to memory server")
		t.Fail()
	}

	return conn.Database, func() {
		conn.Close()
		mm.Close()
	}
}

func createColl(t *testing.T) (h ArticleDbHandler, close func()) {
	db, close := createDb(t)

	h = ArticleDbHandler{}
	err := h.New(db)
	if err != nil {
		t.Error("Failed to create the collection")
		t.FailNow()
	}
	return
}

func TestArticleDbHandler_New(t *testing.T) {
	t.Parallel()
	db, close := createDb(t)
	defer close()

	t.Run("Successfully create the collection", func(t *testing.T) {
		h := ArticleDbHandler{}

		err := h.New(db)
		if err != nil {
			t.Errorf("ArticleDbHandler.New() error = %v, wantErr %v", err, false)
			return
		}
	})
}

func TestArticleDbHandler_InsertOne(t *testing.T) {
	t.Parallel()

	t.Run("Successfully inserted one article", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		article := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
		}
		id, err := h.InsertOne(article)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		article.Id = createdArticle.Id
		article.ImageFilePaths = createdArticle.ImageFilePaths
		if !reflect.DeepEqual(*createdArticle, article) {
			t.Errorf("ArticleDbHandler.InsertOne() = %v, want %v", *createdArticle, article)
			return
		}
	})

	t.Run("Successfully inserted one article with predefined images", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		article := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
			ImageFilePaths: []string{"file_path"},
		}
		id, err := h.InsertOne(article)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		article.Id = createdArticle.Id
		if !reflect.DeepEqual(*createdArticle, article) {
			t.Errorf("ArticleDbHandler.InsertOne() = %v, want %v", *createdArticle, article)
			return
		}
	})
}

func TestArticleDbHandler_AppendImage(t *testing.T) {
	t.Parallel()

	t.Run("Successfully append one image", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		expected := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
		}
		id, err := h.InsertOne(expected)
		if err != nil {
			t.Errorf("ArticleDbHandler.AppendImage() error = %v, wantErr %v", err, false)
			return
		}

		imagePath := "test_path"
		h.AppendImage(id, imagePath)

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.AppendImage() error = %v, wantErr %v", err, false)
			return
		}

		expected.ImageFilePaths = []string{imagePath}
		expected.Id = createdArticle.Id

		if !reflect.DeepEqual(*createdArticle, expected) {
			t.Errorf("ArticleDbHandler.AppendImage() = %v, want %v", *createdArticle, expected)
			return
		}
	})

	t.Run("Successfully append multiple different images", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		expected := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
		}
		id, err := h.InsertOne(expected)
		if err != nil {
			t.Errorf("ArticleDbHandler.AppendImage() error = %v, wantErr %v", err, false)
			return
		}

		imagePath1 := "test_path1"
		imagePath2 := "test_path2"
		h.AppendImage(id, imagePath1)
		h.AppendImage(id, imagePath2)

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.AppendImage() error = %v, wantErr %v", err, false)
			return
		}

		expected.ImageFilePaths = []string{imagePath1, imagePath2}
		expected.Id = createdArticle.Id

		if !reflect.DeepEqual(*createdArticle, expected) {
			t.Errorf("ArticleDbHandler.AppendImage() = %v, want %v", *createdArticle, expected)
			return
		}
	})

	// testing $addToSet
	t.Run("Successfully prevent the same path to be uploaded multiple times", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		expected := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
		}
		id, err := h.InsertOne(expected)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		imagePath1 := "test_path"
		imagePath2 := "test_path"
		h.AppendImage(id, imagePath1)
		h.AppendImage(id, imagePath2)

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.InsertOne() error = %v, wantErr %v", err, false)
			return
		}

		expected.ImageFilePaths = []string{imagePath1}
		expected.Id = createdArticle.Id

		if !reflect.DeepEqual(*createdArticle, expected) {
			t.Errorf("ArticleDbHandler.FindOneById() = %v, want %v", *createdArticle, expected)
			return
		}
	})
}

func TestArticleDbHandler_FindOneById(t *testing.T) {
	t.Parallel()

	t.Run("Successfully found the image by id", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		article := ArticleDb{
			Title:          "Test_Title",
			ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
			Description:    "Test_Description",
		}
		id, err := h.InsertOne(article)
		if err != nil {
			t.Errorf("ArticleDbHandler.FindOneById() error = %v, wantErr %v", err, false)
			return
		}

		createdArticle, err := h.FindOneById(id)
		if err != nil {
			t.Errorf("ArticleDbHandler.FindOneById() error = %v, wantErr %v", err, false)
			return
		}

		article.Id = createdArticle.Id
		article.ImageFilePaths = createdArticle.ImageFilePaths
		if !reflect.DeepEqual(*createdArticle, article) {
			t.Errorf("ArticleDbHandler.FindOneById() = %v, want %v", *createdArticle, article)
			return
		}
	})

	t.Run("Successfully found nothing with non-existing article", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		found, err := h.FindOneById(primitive.NewObjectID())
		if err != nil {
			t.Errorf("ArticleDbHandler.FindOneById() error = %v, wantErr %v", err, false)
			return
		}

		if found != nil {
			t.Errorf("ArticleDbHandler.FindOneById() = %v, want %v", *found, nil)
			return
		}
	})
}

func TestArticleDbHandler_FindAllTitles(t *testing.T) {
	t.Parallel()

	t.Run("Successfully found all the titles", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		titles := []string{"Test_Title1", "Test_Title2", "Test_Title3"}

		for _, title := range titles {
			article := ArticleDb{
				Title:          title,
				ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
				Description:    "Test_Description",
			}
			_, err := h.InsertOne(article)
			if err != nil {
				t.Errorf("ArticleDbHandler.FindAllTitles() error = %v, wantErr %v", err, false)
				return
			}
		}

		found, err := h.FindAllTitles()
		if err != nil {
			t.Errorf("ArticleDbHandler.FindAllTitles() error = %v, wantErr %v", err, false)
			return
		}

		if !reflect.DeepEqual(found, titles) {
			t.Errorf("ArticleDbHandler.FindAllTitles() = %v, want %v", titles, found)
			return
		}
	})
}

func TestArticleDbHandler_FindTitlesByHasImage(t *testing.T) {
	t.Parallel()

	init := func(h ArticleDbHandler) (titlesWithImages []string, titlesWithoutImages []string) {
		titlesWithImages = []string{"With_Title1", "With_Title2", "With_Title3"}
		for _, title := range titlesWithImages {
			article := ArticleDb{
				Title:          title,
				ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
				Description:    "Test_Description",
				ImageFilePaths: []string{"image_path"},
			}
			_, err := h.InsertOne(article)
			if err != nil {
				t.Errorf("ArticleDbHandler.FindTitlesByHasImage() error = %v, wantErr %v", err, false)
				return
			}
		}

		titlesWithoutImages = []string{"Without_Title1", "Without_Title2", "Without_Title3"}
		for _, title := range titlesWithoutImages {
			article := ArticleDb{
				Title:          title,
				ExpirationDate: time.Now().Add(time.Hour).UTC().Truncate(time.Millisecond), // have to truncate, because mongo does not store microseconds
				Description:    "Test_Description",
			}
			_, err := h.InsertOne(article)
			if err != nil {
				t.Errorf("ArticleDbHandler.FindTitlesByHasImage() error = %v, wantErr %v", err, false)
				return
			}
		}

		return
	}

	t.Run("Successfully found all the titles with images", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		titlesWithImages, _ := init(h)

		found, err := h.FindTitlesByHasImage(true)
		if err != nil {
			t.Errorf("ArticleDbHandler.FindTitlesByHasImage() error = %v, wantErr %v", err, false)
			return
		}

		if !reflect.DeepEqual(found, titlesWithImages) {
			t.Errorf("ArticleDbHandler.FindTitlesByHasImage() = %v, want %v", titlesWithImages, found)
			return
		}
	})

	t.Run("Successfully found all the titles without images", func(t *testing.T) {
		t.Parallel()

		h, close := createColl(t)
		defer close()

		_, titlesWithoutImages := init(h)

		found, err := h.FindTitlesByHasImage(false)
		if err != nil {
			t.Errorf("ArticleDbHandler.FindTitlesByHasImage() error = %v, wantErr %v", err, false)
			return
		}

		if !reflect.DeepEqual(found, titlesWithoutImages) {
			t.Errorf("ArticleDbHandler.FindTitlesByHasImage() = %v, want %v", titlesWithoutImages, found)
			return
		}
	})
}
