package routes

import (
	"echolabstack/ratelimitter"
	"echolabstack/service"

	"github.com/labstack/echo/v4"
)

func Echoroutes(e *echo.Echo) {

	e.Static("/static", "static")
	//ratelimitting api
	e.GET("/users", service.NewAPI, ratelimitter.CombinedRateLimiter())
	e.GET("/pdfapi", service.PdfAPI, ratelimitter.CombinedRateLimiter())
	//emailid
	e.POST("/subscribe", service.EmailIDAPI, ratelimitter.CombinedRateLimiter())
	e.Static("/static", "./content")
	//recapcha
	e.POST("/submit", service.CapcheAPI)
	e.GET("/files", service.Filehandler)
	e.GET("/refresh/api", service.Drive)
}
