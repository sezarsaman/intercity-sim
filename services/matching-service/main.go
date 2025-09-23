package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chi "github.com/go-chi/chi/v5"

	"github.com/redis/go-redis/v9"

	"github.com/sezarsaman/intercity-sim/pkg/mq"
	"github.com/sezarsaman/intercity-sim/services/matching-service/internal/eventsub"
	msvc "github.com/sezarsaman/intercity-sim/services/matching-service/internal/service"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisAddr := getenv("REDIS_ADDR", "redis:6379")

	rabbitURL := getenv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")
	rb, err := dialRabbitWithRetry(rabbitURL, 20, 1500*time.Millisecond) // ~30s total
	if err != nil {
		log.Fatal(err)
	}
	defer rb.Close()

	// ensure topology is declared by one service at boot (idempotent)
	if err := mq.BootstrapTopology(rabbitURL); err != nil {
		log.Fatal(err)
	}

	pub, err := rb.Publisher(mq.ExchangeEvents)
	if err != nil {
		log.Fatal(err)
	}

	sub, err := rb.Subscriber(mq.ExchangeEvents, mq.QueueMatchingTripPriced, 20)
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	svc := msvc.New(rdb, pub)

	go func() {
		if err := eventsub.ConsumeTripPriced(ctx, sub, svc.HandleTripPriced); err != nil {
			log.Printf("matching-service: consumer stopped: %v", err)
		}
	}()

	log.Printf("[matching-service] up (redis=%s)", redisAddr)

	// optional health check
	port := getenv("PORT", "8081")
	log.Printf("[matching-service] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, NewRouter()))
}

func dialRabbitWithRetry(url string, attempts int, sleep time.Duration) (*mq.Rabbit, error) {
	var last error
	for i := 1; i <= attempts; i++ {
		rb, err := mq.Dial(url)
		if err == nil {
			return rb, nil
		}
		last = err
		log.Printf("[matching-service] rabbit dial failed (try %d/%d): %v", i, attempts, err)
		time.Sleep(sleep)
	}
	return nil, last
}

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Matching-service: write health response error: %v", err)
		}
	})

	return r
}
