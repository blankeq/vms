package server

import (
	"errors"
	"net/http"
	"os"
	"vms/api/auth"
	"vms/api/handlers"

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

func (s *HTTPServer) StartServer() error {
	router := mux.NewRouter()

	router.Path("/api/login").Methods("POST").HandlerFunc(auth.LoginHandler)

	api := router.PathPrefix("/api").Subrouter()
	api.Use(auth.AuthMiddleware)

	api.Path("/cameras").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetCameras)
	api.Path("/cameras").Methods("POST").HandlerFunc(s.HTTPHandlers.HandleCreateCamera)
	api.Path("/cameras/{id}").Methods("DELETE").HandlerFunc(s.HTTPHandlers.HandleDeleteCamera)
	api.Path("/archive/{id}").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetArchiveFiles)
	api.Path("/stream/{id}").Methods("GET").HandlerFunc(s.HTTPHandlers.HandleGetStream)

	fileServer := http.FileServer(http.Dir("./recordings"))
	api.PathPrefix("/stream/").Handler(http.StripPrefix("api/stream/", fileServer))

	frontEnd := http.FileServer(http.Dir("./frontend"))
	router.PathPrefix("/").Handler(frontEnd)

	ipAddr := os.Getenv("IP_ADDR")
	port := os.Getenv("PORT")

	if err := http.ListenAndServe(ipAddr+port, router); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}

	return nil
}
