package handler

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"strings"

	"smog-detector/internal/model"
	"smog-detector/internal/service"
)

type Service interface {
	UpdateLocation(ctx context.Context, clientID string, lat, lon float64) (model.Dashboard, error)
	UpdateLocationByIP(ctx context.Context, clientID, ip string) (model.Dashboard, error)
	SelectCity(ctx context.Context, clientID, cityName string) (model.Dashboard, error)
	GetDashboard(ctx context.Context, clientID string) (model.Dashboard, error)
}

type Handler struct {
	service      Service
	template     *template.Template
	buildVersion string
}

func New(service Service, tmpl *template.Template, buildVersion string) *Handler {
	return &Handler{service: service, template: tmpl, buildVersion: buildVersion}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleIndex)
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/location", h.handleLocation)
	mux.HandleFunc("/api/location/ip", h.handleIPLocation)
	mux.HandleFunc("/api/city/select", h.handleCitySelect)
	mux.HandleFunc("/api/dashboard", h.handleDashboard)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	setNoStoreHeaders(w)
	if err := h.template.Execute(w, struct {
		BuildVersion string
	}{
		BuildVersion: h.buildVersion,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) handleLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ClientID string  `json:"client_id"`
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.ClientID) == "" {
		writeError(w, http.StatusBadRequest, "missing client_id")
		return
	}

	dashboard, err := h.service.UpdateLocation(r.Context(), strings.TrimSpace(req.ClientID), req.Lat, req.Lon)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handler) handleIPLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.ClientID) == "" {
		writeError(w, http.StatusBadRequest, "missing client_id")
		return
	}

	clientIP := extractClientIP(r)
	if clientIP == "" {
		writeError(w, http.StatusBadRequest, "cannot determine client ip")
		return
	}

	dashboard, err := h.service.UpdateLocationByIP(r.Context(), strings.TrimSpace(req.ClientID), clientIP)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handler) handleCitySelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ClientID string `json:"client_id"`
		CityName string `json:"city_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.ClientID) == "" {
		writeError(w, http.StatusBadRequest, "missing client_id")
		return
	}
	if strings.TrimSpace(req.CityName) == "" {
		writeError(w, http.StatusBadRequest, "missing city_name")
		return
	}

	dashboard, err := h.service.SelectCity(r.Context(), strings.TrimSpace(req.ClientID), strings.TrimSpace(req.CityName))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "missing client_id")
		return
	}
	dashboard, err := h.service.GetDashboard(r.Context(), clientID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrLocationRequired):
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "location_required",
			"message": "请先定位或手动选择城市",
		})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":   "request_failed",
			"message": err.Error(),
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	setNoStoreHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func extractClientIP(r *http.Request) string {
	for _, key := range []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(key))
		if value == "" {
			continue
		}
		if key == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if parsed := normalizeIP(value); parsed != "" {
			return parsed
		}
	}
	if parsed := normalizeIP(r.RemoteAddr); parsed != "" {
		return parsed
	}
	return ""
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return ""
}
