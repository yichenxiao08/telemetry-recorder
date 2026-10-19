package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/signalscout/signalscout/samples/shop/workload"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type poolHealth struct {
	Ready    bool  `json:"ready"`
	InUse    int32 `json:"in_use"`
	Capacity int32 `json:"capacity"`
}
type app struct {
	config    config
	telemetry *telemetry
	inventory *inventory
	health    func() poolHealth
	client    *http.Client
	logger    otellog.Logger
}

func (a *app) handler() http.Handler {
	options := []otelhttp.Option{otelhttp.WithTracerProvider(a.telemetry.Traces), otelhttp.WithMeterProvider(a.telemetry.Metrics), otelhttp.WithPropagators(propagation.TraceContext{})}
	transport := otelhttp.NewTransport(http.DefaultTransport.(*http.Transport).Clone(), options...)
	a.client = &http.Client{Transport: transport, Timeout: 5 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.health())
	})
	path, name := "/checkout", "frontend.request"
	if a.config.Role == "checkout-service" {
		name = "checkout-service.handle"
	}
	if a.config.Role == "inventory-service" {
		path = "/reserve"
		name = "inventory-service.reserve"
	}
	options = append(options, otelhttp.WithSpanNameFormatter(func(string, *http.Request) string { return name }))
	mux.Handle("POST "+path, otelhttp.NewHandler(http.HandlerFunc(a.checkout), name, options...))
	return mux
}

func (a *app) checkout(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(attribute.String("http.route", r.URL.Path), attribute.String("http.request.method", r.Method))
	w.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
	var cart workload.Cart
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cart); err != nil {
		http.Error(w, "invalid cart JSON", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "trailing JSON", http.StatusBadRequest)
		return
	}
	if err := cart.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	span.SetAttributes(attribute.String("shop.request.id", cart.ID), attribute.Int("shop.cart.lines", len(cart.Items)))
	if a.inventory != nil {
		if err := a.inventory.reserve(r.Context(), cart); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "reservation failed")
			emitLog(r.Context(), a.logger, "inventory reservation failed: "+err.Error(), otellog.SeverityError)
			http.Error(w, "inventory unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"reserved":true}`)
	} else if err := a.forward(r.Context(), w, cart); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "downstream failed")
		emitLog(r.Context(), a.logger, err.Error(), otellog.SeverityError)
		http.Error(w, "downstream unavailable", http.StatusBadGateway)
		return
	}
	emitLog(r.Context(), a.logger, "checkout request completed", otellog.SeverityInfo)
}

func (a *app) forward(ctx context.Context, w http.ResponseWriter, cart workload.Cart) error {
	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("encode cart: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.Downstream, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("create downstream request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("call downstream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	if _, err = io.Copy(w, response.Body); err != nil {
		return fmt.Errorf("copy downstream response: %w", err)
	}
	return nil
}
