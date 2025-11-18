package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/health"
	"github.com/latebit-io/bulwarkauthadmin/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If the version flag is passed, print the version and exit
	if *versionFlag {
		fmt.Println(version.GetVersionInfo())
		os.Exit(0)
	}

	//logger := getLogger()
	fmt.Println(`
 ____  _  _  __    _  _   __   ____  __ _   __   _  _  ____  _  _
(  _ \/ )( \(  )  / )( \ / _\ (  _ \(  / ) / _\ / )( \(_  _)/ )( \
 ) _ () \/ (/ (_/\\ /\ //    \ )   / )  ( /    \) \/ (  )(  ) __ (
(____/\____/\____/(_/\_)\_/\_/(__\_)(__\_)\_/\_/\____/ (__) \_)(_/admin v1.0.0`)
	err := godotenv.Load()
	// if err != nil {
	// 	logger.Warn("no .env file loading from system")
	// }

	config, err := NewAppConfig()
	if err != nil {
		panic(err)
	}

	service := echo.New()
	service.HideBanner = true
	//logger.Info("connecting to mongodb: ", "uri", config.DbConnection, "db", config.DbNameSeed)
	//client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(config.DbConnection))
	if err != nil {
		panic(err)
	}

	defer func() {
		// if err := client.Disconnect(context.Background()); err != nil {
		// 	panic(err)
		// }
	}()

	//mongodb := client.Database("bulwarkauth" + config.DbNameSeed)

	healthHandler := health.NewHealthHandler()
	health.HealthRoutes(service, healthHandler)

	if err := service.Start(fmt.Sprintf(":%d", config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		//logger.Error(err.Error())
	}
}
