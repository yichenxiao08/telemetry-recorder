package main

import (
	"fmt"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
)

type config struct {
	Role, Address, Endpoint, Downstream, DatabaseURL, Host, Revision, Dirty, Phase string
	PoolSize                                                                       int
}

func environment(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func readConfig() (config, error) {
	size, err := strconv.Atoi(environment("DB_POOL_SIZE", "20"))
	if err != nil || size < 1 || size > 1000 {
		return config{}, fmt.Errorf("DB_POOL_SIZE must be 1..1000")
	}
	c := config{Role: environment("SERVICE_NAME", "frontend"), Address: environment("LISTEN_ADDR", ":8080"), Endpoint: environment("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318"), Downstream: os.Getenv("DOWNSTREAM_URL"), DatabaseURL: os.Getenv("DATABASE_URL"), Host: environment("SHOP_HOST_NAME", "docker-desktop"), Revision: environment("GIT_COMMIT", "unknown"), Dirty: environment("GIT_DIRTY", "unknown"), Phase: environment("SHOP_PHASE", "baseline"), PoolSize: size}
	if c.Role != "frontend" && c.Role != "checkout-service" && c.Role != "inventory-service" {
		return c, fmt.Errorf("unknown SERVICE_NAME %q", c.Role)
	}
	if c.Role == "inventory-service" && c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required for inventory")
	}
	if c.Role != "inventory-service" && c.Downstream == "" {
		return c, fmt.Errorf("DOWNSTREAM_URL is required")
	}
	return c, nil
}

func (c config) resourceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("service.name", c.Role), attribute.String("service.instance.id", c.Role+"-1"), attribute.String("service.version", "m2"), attribute.String("host.name", c.Host),
		attribute.String("vcs.revision", c.Revision), attribute.String("vcs.dirty", c.Dirty), attribute.String("deployment.environment.name", "shop-demo"), attribute.String("cloud.region", "local"), attribute.String("signalscout.demo.phase", c.Phase),
		attribute.Int("signalscout.config.DB_POOL_SIZE", c.PoolSize), attribute.Int("signalscout.config.ACQUIRE_TIMEOUT_MS", 100), attribute.Int("signalscout.config.DB_RETRIES", 1),
	}
}
