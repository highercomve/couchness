package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/highercomve/couchness/web/jobs"
	viewmodels "github.com/highercomve/couchness/web/models"
	"github.com/highercomve/couchness/web/repository"
	"github.com/highercomve/couchness/web/services"
	"github.com/highercomve/couchness/web/templates"
)

// Handler serves the web UI. One instance is shared by all routes.
type Handler struct {
	Library  *services.Library
	Shows    *services.Shows
	Movies   *services.Movies
	Config   *repository.Config
	Jobs     *jobs.Runner
	Renderer *templates.Renderer
	Version  string
}

// Deps is what New needs; all fields required.
type Deps struct {
	Library  *services.Library
	Shows    *services.Shows
	Movies   *services.Movies
	Config   *repository.Config
	Jobs     *jobs.Runner
	Renderer *templates.Renderer
	Version  string
}

// New creates a Handler from its dependencies.
func New(deps Deps) *Handler {
	return &Handler{
		Library:  deps.Library,
		Shows:    deps.Shows,
		Movies:   deps.Movies,
		Config:   deps.Config,
		Jobs:     deps.Jobs,
		Renderer: deps.Renderer,
		Version:  deps.Version,
	}
}

// Register mounts every route on mux using Go 1.22 method+path patterns.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /static/", http.StripPrefix(path.Join("/static")+"/", staticFiles()))
	mux.HandleFunc("GET /{$}", h.root)
	mux.HandleFunc("GET /shows", h.shows)
	mux.HandleFunc("POST /scan", h.scan)
	mux.HandleFunc("POST /update-all", h.updateAll)
	mux.HandleFunc("GET /shows/{id}", h.show)
	mux.HandleFunc("POST /shows/{id}/download", h.showDownload)
	mux.HandleFunc("POST /shows/{id}/update", h.showUpdate)
	mux.HandleFunc("POST /shows/{id}/disable", h.showDisable)
	mux.HandleFunc("POST /shows/{id}/delete", h.showDelete)
	mux.HandleFunc("POST /shows/{id}/config", h.showConfig)
	mux.HandleFunc("GET /shows/{id}/identify", h.identifySearch)
	mux.HandleFunc("POST /shows/{id}/identify", h.identify)
	mux.HandleFunc("GET /movies", h.movies)
	mux.HandleFunc("GET /movies/list", h.moviesList)
	mux.HandleFunc("POST /movies/delete", h.movieDelete)
	mux.HandleFunc("GET /movies/search", h.movieSearch)
	mux.HandleFunc("GET /movies/torrents", h.movieTorrents)
	mux.HandleFunc("POST /movies/download", h.movieDownload)
	mux.HandleFunc("GET /config", h.configPage)
	mux.HandleFunc("POST /config", h.configSave)
	mux.HandleFunc("POST /media-servers/test", h.mediaServersTest)
	mux.HandleFunc("POST /media-servers/refresh", h.mediaServersRefresh)
	mux.HandleFunc("GET /jobs", h.jobsPage)
	mux.HandleFunc("GET /jobs/{id}", h.job)
}

// BasicAuth wraps next with HTTP basic auth when auth ("user:password") is non-empty.
// Compares sha256 digests of user and password with crypto/subtle.ConstantTimeCompare.
// On failure: header WWW-Authenticate: Basic realm="couchness" and 401.
// Empty auth returns next unchanged.
func BasicAuth(next http.Handler, auth string) http.Handler {
	if auth == "" {
		return next
	}
	parts := strings.SplitN(auth, ":", 2)
	if len(parts) != 2 {
		return next
	}
	expectedUser := sha256.Sum256([]byte(parts[0]))
	expectedPass := sha256.Sum256([]byte(parts[1]))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"couchness\"")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		givenUser := sha256.Sum256([]byte(user))
		givenPass := sha256.Sum256([]byte(pass))
		if subtle.ConstantTimeCompare(expectedUser[:], givenUser[:]) != 1 ||
			subtle.ConstantTimeCompare(expectedPass[:], givenPass[:]) != 1 {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"couchness\"")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (h *Handler) page(title, active string, data interface{}) viewmodels.Page {
	var running *jobs.Snapshot
	if job := h.Jobs.Running(); job != nil {
		snap := job.Snapshot()
		running = &snap
	}
	return viewmodels.Page{
		Title:      title,
		Active:     active,
		Version:    h.Version,
		Config:     h.Config.Get(),
		RunningJob: running,
		Data:       data,
	}
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, p viewmodels.Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.Renderer.Render(w, page, p); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) partial(w http.ResponseWriter, status int, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.Renderer.Partial(w, name, data); err != nil {
		log.Printf("partial error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) flash(w http.ResponseWriter, status int, kind, message string) {
	h.partial(w, status, "flash", viewmodels.Flash{Kind: kind, Message: message})
}

func (h *Handler) startJob(w http.ResponseWriter, name string, task jobs.Task) {
	job, err := h.Jobs.Start(name, task)
	if errors.Is(err, jobs.ErrBusy) {
		h.flash(w, http.StatusConflict, "error", err.Error())
		return
	}
	if err != nil {
		h.flash(w, http.StatusInternalServerError, "error", err.Error())
		return
	}
	h.partial(w, http.StatusOK, "job", job.Snapshot())
}

func (h *Handler) fail(w http.ResponseWriter, status int, err error) {
	h.flash(w, status, "error", err.Error())
}

func errStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := strings.ToLower(err.Error())
	if strings.HasPrefix(msg, "a ") || strings.HasPrefix(msg, "no ") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
