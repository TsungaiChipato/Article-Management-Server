package controller

import (
	"article-management-service/pkg/db"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const MAX_IMAGE_SIZE = 5 * 1024 * 1024
const MAX_IMAGE_AMOUNT = 3

type ArticleController struct {
	ImageDirectory     string
	GenerateIdentifier func() string
	ArticleDbHandler   db.ArticleDbHandlerInterface
	Validate           *validator.Validate
}

type NewArticleBody struct {
	Title          string    `json:"title" validate:"required"`
	ExpirationDate time.Time `json:"expirationDate" validate:"required"`
	Description    string    `json:"description" validate:"required,max=4000"`
}

// Create controller inserts the article based on json body; return the id hex.
// Note: that is the same body is used twice, multiple documents will be made.
// Preventing this was not part of the acceptance criteria; a possible fix for this
// is that the unique identifier should be part of the request body
func (c *ArticleController) Create(context *gin.Context) {
	article := &NewArticleBody{}
	if err := context.BindJSON(article); err != nil {
		handleError(context, err, http.StatusBadRequest)
		return
	}

	if err := c.Validate.Struct(article); err != nil {
		handleError(context, err, http.StatusBadRequest)
		return
	}

	id, err := c.ArticleDbHandler.InsertOne(db.ArticleDb{
		Title:          article.Title,
		Description:    article.Description,
		ExpirationDate: article.ExpirationDate,
	})

	if err != nil {
		handleError(context, err, http.StatusInternalServerError)
		return
	}

	context.JSON(http.StatusCreated, gin.H{"id": id.Hex()})
}

// TODO: check mime type
func (c *ArticleController) AttachImage(context *gin.Context) {
	articleId, err := primitive.ObjectIDFromHex(context.Param("articleId"))
	if err != nil {
		handleError(context, err, http.StatusBadRequest)
		return
	}

	article, err := c.ArticleDbHandler.FindOneById(articleId)
	if err != nil {
		handleError(context, nil, http.StatusInternalServerError)
		return
	}

	if article == nil {
		handleError(context, nil, http.StatusNotFound)
		return
	}

	if len(article.ImageFilePaths) >= MAX_IMAGE_AMOUNT {
		handleError(context, nil, http.StatusForbidden)
		return
	}

	file, err := context.FormFile("file")
	if err != nil {
		handleError(context, err, http.StatusInternalServerError)
		return
	}

	if file.Size > MAX_IMAGE_SIZE {
		handleError(context, nil, http.StatusBadRequest)
		return
	}

	id := c.GenerateIdentifier()
	path := filepath.Join(c.ImageDirectory, id)
	context.SaveUploadedFile(file, path)
	c.ArticleDbHandler.AppendImage(articleId, path)

	context.Status(http.StatusOK)
}

func (c *ArticleController) Find(context *gin.Context) {
	withImagesStr := context.Request.URL.Query().Get("withImages")
	var withImages *bool
	tmp, err := strconv.ParseBool(withImagesStr)
	if err == nil {
		withImages = &tmp
	}

	var titles []string
	if withImages == nil {
		titles, err = c.ArticleDbHandler.FindAllTitles()
	} else {
		titles, err = c.ArticleDbHandler.FindTitlesByHasImage(*withImages)

	}
	if err != nil {
		handleError(context, nil, http.StatusBadRequest)
	}

	context.JSON(http.StatusOK, titles)
}

func handleError(context *gin.Context, err error, status int) {
	if err != nil {
		log.Println("Error:", err)
	}
	context.AbortWithStatus(status)
}
