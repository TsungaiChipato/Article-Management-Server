package controller

import (
	"article-management-service/pkg/db"
	"article-management-service/pkg/mocks"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func createJSONBodyContext(t *testing.T, body NewArticleBody) *gin.Context {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Create a JSON-RPC request payload
	jsonRequest, err := json.Marshal(body)
	if err != nil {
		t.Error("Failed to Marshal given NewArticleBody")
		t.FailNow()
	}

	requestBody := bytes.NewBuffer(jsonRequest)

	// Set the request body and content type in the context
	context.Request = &http.Request{
		Body:   io.NopCloser(requestBody),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}

	return context
}

func TestArticleController_Create(t *testing.T) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	type fields struct {
		ImageDirectory     string
		GenerateIdentifier func() string
		ArticleDbHandler   db.ArticleDbHandlerInterface
		Validate           *validator.Validate
	}
	type args struct {
		context *gin.Context
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus int
	}{
		{
			name:           "Prevent missing title",
			fields:         fields{ArticleDbHandler: &mocks.MockArticleDbHandler{}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{ExpirationDate: time.Now(), Description: "Test_Description"})},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Prevent missing expirationDate",
			fields:         fields{ArticleDbHandler: &mocks.MockArticleDbHandler{}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{Title: "Test_Title", Description: "Test_Description"})},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Prevent missing description",
			fields:         fields{ArticleDbHandler: &mocks.MockArticleDbHandler{}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{Title: "Test_Title", ExpirationDate: time.Now()})},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Prevent too long description",
			fields:         fields{ArticleDbHandler: &mocks.MockArticleDbHandler{}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{Title: "Test_Title", ExpirationDate: time.Now(), Description: strings.Repeat("A", 40001)})},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal error - insertOne failure",
			fields: fields{ArticleDbHandler: &mocks.MockArticleDbHandler{InsertOneFunc: func(new db.ArticleDb) (primitive.ObjectID, error) {
				return primitive.NilObjectID, fmt.Errorf("test failure")
			}}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{Title: "Test_Title", ExpirationDate: time.Now(), Description: "Test_Description"})},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			fields: fields{ArticleDbHandler: &mocks.MockArticleDbHandler{InsertOneFunc: func(new db.ArticleDb) (primitive.ObjectID, error) {
				return primitive.ObjectIDFromHex("6547986414e33ec8c072c2d3")
			}}, Validate: validate},
			args:           args{context: createJSONBodyContext(t, NewArticleBody{Title: "Test_Title", ExpirationDate: time.Now(), Description: "Test_Description"})},
			expectedStatus: http.StatusCreated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ArticleController{
				ImageDirectory:     tt.fields.ImageDirectory,
				GenerateIdentifier: tt.fields.GenerateIdentifier,
				ArticleDbHandler:   tt.fields.ArticleDbHandler,
				Validate:           tt.fields.Validate,
			}
			c.Create(tt.args.context)

			foundStatus := tt.args.context.Writer.Status()
			if foundStatus != tt.expectedStatus {
				t.Errorf("ArticleController_Create() = %v, want %v", foundStatus, tt.expectedStatus)
				return
			}
		})
	}
}

func TestArticleController_AttachImage(t *testing.T) {
	t.Skip("TODO: no param; bad request")
	t.Skip("TODO: internal error - findOneById failure")
	t.Skip("TODO: article not found")
	t.Skip("TODO: too many articles attached")
	t.Skip("TODO: image size too big")
	t.Skip("TODO: success")
}

func TestArticleController_Find(t *testing.T) {
	t.Skip("TODO: success - no param")
	t.Skip("TODO: success - invalid param")
	t.Skip("TODO: success - withImages:true")
	t.Skip("TODO: success - withImages:false")
	t.Skip("TODO: internal error - findAllTitles failure")
	t.Skip("TODO: internal error - findTitlesByHasImage failure")
}
