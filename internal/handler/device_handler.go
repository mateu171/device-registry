package handler

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"device-registry/internal/domain"
	"device-registry/internal/repository"

	"github.com/go-chi/chi/v5"
)

type DeviceHandler struct {
	repo      repository.DeviceRepository
	idCounter atomic.Uint64
}

func NewDeviceHandler(repo repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *DeviceHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	devices := h.repo.GetAll()
	writeJSON(w, http.StatusOK, devices)
}

func (h *DeviceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	device, exists := h.repo.GetByID(id)
	if !exists {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var device domain.Device

	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	device.Name = strings.TrimSpace(device.Name)
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Type = strings.TrimSpace(device.Type)

	if device.Name == "" || device.IPAddress == "" || device.Type == "" {
		http.Error(w, "name, ip_address and type are required", http.StatusBadRequest)
		return
	}

	device.ID = fmt.Sprintf("dev-%d", h.idCounter.Add(1))
	device.Status = strings.TrimSpace(device.Status)
	if device.Status == "" {
		device.Status = "offline"
	}
	device.UpdatedAt = time.Now().UTC()

	if err := h.repo.Create(device); err != nil {
		http.Error(w, "Could not create device", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, device)
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var device domain.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	device.Name = strings.TrimSpace(device.Name)
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Type = strings.TrimSpace(device.Type)

	if device.Name == "" || device.IPAddress == "" || device.Type == "" {
		http.Error(w, "name, ip_address and type are required", http.StatusBadRequest)
		return
	}

	if _, exists := h.repo.GetByID(id); !exists {
		http.NotFound(w, r)
		return
	}

	device.ID = id
	device.Status = strings.TrimSpace(device.Status)
	device.UpdatedAt = time.Now().UTC()

	if device.Status == "" {
		device.Status = "offline"
	}

	if !h.repo.Update(id, device) {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !h.repo.Delete(id) {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sr *statusRecorder) WriteHeader(status int) {
	if sr.wroteHeader {
		return
	}

	sr.status = status
	sr.wroteHeader = true
	sr.ResponseWriter.WriteHeader(status)
}

func (sr *statusRecorder) Write(data []byte) (int, error) {
	if !sr.wroteHeader {
		sr.WriteHeader(http.StatusOK)
	}

	return sr.ResponseWriter.Write(data)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		clientIP := r.RemoteAddr

		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			clientIP = host
		}

		fmt.Printf(
			"[HTTP] %s - %s %s -> %d (%v)\n",
			clientIP,
			r.Method,
			r.URL.RequestURI(),
			recorder.status,
			duration.Round(time.Microsecond),
		)
	})
}
