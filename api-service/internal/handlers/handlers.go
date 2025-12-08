// File: internal/handlers/handlers.go
package handlers

import (
	"time"

	"azlo-goboiler/internal/config"
	"azlo-goboiler/internal/core"
)

type Handlers struct {
	app     *config.Application
	service core.UserService
}

func New(app *config.Application, service core.UserService) *Handlers {
	return &Handlers{
		app:     app,
		service: service,
	}
}

var startTime = time.Now()
