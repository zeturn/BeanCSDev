package main

import (
	"context"
	"errors"
	"io"
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
	credentialSvc := service.NewCredentialService(db, cipher)
	quotaSvc := service.NewQuotaService(db)
	dnsSvc := service.NewDNSService(cfg.IngressIP)
	gitopsSvc := service.NewGitOpsService()
	projectSvc := service.NewProjectService(db, credentialSvc, quotaSvc, dnsSvc, gitopsSvc, k8sManager, registry, cipher)
	deploymentSvc := service.NewDeploymentService(db)

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

	api := app.Group("/api/v1")
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": cfg.Version})
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

	webhookLimiter := limiter.New(limiter.Config{Max: 30, Expiration: time.Minute})
	handler.NewWebhookHandler(deploymentSvc, v).Register(api.Group("/webhooks", webhookLimiter, middleware.WebhookVerify(cfg.WebhookSecret)))

	authLimiter := limiter.New(limiter.Config{
		Max:        60,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			auth := c.Get("Authorization")
			if auth == "" {
				return c.IP()
			}
			return strings.TrimSpace(auth)
		},
	})
	secured := api.Group("/", authLimiter, middleware.Auth(registry), middleware.Audit(db))
	handler.NewCredentialHandler(db, credentialSvc, registry, v).Register(secured)
	handler.NewProjectHandler(db, projectSvc, k8sManager, v).Register(secured)
	handler.NewDeploymentHandler(db, deploymentSvc, v).Register(secured)
	handler.NewRuntimeHandler(db, k8sManager, v).Register(secured)
	secured.Use("/admin", middleware.RequireScope("beancs.admin"))
	handler.NewAdminHandler(db, k8sManager, v).Register(secured)

	app.Get("/", serveIndex)
	app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/") {
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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

func openDatabase(log *zap.Logger, databaseURL string) (*gorm.DB, error) {
	dsn := databaseURLWithConnectTimeout(databaseURL, "5")
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	return c.SendString(web.IndexHTML())
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
