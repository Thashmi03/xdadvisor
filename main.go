package main

import (
	"database/sql"
	"echolabstack/routes"
	"echolabstack/service"

	"github.com/labstack/echo/v4"
)

var db *sql.DB
var err error

func main() {
	e := echo.New()

	routes.Echoroutes(e)

	service.Database()

	e.Logger.Fatal(e.Start(":8080"))
}
