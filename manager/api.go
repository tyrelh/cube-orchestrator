package manager

import "github.com/go-chi/chi/v5"

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}

type Api struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}
