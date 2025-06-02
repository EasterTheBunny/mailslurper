package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/mailslurper/mailslurper/v2/internal/handlers"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/middleware"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/ui"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authfactory"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authscheme"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/jwt"
	"github.com/mailslurper/mailslurper/v2/web"
)

type Persistance interface {
	handlers.MailRemover
	handlers.MailCounter
	handlers.MailCollectionGetter
	middleware.MailGetter
	handlers.MailMessageRawGetter
}

type APIRouter struct {
	Version    string
	Data       Persistance
	Config     *io.Config
	JWTService *jwt.JWTService
	Logger     *log.Logger
}

func (r *APIRouter) Routes() http.Handler {
	router := chi.NewRouter()

	// set CORS headers early and short circuit the response loop
	router.Use(middleware.SetCORSHeaders)

	// TODO: /login

	router.Use(middleware.JWTAuth(r.Config, r.JWTService, r.Logger))
	// TODO: /logout
	router.Get("/version", handlers.Version(r.Version, r.Logger))
	router.Get("/pruneoptions", handlers.GetPruneOptions(r.Logger))
	router.Get("/mailcount", handlers.GetMailCount(r.Data, r.Logger))

	// setup mail routes
	router.Route("/mail", r.MailRoutes())

	return router
}

func (r *APIRouter) MailRoutes() func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handlers.GetMailCollection(r.Data, r.Logger)) // bulk get
		router.Delete("/", handlers.DeleteMail(r.Data, r.Logger))     // bulk delete

		router.Route(fmt.Sprintf("/{%s}", requests.MailIDPathParam), r.MailSubRoutes())
	}
}

func (r *APIRouter) MailSubRoutes() func(chi.Router) {
	return func(router chi.Router) {
		router.Use(middleware.MailCtx(r.Data, chi.URLParam, r.Logger))

		router.Get("/", handlers.GetMail(r.Data, r.Logger))
		router.Get("/message", handlers.GetMailMessage(r.Logger))
		router.Get("/messageraw", handlers.GetMailMessageRaw(r.Data, r.Logger))

		router.Route("/attachment", r.MailDetailSubRoutes())
	}
}

func (r *APIRouter) MailDetailSubRoutes() func(chi.Router) {
	return func(router chi.Router) {
		router.Route(fmt.Sprintf("/{%s}", requests.MailAttachmentIDPathParam), r.MailAttachmentSubRoutes())
	}
}

func (r *APIRouter) MailAttachmentSubRoutes() func(chi.Router) {
	return func(router chi.Router) {
		router.Get("/", handlers.DownloadAttachment(r.Logger))
	}
}

type Router struct {
	Version     string
	Config      *io.Config
	Renderer    *ui.TemplateRenderer
	AuthFactory *authfactory.AuthFactory
	Logger      *log.Logger
	MountPaths  map[string]http.Handler
}

func (r *Router) Routes() http.Handler {
	router := chi.NewRouter()

	// mount static assets before authorization
	fs := http.FileServer(http.FS(web.StaticWebAssets))
	router.Mount("/www", http.StripPrefix("/", fs))

	// TODO: login/logout features should not be stateful actions on the server
	if r.Config.AuthenticationScheme != authscheme.NONE {
		router.Get("/login", handlers.PageLogin(r.Config, r.Renderer, chi.URLParam, r.Logger))
		router.Post("/perform-login", handlers.ActionPerformLogin(r.AuthFactory, r.Logger))
		router.Get("/logout", handlers.ActionLogout())
	}

	router.Get("/", handlers.PageIndex(r.Config, r.Renderer, r.Logger))
	router.Get("/admin", handlers.PageAdmin(r.Config, r.Renderer, r.Logger))
	router.Get("/savedsearches", handlers.PageManageSavedSearches(r.Config, r.Renderer, r.Logger))
	router.Get("/servicesettings", handlers.GetServiceSettings(r.Version, r.Config, r.Logger))
	router.Get("/version", handlers.GetVersion(r.Version, r.Logger))
	router.Get("/masterversion", handlers.GetVersionFromMaster(r.Version, r.Logger))
	router.Post("/theme", handlers.ApplyTheme(r.Config, r.Renderer, r.Logger))

	for path, mount := range r.MountPaths {
		router.Mount(path, mount)
	}

	return router
}
