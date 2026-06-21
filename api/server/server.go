package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"vms/api/auth"
	"vms/api/handlers"
	"vms/api/record"
	"vms/api/stream"

	"github.com/gorilla/mux"
)

type HTTPServer struct {
	HTTPHandlers *handlers.HTTPHandlers
}

func NewHTTPServer(httpHandlers *handlers.HTTPHandlers) *HTTPServer {
	return &HTTPServer{
		HTTPHandlers: httpHandlers,
	}
}

func (s *HTTPServer) StartServer() {
	router := mux.NewRouter()

	router.Path("/api/login").Methods("POST").HandlerFunc(auth.LoginHandler)

	api := router.PathPrefix("/api").Subrouter()
	api.Use(auth.AuthMiddleware)

	api.Path("/cameras").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetCameras)
	api.Path("/cameras").Methods("POST").HandlerFunc(s.HTTPHandlers.HandleCreateCamera)
	api.Path("/cameras/{id}").Methods("DELETE").HandlerFunc(s.HTTPHandlers.HandleDeleteCamera)
	api.Path("/cameras/{id}/record/start").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleStartRecording)
	api.Path("/cameras/{id}/record/stop").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleStopRecording)
	api.Path("/cameras/{id}/stream/start").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleStartStream)
	api.Path("/cameras/{id}/stream/stop").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleStopStream)
	api.Path("/stream/{id}").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetStream)
	api.Path("/archive/{id}/{date}").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetArchiveFiles)

	recordingsServer := http.FileServer(http.Dir(record.RecordingsDir))
	api.PathPrefix("/recordings/").Handler(http.StripPrefix("/api/recordings/", recordingsServer))

	frontEnd := http.FileServer(http.Dir("../frontend"))
	router.PathPrefix("/").Handler(frontEnd)

	ipAddr := os.Getenv("IP_ADDR")
	port := os.Getenv("PORT")

	server := http.Server{
		Addr:    ipAddr + port,
		Handler: router,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Println("HTTP Server started at:", ipAddr+port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP Server error:", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down the HTTP Server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("Error during shutdown:", err)
	}

	wg := sync.WaitGroup{}

	wg.Go(record.Manager.StopAll)
	wg.Go(stream.Manager.StopAll)

	wg.Wait()

	log.Println("HTTP Server closed...")
}
