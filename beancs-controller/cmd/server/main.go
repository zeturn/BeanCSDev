package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/zeturn/beancs-controller/internal/basaltpass"
	"github.com/zeturn/beancs-controller/internal/config"
	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/handler"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/migration"
	"github.com/zeturn/beancs-controller/internal/service"
	"github.com/zeturn/beancs-controller/internal/web"
)

func main() {
	log, _ := zap.NewProduction()
	defer func() { _ = log.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}
	log.Info("config loaded", zap.String("version", cfg.Version))
	db, err := openDatabase(log, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("connect database", zap.Error(err))
	}
	log.Info("database connected")
	if err := migration.AutoMigrate(db); err != nil {
		log.Fatal("migrate database", zap.Error(err))
	}
	log.Info("database migrated")
	cipher, err := cryptoutil.NewAESGCMCipher(cfg.EncryptionKey)
	if err != nil {
		log.Fatal("init cipher", zap.Error(err))
	}

	v := validator.New()
	k8sManager := k8s.NewManager(cfg)
	registry := basaltpass.NewClientRegistry(db, cipher, cfg)
	credentialSvc := service.NewCredentialService(db, cipher, cfg)
	apiKeySvc := service.NewAPIKeyService(db)
	registryImageSvc := service.NewContainerRegistryService(db, cipher)
	quotaSvc := service.NewQuotaService(db)
	dnsSvc := service.NewDNSService(cfg.IngressIP)
	gitopsSvc := service.NewGitOpsService(db, credentialSvc)
	buildSvc := service.NewGitHubBuildService(db, cfg, credentialSvc, gitopsSvc)
	projectSvc := service.NewProjectService(db, credentialSvc, quotaSvc, dnsSvc, gitopsSvc, buildSvc, k8sManager, registry, cipher, cfg)
	processSvc := service.NewProcessService(db, buildSvc, credentialSvc, gitopsSvc, dnsSvc, k8sManager)
	deploymentSvc := service.NewDeploymentService(db, buildSvc, credentialSvc, gitopsSvc, processSvc)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				code = fiberErr.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": "request failed"})
		},
	})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{AllowOrigins: cfg.CORSOrigins}))

	registerAPI(app.Group("/v1/api"), cfg, db, registry, credentialSvc, apiKeySvc, registryImageSvc, projectSvc, deploymentSvc, processSvc, k8sManager, v)
	registerAPI(app.Group("/api/v1"), cfg, db, registry, credentialSvc, apiKeySvc, registryImageSvc, projectSvc, deploymentSvc, processSvc, k8sManager, v)

	app.Get("/assets/*", serveAsset)
	app.Get("/", serveIndex)
	app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/v1/api/") {
			return fiber.ErrNotFound
		}
		return serveIndex(c)
	})
	if cfg.SelfManageIngress {
		go func() {
			reconcileCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := k8sManager.ApplyControllerAccess(reconcileCtx, k8s.ControllerAccessOptions{
				Namespace:     cfg.ControllerNamespace,
				Name:          cfg.ControllerName,
				ServicePort:   cfg.SelfServicePort,
				PublicHost:    cfg.SelfPublicHost,
				TailscaleHost: cfg.SelfTailscaleHost,
				WebhookHost:   cfg.SelfWebhookHost,
			}); err != nil {
				log.Warn("self access reconcile failed", zap.Error(err))
				return
			}
			log.Info("self access reconciled")
		}()
	}
	go func() {
		reconcileCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, err := k8sManager.EnsureTraefikPodNetwork(reconcileCtx)
		if err != nil {
			log.Warn("traefik pod network reconcile failed", zap.Error(err))
			return
		}
		log.Info("traefik pod network reconciled", zap.String("namespace", result.Namespace), zap.String("name", result.Name), zap.Bool("updated", result.Updated))
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go buildSvc.StartReconciler(ctx)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(shutdownCtx)
	}()
	log.Info("starting beancs controller", zap.String("port", cfg.Port), zap.String("version", cfg.Version))
	if err := app.Listen(":" + cfg.Port); err != nil && !strings.Contains(strings.ToLower(err.Error()), "server closed") {
		log.Fatal("listen", zap.Error(err))
	}
}

func registerAPI(api fiber.Router, cfg *config.Config, db *gorm.DB, registry *basaltpass.ClientRegistry, credentialSvc *service.CredentialService, apiKeySvc *service.APIKeyService, registryImageSvc *service.ContainerRegistryService, projectSvc *service.ProjectService, deploymentSvc *service.DeploymentService, processSvc *service.ProcessService, k8sManager *k8s.Manager, v *validator.Validate) {
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": cfg.Version})
	})
	api.Get("/ready", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready", "version": cfg.Version})
	})
	api.Get("/db-ready", func(c *fiber.Ctx) error {
		sqlDB, err := db.DB()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unavailable", "error": "database handle unavailable"})
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unavailable", "error": "database unavailable"})
		}
		return c.JSON(fiber.Map{"status": "ready", "database": "ok", "version": cfg.Version})
	})
	api.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": cfg.Version})
	})
	api.Get("/ui/config", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"auth_url":  cfg.BPBrowserAuthURL,
			"client_id": cfg.BPBrowserClientID,
		})
	})
	api.Get("/ui/oauth/callback", serveIndex)
	api.Post("/ui/oauth/token", func(c *fiber.Ctx) error {
		return exchangeBrowserToken(c, cfg)
	})

	webhookLimiter := limiter.New(limiter.Config{Max: cfg.WebhookRateLimitPerMinute, Expiration: time.Minute})
	handler.NewWebhookHandler(deploymentSvc, v).Register(api.Group("/webhooks", webhookLimiter, middleware.WebhookVerify(cfg.WebhookSecret)))

	authLimiter := limiter.New(limiter.Config{
		Max:        cfg.APIRateLimitPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			auth := c.Get("Authorization")
			if auth == "" {
				return c.IP()
			}
			return strings.TrimSpace(auth)
		},
	})
	credentialHandler := handler.NewCredentialHandler(db, credentialSvc, registry, cfg, v)
	credentialHandler.RegisterGitHubAppCallback(api.Group("/credentials/github"))
	secured := api.Group("/", authLimiter, middleware.Auth(registry, apiKeySvc), middleware.Audit(db))
	credentialHandler.Register(secured)
	handler.NewAPIKeyHandler(apiKeySvc, v).Register(secured)
	handler.NewContainerRegistryHandler(registryImageSvc, v).Register(secured)
	handler.NewProjectHandler(db, projectSvc, k8sManager, v).Register(secured)
	handler.NewDeploymentHandler(db, deploymentSvc, v).Register(secured)
	handler.NewProcessHandler(db, processSvc).Register(secured)
	handler.NewRuntimeHandler(db, k8sManager, v).Register(secured)
	secured.Get("/me", func(c *fiber.Ctx) error {
		return browserUserInfo(c, cfg)
	})
	secured.Use("/admin", middleware.RequireScope("beancs.admin"))
	handler.NewAdminHandler(db, k8sManager, v).Register(secured)
}

func openDatabase(log *zap.Logger, databaseURL string) (*gorm.DB, error) {
	dsn := databaseURLWithConnectTimeout(databaseURL, "15")
	deadline := time.Now().Add(5 * time.Minute)
	var lastErr error
	for {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				configureDatabasePool(sqlDB)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				sqlErr = sqlDB.PingContext(ctx)
				cancel()
			}
			if sqlErr == nil {
				return db, nil
			}
			lastErr = sqlErr
			if sqlDB != nil {
				_ = sqlDB.Close()
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		log.Warn("database unavailable; retrying", zap.Error(lastErr))
		time.Sleep(3 * time.Second)
	}
}

type databasePool interface {
	SetMaxOpenConns(int)
	SetMaxIdleConns(int)
	SetConnMaxLifetime(time.Duration)
	SetConnMaxIdleTime(time.Duration)
}

func configureDatabasePool(db databasePool) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}

func databaseURLWithConnectTimeout(databaseURL, timeout string) string {
	u, err := url.Parse(databaseURL)
	if err != nil || u.Scheme == "" {
		return databaseURL
	}
	q := u.Query()
	if q.Get("connect_timeout") == "" {
		q.Set("connect_timeout", timeout)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func serveIndex(c *fiber.Ctx) error {
	c.Type("html", "utf-8")
	c.Set(fiber.HeaderCacheControl, "no-store")
	body, err := web.IndexHTML()
	if err != nil {
		return err
	}
	return c.Send(body)
}

func serveAsset(c *fiber.Ctx) error {
	path := strings.TrimPrefix(c.Path(), "/")
	body, err := web.Asset(path)
	if err != nil {
		return fiber.ErrNotFound
	}
	if typ := mime.TypeByExtension(urlPathExt(path)); typ != "" {
		c.Set(fiber.HeaderContentType, typ)
	}
	return c.Send(body)
}

func urlPathExt(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx:]
	}
	return ""
}

func exchangeBrowserToken(c *fiber.Ctx, cfg *config.Config) error {
	var req struct {
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Code == "" || req.RedirectURI == "" || req.CodeVerifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code, redirect_uri, and code_verifier are required"})
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.BPBrowserClientID)
	form.Set("code", req.Code)
	form.Set("redirect_uri", req.RedirectURI)
	form.Set("code_verifier", req.CodeVerifier)
	if cfg.BPBrowserSecret != "" {
		form.Set("client_secret", cfg.BPBrowserSecret)
	}
	httpReq, err := http.NewRequestWithContext(c.UserContext(), http.MethodPost, strings.TrimRight(cfg.BPBrowserAuthURL, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid auth configuration"})
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "auth token exchange failed"})
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "auth token response failed"})
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Set("Content-Type", contentType)
	return c.Status(resp.StatusCode).Send(body)
}

func browserUserInfo(c *fiber.Ctx, cfg *config.Config) error {
	if middleware.AuthMethod(c) == "api_key" {
		return c.JSON(fiber.Map{
			"sub":         middleware.UserID(c),
			"tenant_id":   middleware.TenantID(c),
			"tenant_code": middleware.TenantCode(c),
			"scope":       strings.Join(middleware.Scopes(c), " "),
			"auth_method": "api_key",
		})
	}
	token := bearerToken(c.Get("Authorization"))
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
	}
	httpReq, err := http.NewRequestWithContext(c.UserContext(), http.MethodGet, strings.TrimRight(cfg.BPBrowserAuthURL, "/")+"/oauth/userinfo", nil)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid auth configuration"})
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "auth userinfo request failed"})
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "auth userinfo response failed"})
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.Status(resp.StatusCode).Send(body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "auth userinfo returned invalid JSON"})
	}
	out["tenant_id"] = middleware.TenantID(c)
	out["tenant_code"] = middleware.TenantCode(c)
	out["auth_method"] = "basaltpass"
	return c.JSON(out)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
