package microscope

import (
	"io/fs"
	"net/http"
	"strings"
)

func (h *Handler) registerSPARoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/assets/", h.serveStatic(prefix))
	mux.HandleFunc("GET "+prefix+"/entries/{id}", h.serveSPA)
	mux.HandleFunc("GET "+prefix+"/settings", h.serveSPA)
	mux.HandleFunc("GET "+prefix, h.serveSPA)
	mux.HandleFunc("GET "+prefix+"/", h.serveSPA)
}

func (h *Handler) serveSPA(w http.ResponseWriter, r *http.Request) {
	prefix := h.Hub.cfg.pathPrefix()
	if r.URL.Path != prefix && r.URL.Path != prefix+"/" && r.URL.Path != prefix+"/settings" && !strings.HasPrefix(r.URL.Path, prefix+"/entries/") {
		http.NotFound(w, r)
		return
	}
	h.serveFile(w, r, "index.html")
}

func (h *Handler) serveStatic(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, prefix+"/")
		h.serveFile(w, r, rel)
	}
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	sub, err := fs.Sub(uiFS, "ui/dist")
	if err != nil {
		http.Error(w, "ui not available", http.StatusInternalServerError)
		return
	}
	data, err := fs.ReadFile(sub, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(name, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
