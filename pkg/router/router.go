package router

import (
	"errors"

	"github.com/gin-gonic/gin"
)

type ArticleController interface {
	Create(c *gin.Context)
	AttachImage(c *gin.Context)
	Find(c *gin.Context)
}

const (
	routeArticle      = "/article"
	routeImage        = "/image/:articleId"
	routeFindArticles = "/article"
)

type Router struct {
	ArticleCtrl ArticleController
	Engine      *gin.Engine
}

func NewRouter(articleCtrl ArticleController, engine *gin.Engine) *Router {
	return &Router{
		ArticleCtrl: articleCtrl,
		Engine:      engine,
	}
}

func (r *Router) Init() error {
	if err := r.addRoutes(); err != nil {
		return err
	}
	return nil
}

func (r *Router) addRoutes() error {
	if r.Engine == nil {
		return errors.New("engine is not initialized")
	}

	r.Engine.POST(routeArticle, r.ArticleCtrl.Create)
	r.Engine.POST(routeImage, r.ArticleCtrl.AttachImage)
	r.Engine.GET(routeFindArticles, r.ArticleCtrl.Find)

	return nil
}

func (r *Router) Run(addr ...string) {
	r.Engine.Run(addr...)
}
