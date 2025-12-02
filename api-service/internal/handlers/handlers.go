// File: internal/handlers/handlers.go
package handlers

import (
	"time"

	"azlo-goboiler/internal/config"
)

type Handlers struct {
	app *config.Application
}

func New(app *config.Application) *Handlers {
	return &Handlers{app: app}
}

var startTime = time.Now()
