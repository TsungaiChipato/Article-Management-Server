package main

import (
	"article-management-service/pkg/controller"
	db "article-management-service/pkg/db"
	"article-management-service/pkg/env"
	"article-management-service/pkg/router"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func main() {
	cfg, err := env.Load()
	if err != nil {
		panic(err)
	}

	mm := db.MockMongo{}
	uri, err := mm.HostMemoryDb(cfg.MongodPath)
	if err != nil {
		panic(err)
	}
	defer mm.Close()

	conn := db.Connection{}
	err = conn.Connect(uri)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

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
		ImageDirectory:     "images",
		GenerateIdentifier: func() string { return uuid.New().String() },
		Validate:           validate,
	}

	router := router.NewRouter(articleController, engine)

	router.Init()
	router.Run(":5000")
}
