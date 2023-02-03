package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	httpRequest "net/http"
	"net/http/httputil"
	"net/url"
	"synchrodb/gateway/config"
	"synchrodb/gateway/pkg/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func apiMiddleware(fc echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		context.Request().Header.Set("gateway", "true")
		fmt.Println("HAIII TESTs222")
		return fc(context)

	}
}

var ErrorUnAuthorized = errors.New("unauthorized")

func AllowLinkRequestsMiddleware(project httpRequest.Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {

			if ctx.Request().Method == "LINK" || ctx.Request().Method == "UNLINK" {
				return echo.WrapHandler(project)(ctx)
			}

			err := next(ctx)
			return err
		}
	}
}

func queryProxy(ctx echo.Context) error {
	projectId := ctx.Param("projectId")
	tableId := ctx.Param("tableId")
	filter := ctx.QueryString()

	url := fmt.Sprintf(config.CoreAPIURL+"/api/v1/projects/%s/tables/%s/records?%s", projectId, tableId, filter)
	method := "GET"

	client := &httpRequest.Client{}
	req, err := httpRequest.NewRequest(method, url, nil)

	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	var records interface{}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&records)

	if err != nil {
		return err
	}

	ctx.Response().Header().Set("X-Total-Count", response.Header.Get("X-Total-Count"))
	ctx.Response().Header().Set("X-Pagination-Limit", response.Header.Get("X-Pagination-Limit"))
	ctx.Response().Header().Set("X-Pagination-Skip", response.Header.Get("X-Pagination-Skip"))
	ctx.Response().Header().Set("Access-Control-Expose-Headers", "X-Total-Count, X-Pagination-Limit, X-Pagination-Skip")

	return ctx.JSON(response.StatusCode, records)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	e := http.NewEchoHTTPServer()
	e.Echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{httpRequest.MethodGet, httpRequest.MethodHead, httpRequest.MethodPut, httpRequest.MethodPatch, httpRequest.MethodPost, httpRequest.MethodDelete, "LINK", "UNLINK"},
	}))

	queryService, err := url.Parse(config.QueryAPIURL)
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	coreService, err := url.Parse(config.CoreAPIURL + "/api/v1")
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	uploadService, err := url.Parse(config.UploadAPIURL + "/storage")
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	queryServiceMonitoring, err := url.Parse(config.QueryAPIURL + "/monitoring")
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	queryServiceMultiplayer, err := url.Parse(config.BridgeAPIURL + "/multiplayer")
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	userService, err := url.Parse(config.UserAPIURL)
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	databaseService, err := url.Parse(config.DatabaseAPIURL)
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	clusterService, err := url.Parse(config.ClusterAPIURL)
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	automationService, err := url.Parse(config.AutomationAPIURL + "/api/v1")
	if err != nil {
		e.Echo.Logger.Fatal(err)
	}

	projectProxy := httpRequest.StripPrefix("/api/v1", httputil.NewSingleHostReverseProxy(coreService))

	e.Echo.Use(AllowLinkRequestsMiddleware(projectProxy))

	e.SetupRoutes(func(s *http.Server) {
		s.Echo.Pre(middleware.RemoveTrailingSlash())
		authRoutes := s.Echo.Group("/api/v1")
		authRoutes.Use(authMiddleware)

		v1 := s.Echo.Group("/api/v1")
		v1.Any("/query/:projectId/:tableId", queryProxy)

		monitoringProxy := createProxy("/api/v1/monitoring", queryServiceMonitoring)
		authRoutes.Any("/monitoring/*", echo.WrapHandler(monitoringProxy))

		multiplayerProxy := createProxy("/api/v1/multiplayer", queryServiceMultiplayer)
		authRoutes.Any("/multiplayer/:projectId/*", echo.WrapHandler(multiplayerProxy))

		// used for finding blocks without projectId
		authRoutes.Any("/blocks/*", echo.WrapHandler(projectProxy))

		databaseProxy := createProxy("/api/v1", databaseService)
		v1.GET("/projects/:projectId/downloadExportedTable*", echo.WrapHandler(databaseProxy))

		automationProxy := createProxy("/api/v1", automationService)
		authRoutes.Any("/projects/:projectId/workflows*", echo.WrapHandler(automationProxy))
		v1.POST("/hooks/:workflowId", echo.WrapHandler(automationProxy))

		authRoutes.Any("/projects/:projectId", echo.WrapHandler(projectProxy))
		authRoutes.Any("/projects/:projectId/*", echo.WrapHandler(projectProxy))

		authRoutes.POST("/projects/:projectId/importTable", echo.WrapHandler(databaseProxy))
		authRoutes.POST("/projects/:projectId/exportTable", echo.WrapHandler(databaseProxy))

		apiProxy := createProxy("", queryService)
		authRoutes.POST("/projects/:projectId/fields/check", echo.WrapHandler(apiProxy))

		authRoutes.Any("/backups*", echo.WrapHandler(databaseProxy))

		authRoutes.POST("/clusters", echo.WrapHandler(projectProxy))
		authRoutes.GET("/clusters/:id", echo.WrapHandler(projectProxy))
		authRoutes.Any("/clusters/:id/apps*", echo.WrapHandler(projectProxy))
		authRoutes.POST("/clusters/:id/deploy*", echo.WrapHandler(projectProxy))

		// apps builder
		v1.GET("/apps/:appName", echo.WrapHandler(projectProxy))

		pluginProxy := createProxy("/api/v1", coreService)
		authRoutes.GET("/plugins", echo.WrapHandler(pluginProxy))

		authRoutes.Any("/:apiKey/*", echo.WrapHandler(apiProxy), apiMiddleware)

		// share
		authRoutes.GET("/shares/*", echo.WrapHandler(projectProxy))

		uploadProxy := createProxy("/api/v1/storage", uploadService)
		authRoutes.Any("/storage/*", echo.WrapHandler(uploadProxy))

		clusterProxy := createProxy("/api/v1", clusterService)
		authRoutes.GET("/clusters*", echo.WrapHandler(clusterProxy))

		userProxy := createProxy("/api/v1", userService)

		v1.POST("/register", echo.WrapHandler(userProxy))

		v1.POST("/login", echo.WrapHandler(userProxy))

		authRoutes.Any("/users*", echo.WrapHandler(userProxy))

		authRoutes.Any("/settings*", echo.WrapHandler(userProxy))
	})

	e.Start("0.0.0.0:" + config.HttpPort)
}

func createProxy(prefix string, url *url.URL) httpRequest.Handler {
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ErrorHandler = func(rw httpRequest.ResponseWriter, req *httpRequest.Request, err error) {

		userId := req.Header.Get("x-user-id")
		log.Printf("http: proxy error: %v", err)
		log.Printf("%v %v", req.Method, req.URL)
		log.Printf("X-User-ID: %v\n\n", userId)
		rw.WriteHeader(httpRequest.StatusBadGateway)
	}

	return httpRequest.StripPrefix(prefix, proxy)
}

func authMiddleware(fc echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		client := httpRequest.Client{}
		userServiceURL, err := url.Parse(config.UserAPIURL)
		if err != nil {
			return err
		}

		req, err := httpRequest.NewRequest("GET", userServiceURL.String()+"/validate-key", nil)
		if err != nil {
			log.Println(err)
			return err
		}

		apiKey := ctx.Request().Header.Get("x-api-key")

		if apiKey == "" {
			apiKey = ctx.QueryParam("api-key")
		}

		req.Header = ctx.Request().Header
		req.Header.Set("X-Api-Key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return err
		}

		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body), "BODY")

		defer resp.Body.Close()

		if resp.StatusCode == httpRequest.StatusOK && resp.Header.Get("X-User-ID") != "" {
			ctx.Request().Header.Set("X-User-ID", resp.Header.Get("X-User-ID"))

			return fc(ctx)
		}

		return echo.NewHTTPError(httpRequest.StatusUnauthorized, "please provide valid credentials")
	}
}
