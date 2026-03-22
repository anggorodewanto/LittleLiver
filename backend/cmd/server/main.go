package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/backup"
	"github.com/ablankz/LittleLiver/backend/internal/cron"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/notify"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "littleliver.db"
	}

	db, err := store.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migDir := os.Getenv("MIGRATIONS_DIR")
	if migDir == "" {
		migDir = "migrations"
	}
	if err := store.RunMigrations(db, migDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	// Build handler options
	opts := []handler.Option{handler.WithDB(db)}

	// Initialize R2 object store if configured
	var objStore storage.ObjectStore
	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKey := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2Bucket := os.Getenv("R2_BUCKET_NAME")
	if r2AccountID != "" && r2AccessKey != "" && r2SecretKey != "" && r2Bucket != "" {
		r2Client, err := storage.NewR2Client(context.Background(), storage.R2Config{
			AccountID:       r2AccountID,
			AccessKeyID:     r2AccessKey,
			SecretAccessKey: r2SecretKey,
			BucketName:      r2Bucket,
		})
		if err != nil {
			log.Fatalf("init r2 client: %v", err)
		}
		objStore = r2Client
		opts = append(opts, handler.WithObjectStore(objStore))
		log.Printf("R2 object store configured (bucket: %s)", r2Bucket)
	} else {
		log.Printf("WARNING: R2 not configured — photo uploads disabled")
	}

	// Validate SESSION_SECRET is set when OAuth is configured
	sessionSecret := os.Getenv("SESSION_SECRET")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if googleClientID != "" && sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required when Google OAuth is configured")
	}

	// Initialize push notifications if VAPID keys are configured
	vapidPublic := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivate := os.Getenv("VAPID_PRIVATE_KEY")
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	if vapidPublic != "" {
		opts = append(opts, handler.WithVAPIDPublicKey(vapidPublic))
	}

	if vapidPublic != "" && vapidPrivate != "" {
		vapidSubscriber := os.Getenv("VAPID_SUBSCRIBER")
		if vapidSubscriber == "" {
			vapidSubscriber = baseURL
		}
		pusher := notify.NewWebPusher(notify.VAPIDConfig{
			PublicKey:  vapidPublic,
			PrivateKey: vapidPrivate,
			Subscriber: vapidSubscriber,
		})

		// Start medication reminder scheduler
		scheduler := notify.NewScheduler(db, pusher)
		go scheduler.Start()
		defer scheduler.Stop()
		log.Printf("medication reminder scheduler started")
	} else {
		log.Printf("WARNING: VAPID keys not configured — push notifications disabled")
	}

	// Start cron jobs (invite/session/photo cleanup)
	cronRunner := cron.NewRunner(db, objStore, cron.DefaultInterval)
	cronRunner.Start()
	defer cronRunner.Stop()
	log.Printf("cron jobs started (interval: %s)", cron.DefaultInterval)

	// Start daily backup job (if R2 is configured)
	if objStore != nil {
		backupRunner := backup.NewRunner(db, objStore, backup.DefaultInterval)
		backupRunner.Start()
		defer backupRunner.Stop()
		log.Printf("backup job started (interval: %s)", backup.DefaultInterval)
	} else {
		log.Printf("WARNING: backup job not started — R2 not configured")
	}

	mux := handler.NewMux(opts...)

	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	// Graceful shutdown on SIGTERM/SIGINT so deferred cleanup runs.
	// 5-second timeout ensures deferred db.Close/cron.Stop run before Fly.io SIGKILL.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		log.Printf("received %s, shutting down gracefully...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown error: %v", err)
		}
	}()

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Printf("server stopped")
}
