package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	bulwark "github.com/latebit-io/bulwark-auth-guard"
	accountsapi "github.com/latebit-io/bulwarkauthadmin/api/accounts"
	accountsrbacapi "github.com/latebit-io/bulwarkauthadmin/api/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/api/health"
	bulwarkauthmiddleware "github.com/latebit-io/bulwarkauthadmin/api/middleware"
	rbacapi "github.com/latebit-io/bulwarkauthadmin/api/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	adminAccount "github.com/latebit-io/bulwarkauthadmin/internal/accounts/admin"
	accountsRbac "github.com/latebit-io/bulwarkauthadmin/internal/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/version"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If the version flag is passed, print the version and exit
	if *versionFlag {
		fmt.Println(version.GetVersionInfo())
		os.Exit(0)
	}

	logger := getLogger()
	fmt.Println(`
 ____  _  _  __    _  _   __   ____  __ _   __   _  _  ____  _  _
(  _ \/ )( \(  )  / )( \ / _\ (  _ \(  / ) / _\ / )( \(_  _)/ )( \
 ) _ () \/ (/ (_/\\ /\ //    \ )   / )  ( /    \) \/ (  )(  ) __ (
(____/\____/\____/(_/\_)\_/\_/(__\_)(__\_)\_/\_/\____/ (__) \_)(_/admin v1.0.0`)
	err := godotenv.Load()
	if err != nil {
		logger.Warn("no .env file loading from system")
	}

	config, err := NewAppConfig()
	if err != nil {
		panic(err)
	}

	service := echo.New()
	service.HideBanner = true
	logger.Info("connecting to mongodb: ", "uri", config.DbConnection, "db", config.DbNameSeed)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(config.DbConnection))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()
	httpClient := &http.Client{}
	bulwarkGuard := bulwark.NewGuard(config.BulwarkAuthUrl, httpClient)
	jwt := bulwarkauthmiddleware.NewJWTMiddleware(bulwarkGuard)
	mongodb := client.Database("bulwarkauth" + config.DbNameSeed)
	accountRepository := accounts.NewMongoDBAccountRepository(mongodb)
	accountsManagmentService := accounts.NewAccountManagementServiceDefault(accountRepository)
	accountsHandler := accountsapi.NewAccountHandler(accountsManagmentService)
	accountsapi.AccountRoutesV1(service, accountsHandler)

	permissionsRepository := rbac.NewMongoDBPermissionsRepository(mongodb)
	rolesRepository := rbac.NewMongoDBRolesRepository(mongodb)
	roleService := rbac.NewRoleServiceDefault(rolesRepository)
	permissionService := rbac.NewPermissionServiceDefault(permissionsRepository)
	rbacHandler := rbacapi.NewRbacHandler(roleService, permissionService)
	rbacapi.RbacRoutesV1(service, rbacHandler)

	accountsRBAC := accountsRbac.NewAccountRBACServiceDefault(accountRepository, permissionService, roleService)
	accountsRbacHandler := accountsrbacapi.NewAccountRBACHandler(accountsRBAC)
	accountsrbacapi.AccountRBACRoutesV1(service, accountsRbacHandler)

	adminAccountService := adminAccount.NewAdminAccountsServiceDefault(
		accountRepository,
		rolesRepository,
		permissionsRepository,
		accountsRBAC,
		bulwarkGuard,
	)

	err = adminAccountService.CreateInternalRoles(context.Background())
	if err != nil {
		logger.Error("failed to create internal roles", "error", err)
		panic(err)
	}
	defaultAdminAccount := config.AdminAccount
	defaultAdminPassword := config.AdminAccountPassword
	if defaultAdminAccount != "" {
		err = adminAccountService.RegisterAccount(context.Background(), defaultAdminAccount, defaultAdminPassword)
		if err != nil {
			logger.Error("could configure default admin account", "error", err)
			panic(err)
		}
	}

	healthHandler := health.NewHealthHandler()
	health.HealthRoutes(service, healthHandler)
	corsSetting(service, config, logger)
	service.Use(jwt.Jwt)

	if err := service.Start(fmt.Sprintf(":%d", config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err.Error())
	}
}

func getLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stderr, nil)
	logger := slog.New(jsonHandler)
	return logger
}

func corsSetting(service *echo.Echo, config *AppConfig, logger *slog.Logger) {
	if !config.CORSEnabled {
		return
	}

	service.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.AllowedOrigins,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
	}))
	logger.Info("cors enabled")
	logger.Info("cors allowed origins", "origins", config.AllowedOrigins)
}
