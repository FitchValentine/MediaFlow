package ingestion

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// HTTPHandler exposes REST endpoints for the ingestion service.
type HTTPHandler struct {
	service      *Service
	logger       *zap.Logger
	maxSizeBytes int64
	formMemBytes int64
	router       chi.Router
}

// NewHTTPHandler constructs the HTTP handler and wires routes.
func NewHTTPHandler(service *Service, logger *zap.Logger, maxSizeBytes, formMemBytes int64) *HTTPHandler {
	h := &HTTPHandler{
		service:      service,
		logger:       logger,
		maxSizeBytes: maxSizeBytes,
		formMemBytes: formMemBytes,
	}
	h.buildRouter()
	return h
}

func (h *HTTPHandler) buildRouter() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(2 * time.Minute))

	r.Get("/healthz", h.handleHealth)
	r.Post("/api/v1/uploads", h.handleUpload)

	h.router = r
}

// Router exposes the configured chi router.
func (h *HTTPHandler) Router() http.Handler {
	return h.router
}

func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HTTPHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 0 && r.ContentLength > h.maxSizeBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}

	if err := r.ParseMultipartForm(h.formMemBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	if header.Size > h.maxSizeBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file exceeds max size limit")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	metadata := map[string]string{}
	for key, values := range r.MultipartForm.Value {
		if key == "file" {
			continue
		}
		if len(values) == 0 {
			continue
		}
		metadata[strings.ToLower(key)] = values[len(values)-1]
	}

	result, err := h.service.ProcessUpload(r.Context(), file, header.Size, UploadOptions{
		Filename:    header.Filename,
		ContentType: contentType,
		Metadata:    metadata,
	})
	if err != nil {
		h.logger.Error("upload failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"media_id":    result.MediaID,
		"object_key":  result.ObjectKey,
		"checksum":    result.Checksum,
		"size_bytes":  result.Size,
		"uploaded_at": result.UploadedAt,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}
