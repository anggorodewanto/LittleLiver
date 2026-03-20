package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// Initialize push notifications if VAPID keys are configured
	vapidPublic := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivate := os.Getenv("VAPID_PRIVATE_KEY")
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	if vapidPublic != "" && vapidPrivate != "" {
		pusher := notify.NewWebPusher(notify.VAPIDConfig{
			PublicKey:  vapidPublic,
			PrivateKey: vapidPrivate,
			Subscriber: baseURL,
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

	mux := handler.NewMux(opts...)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
