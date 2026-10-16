package main

import (
	"fmt"
	"log"
	"net/http"

	"device-registry/internal/handler"
	"device-registry/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	repo := repository.NewMemoryDeviceRepository()
	deviceHandler := handler.NewDeviceHandler(repo)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(handler.LoggingMiddleware)

	r.Route("/api/v1/devices", func(r chi.Router) {
		r.Get("/", deviceHandler.GetAll)
		r.Post("/", deviceHandler.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", deviceHandler.GetByID)
			r.Put("/", deviceHandler.Update)
			r.Delete("/", deviceHandler.Delete)
		})
	})

	fmt.Println("Device Registry API")
	fmt.Println("[SERVER] In-memory repository initialized")
	fmt.Println("[SERVER] Chi routes configured")
	fmt.Println("[SERVER] Logging middleware enabled")
	fmt.Println("[SERVER] Server starting on :8080")

	log.Fatal(http.ListenAndServe(":8080", r))
}
