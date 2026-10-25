package intake

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/signalscout/signalscout/internal/httpdrain"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/redaction"
	"github.com/signalscout/signalscout/internal/runs"
	"github.com/signalscout/signalscout/internal/store"
	logscol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracescol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const MaxBodyBytes = 16 << 20

type Submitter interface {
	Submit(context.Context, model.Batch) error
}
type Handler struct {
	queue      Submitter
	normalizer *runs.Normalizer
	policy     *redaction.Policy
	profile    string
}

func New(queue Submitter) http.Handler {
	return NewWithPolicy(queue, nil, "")
}

func NewWithPolicy(queue Submitter, policy *redaction.Policy, profile string) http.Handler {
	h := &Handler{queue: queue, normalizer: runs.NewNormalizer(), policy: policy, profile: profile}
	mux := http.NewServeMux()
	for _, kind := range []string{"traces", "logs", "metrics"} {
		mux.Handle("POST /v1/"+kind, h)
	}
	return mux
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request, response proto.Message
	switch r.URL.Path {
	case "/v1/traces":
		request = &tracescol.ExportTraceServiceRequest{}
		response = &tracescol.ExportTraceServiceResponse{}
	case "/v1/logs":
		request = &logscol.ExportLogsServiceRequest{}
		response = &logscol.ExportLogsServiceResponse{}
	case "/v1/metrics":
		request = &metricscol.ExportMetricsServiceRequest{}
		response = &metricscol.ExportMetricsServiceResponse{}
	default:
		httpdrain.Error(w, r, "404 page not found", http.StatusNotFound)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil && r.Header.Get("Content-Type") != "" {
		httpdrain.Error(w, r, "invalid content type", http.StatusUnsupportedMediaType)
		return
	}
	isJSON := contentType == "application/json" || strings.HasSuffix(contentType, "+json")
	if !isJSON && contentType != "" && contentType != "application/x-protobuf" && contentType != "application/protobuf" {
		httpdrain.Error(w, r, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	encoding := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
	if encoding != "" && encoding != "identity" && encoding != "gzip" {
		httpdrain.Error(w, r, "unsupported content encoding", http.StatusUnsupportedMediaType)
		return
	}
	body := io.Reader(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if encoding == "gzip" {
		gz, err := gzip.NewReader(body)
		if err != nil {
			httpdrain.Error(w, r, "invalid gzip body", http.StatusBadRequest)
			return
		}
		defer func() { _ = gz.Close() }()
		body = io.LimitReader(gz, MaxBodyBytes+1)
	}
	data, err := io.ReadAll(body)
	if len(data) > MaxBodyBytes {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid body", http.StatusBadRequest)
		}
		return
	}
	if isJSON {
		data, err = otlpJSON(data)
		if err == nil {
			err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, request)
		}
	} else {
		err = proto.Unmarshal(data, request)
	}
	if err != nil {
		http.Error(w, "malformed OTLP payload", http.StatusBadRequest)
		return
	}
	if h.policy != nil {
		h.policy.OTLP(request, h.profile)
	}
	batch, err := decode(request, h.normalizer)
	if err != nil {
		http.Error(w, "invalid OTLP records: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err = h.queue.Submit(r.Context(), batch); err != nil {
		w.Header().Set("Retry-After", "1")
		if errors.Is(err, store.QueueFull) {
			http.Error(w, "ingest queue full", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	if isJSON {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	} else {
		data, err = proto.Marshal(response)
		if err != nil {
			http.Error(w, "encode response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, _ = w.Write(data)
	}
}

// OTLP JSON uses hex IDs where generic protojson expects base64 byte fields.
// Translate only identifier fields, including links/exemplars, before protojson.
func otlpJSON(data []byte) ([]byte, error) {
	var value any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON data")
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for key, item := range x {
				if key == "traceId" || key == "trace_id" || key == "spanId" || key == "span_id" || key == "parentSpanId" || key == "parent_span_id" {
					if s, ok := item.(string); ok {
						if decoded, err := hex.DecodeString(s); err == nil {
							x[key] = base64.StdEncoding.EncodeToString(decoded)
						}
					}
				} else {
					walk(item)
				}
			}
		case []any:
			for _, item := range x {
				walk(item)
			}
		}
	}
	walk(value)
	result, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode OTLP JSON: %w", err)
	}
	return result, nil
}
