package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"article-management-service/pkg/controller"
	"article-management-service/pkg/db"
	"article-management-service/pkg/env"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// generate a random image with random colors so PNG compression does not make it too small
func createImage(width, height int) []byte {
	// Seed the random number generator to produce different results each time
	rand.Seed(time.Now().UnixNano())

	rect := image.Rect(0, 0, width, height)
	img := image.NewRGBA(rect)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			// Generate a random color for each pixel
			col := color.RGBA{
				uint8(rand.Intn(256)), // Red
				uint8(rand.Intn(256)), // Green
				uint8(rand.Intn(256)), // Blue
				0xFF,                  // Alpha (fully opaque)
			}
			img.Set(x, y, col)
		}
	}

	var imgBytes bytes.Buffer
	err := png.Encode(&imgBytes, img)
	if err != nil {
		panic(err)
	}

	return imgBytes.Bytes()
}

func generateLargeString() string {
	largeString := strings.Repeat("a", 4001) // Adjust the length based on your allowed limit
	return largeString
}

func createValidArticleBody(title string) []byte {
	// Create a JSON-RPC request payload
	jsonRequest, err := json.Marshal(controller.NewArticleBody{
		Title:          title,
		ExpirationDate: time.Now().Add(time.Hour * 24),
		Description:    "Lorum ipsum",
	})
	if err != nil {
		panic(err)
	}

	return jsonRequest
}

func initializeServer(t *testing.T) (*gin.Engine, func()) {
	cfg, err := env.Load()
	if err != nil {
		panic(err)
	}

	mm := db.MockMongo{}
	uri, err := mm.HostMemoryDb(cfg.MongodPath)
	if err != nil {
		panic(err)
	}

	conn := db.Connection{}
	err = conn.Connect(uri)
	if err != nil {
		panic(err)
	}

	engine := gin.Default()

	engine.SetTrustedProxies(nil)

	dbHandler := &db.ArticleDbHandler{}
	err = dbHandler.New(conn.Database)
	if err != nil {
		panic(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	articleController := &controller.ArticleController{
		ArticleDbHandler:   dbHandler,
		ImageDirectory:     t.TempDir(),
		GenerateIdentifier: func() string { return uuid.New().String() },
		Validate:           validate,
	}

	router := NewRouter(articleController, engine)
	router.Init()

	return engine, func() {
		conn.Close()
		mm.Close()
	}
}

func TestRouter_PostArticle(t *testing.T) {
	t.Parallel()

	t.Run("Successfully create an article", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		reqBody := createValidArticleBody("create an article")

		req, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)

		if response.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, but got %d", http.StatusCreated, response.Code)
		}

		var responseJSON map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &responseJSON); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
		}

		if id, ok := responseJSON["id"].(string); !ok || id == "" {
			t.Errorf("Expected a valid article ID in the response, but got an empty or invalid value.")
		}
	})

	t.Run("Prevent too large description", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		reqBody, err := json.Marshal(controller.NewArticleBody{
			Title:          "Prevent too large description",
			ExpirationDate: time.Now().Add(time.Hour * 24),
			Description:    generateLargeString(),
		})
		if err != nil {
			t.Errorf("Unable to Marshal JSON")
			return
		}

		req, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)

		if response.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d; got %d", http.StatusBadRequest, response.Code)
		}
	})

	// TODO: write test for image Validating expiration date
	t.Run("Validate expiration date", func(t *testing.T) {})
}

func TestRouter_PostImage(t *testing.T) {
	t.Parallel()

	t.Run("Prevent adding more than 3 images to an article", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		articleReqBody := createValidArticleBody("Prevent adding more than 3")

		articleReq, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(articleReqBody))
		articleReq.Header.Set("Content-Type", "application/json")

		articleResponse := httptest.NewRecorder()
		engine.ServeHTTP(articleResponse, articleReq)

		if articleResponse.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, but got %d", http.StatusCreated, articleResponse.Code)
		}

		var articleResponseJSON map[string]interface{}
		if err := json.Unmarshal(articleResponse.Body.Bytes(), &articleResponseJSON); err != nil {
			t.Errorf("Failed to parse article response JSON: %v", err)
		}

		articleID, _ := articleResponseJSON["id"].(string)

		// Try to add more than 3 images to the article
		for i := 0; i <= 3; i++ {
			imageReq, _ := http.NewRequest("POST", "/image/"+articleID, nil)
			imageReq.Header.Set("Content-Type", "multipart/form-data")

			imageData := createImage(10, 10)
			imageBuf := new(bytes.Buffer)
			imageWriter := multipart.NewWriter(imageBuf)
			imagePart, _ := imageWriter.CreateFormFile("file", "image.jpg")
			imagePart.Write(imageData)
			imageWriter.Close()

			imageReq, _ = http.NewRequest("POST", "/image/"+articleID, imageBuf)
			imageReq.Header.Set("Content-Type", imageWriter.FormDataContentType())

			imageResponse := httptest.NewRecorder()
			engine.ServeHTTP(imageResponse, imageReq)
			if i == 3 {
				if imageResponse.Code != http.StatusForbidden {
					t.Errorf("Expected status %d; got %d", http.StatusForbidden, imageResponse.Code)
				}
			}
		}
	})

	t.Run("Attach image to article", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		reqBody := createValidArticleBody("Attach image to article")

		req, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)

		if response.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, but got %d", http.StatusCreated, response.Code)
		}

		var responseJSON map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &responseJSON); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
		}

		articleID, ok := responseJSON["id"].(string)
		if !ok || articleID == "" {
			t.Errorf("Expected a valid article ID in the response, but got an empty or invalid value.")
		}

		imageData := createImage(10, 10)
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)
		part, _ := writer.CreateFormFile("file", "image.jpg")
		part.Write(imageData)
		writer.Close()

		req, _ = http.NewRequest("POST", "/image/"+articleID, buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d; got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("Prevent too large image", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		reqBody := createValidArticleBody("Attach image to article")

		req, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)

		if response.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, but got %d", http.StatusCreated, response.Code)
		}

		var responseJSON map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &responseJSON); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
		}

		articleID, ok := responseJSON["id"].(string)
		if !ok || articleID == "" {
			t.Errorf("Expected a valid article ID in the response, but got an empty or invalid value.")
		}

		imageData := createImage(2000, 2000) //around 6mb
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)
		part, _ := writer.CreateFormFile("file", "image.jpg")
		part.Write(imageData)
		writer.Close()

		req, _ = http.NewRequest("POST", "/image/"+articleID, buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d; got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestRouter_GetArticles(t *testing.T) {
	t.Parallel()

	init := func(engine *gin.Engine) (withImages []string, withoutImages []string) {
		for i := 0; i < 5; i++ {
			title := strconv.Itoa(i + 1)
			articleReqBody := createValidArticleBody(title)

			articleReq, _ := http.NewRequest("POST", "/article", bytes.NewBuffer(articleReqBody))
			articleReq.Header.Set("Content-Type", "application/json")

			articleResponse := httptest.NewRecorder()
			engine.ServeHTTP(articleResponse, articleReq)

			if articleResponse.Code != http.StatusCreated {
				t.Errorf("Expected status code %d, but got %d", http.StatusCreated, articleResponse.Code)
				return
			}

			if i%2 == 0 {
				withImages = append(withImages, title)
				var articleJSON map[string]interface{}
				if err := json.Unmarshal(articleResponse.Body.Bytes(), &articleJSON); err != nil {
					t.Errorf("Failed to parse article response JSON: %v", err)
				}

				articleID, _ := articleJSON["id"].(string)

				imageReq, _ := http.NewRequest("POST", "/image/"+articleID, nil)
				imageReq.Header.Set("Content-Type", "multipart/form-data")

				imageData := createImage(10, 10)
				imageBuf := new(bytes.Buffer)
				imageWriter := multipart.NewWriter(imageBuf)
				imagePart, _ := imageWriter.CreateFormFile("file", "image.jpg")
				imagePart.Write(imageData)
				imageWriter.Close()

				imageReq, _ = http.NewRequest("POST", "/image/"+articleID, imageBuf)
				imageReq.Header.Set("Content-Type", imageWriter.FormDataContentType())

				engine.ServeHTTP(httptest.NewRecorder(), imageReq)
			} else {
				withoutImages = append(withoutImages, title)
			}
		}
		return
	}

	t.Run("Find all articles with images", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		withImages, _ := init(engine)

		allArticlesWithImagesReq, _ := http.NewRequest("GET", "/article?withImages=true", nil)
		allArticlesWithImagesResponse := httptest.NewRecorder()
		engine.ServeHTTP(allArticlesWithImagesResponse, allArticlesWithImagesReq)

		if allArticlesWithImagesResponse.Code != http.StatusOK {
			t.Errorf("Expected status code %d; got %d", http.StatusOK, allArticlesWithImagesResponse.Code)
		}

		// Parse the response and check if there are exactly 2 articles with images
		var responseArray []string
		if err := json.Unmarshal(allArticlesWithImagesResponse.Body.Bytes(), &responseArray); err != nil {
			fmt.Printf("Response Body: %s\n", allArticlesWithImagesResponse.Body.String())
			t.Errorf("Failed to parse articles response JSON: %v", err)
			return
		}

		// Check the number of articles in the response
		count := len(responseArray)
		if count != len(withImages) {
			t.Errorf("Expected 4 articles; got %d", count)
		}

		if !reflect.DeepEqual(withImages, responseArray) {
			t.Errorf("ArticleDbHandler.AppendImage() = %v, want %v", responseArray, withImages)
			return
		}
	})

	t.Run("Find all articles without images", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		_, withoutImages := init(engine)

		allArticlesWithoutImagesReq, _ := http.NewRequest("GET", "/article?withImages=false", nil)
		allArticlesWithoutImagesResponse := httptest.NewRecorder()
		engine.ServeHTTP(allArticlesWithoutImagesResponse, allArticlesWithoutImagesReq)

		if allArticlesWithoutImagesResponse.Code != http.StatusOK {
			t.Errorf("Expected status code %d; got %d", http.StatusOK, allArticlesWithoutImagesResponse.Code)
		}

		var responseArray []string
		if err := json.Unmarshal(allArticlesWithoutImagesResponse.Body.Bytes(), &responseArray); err != nil {
			t.Errorf("Failed to parse articles response JSON: %v", err)
			return
		}

		count := len(responseArray)
		if count != len(withoutImages) {
			t.Errorf("Expected 3 articles; got %d", count)
		}

		if !reflect.DeepEqual(withoutImages, responseArray) {
			t.Errorf("ArticleDbHandler.AppendImage() = %v, want %v", responseArray, withoutImages)
			return
		}

	})
	t.Run("Find all articles", func(t *testing.T) {
		t.Parallel()
		engine, close := initializeServer(t)
		defer close()

		withImages, withoutImages := init(engine)

		// Retrieve all articles
		allArticlesReq, _ := http.NewRequest("GET", "/article", nil)
		allArticlesResponse := httptest.NewRecorder()
		engine.ServeHTTP(allArticlesResponse, allArticlesReq)

		if allArticlesResponse.Code != http.StatusOK {
			t.Errorf("Expected status code %d; got %d", http.StatusOK, allArticlesResponse.Code)
		}

		// Parse the response and check if there are exactly 3 articles
		var responseArray []string
		if err := json.Unmarshal(allArticlesResponse.Body.Bytes(), &responseArray); err != nil {
			fmt.Printf("Response Body: %s\n", allArticlesResponse.Body.String())
			t.Errorf("Failed to parse articles response JSON: %v", err)
			return
		}

		// Check the number of articles in the response
		count := len(responseArray)
		if count != len(withoutImages)+len(withImages) {
			fmt.Println("Articles:", responseArray)
			t.Errorf("Expected 7 articles; got %d", count)
		}
	})

}
