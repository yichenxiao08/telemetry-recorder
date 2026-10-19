package intake

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/signalscout/signalscout/internal/model"
	"github.com/signalscout/signalscout/internal/runs"
	logscol "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscol "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracescol "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type decoder struct {
	batch       model.Batch
	err         error
	occurrences map[string]int
}

func canonical(message proto.Message) (json.RawMessage, error) {
	data, err := protojson.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("encode OTLP record: %w", err)
	}
	// protojson may insert insignificant whitespace. Canonical JSON keeps IDs stable.
	var value any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err = dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("canonicalize OTLP record: %w", err)
	}
	data, err = json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical record: %w", err)
	}
	return data, nil
}

func attributes(attrs []*common.KeyValue) map[string]any {
	result := map[string]any{}
	for _, attr := range attrs {
		result[attr.GetKey()] = anyValue(attr.GetValue())
	}
	return result
}

func anyValue(value *common.AnyValue) any {
	switch v := value.GetValue().(type) {
	case *common.AnyValue_StringValue:
		return v.StringValue
	case *common.AnyValue_BoolValue:
		return v.BoolValue
	case *common.AnyValue_IntValue:
		return v.IntValue
	case *common.AnyValue_DoubleValue:
		if math.IsNaN(v.DoubleValue) || math.IsInf(v.DoubleValue, 0) {
			return strconv.FormatFloat(v.DoubleValue, 'g', -1, 64)
		}
		return v.DoubleValue
	case *common.AnyValue_BytesValue:
		return hex.EncodeToString(v.BytesValue)
	case *common.AnyValue_ArrayValue:
		values := []any{}
		for _, item := range v.ArrayValue.GetValues() {
			values = append(values, anyValue(item))
		}
		return values
	case *common.AnyValue_KvlistValue:
		return attributes(v.KvlistValue.GetValues())
	default:
		return nil
	}
}

func (d *decoder) makeResource(r *resource.Resource, schema string) model.Resource {
	raw, err := canonical(r)
	if err != nil {
		d.err = err
		return model.Resource{}
	}
	wrapped, err := json.Marshal(struct {
		Schema   string          `json:"schema_url"`
		Resource json.RawMessage `json:"resource"`
	}{schema, raw})
	if err != nil {
		d.err = err
		return model.Resource{}
	}
	identity := wrapped
	if r != nil {
		// Attribute order has no resource meaning. Keep original order in Raw,
		// but hash a sorted copy so SDK ordering cannot break context matching.
		ordered := proto.Clone(r).(*resource.Resource)
		sort.SliceStable(ordered.Attributes, func(i, j int) bool { return ordered.Attributes[i].GetKey() < ordered.Attributes[j].GetKey() })
		orderedRaw, err := canonical(ordered)
		if err != nil {
			d.err = err
			return model.Resource{}
		}
		identity, err = json.Marshal(struct {
			Schema   string          `json:"schema_url"`
			Resource json.RawMessage `json:"resource"`
		}{schema, orderedRaw})
		if err != nil {
			d.err = err
			return model.Resource{}
		}
	}
	hash := sha256.Sum256(identity)
	return model.Resource{ID: hex.EncodeToString(hash[:]), Attrs: attributes(r.GetAttributes()), Raw: wrapped}
}

func (d *decoder) scope(s *common.InstrumentationScope, schema string) json.RawMessage {
	raw, err := canonical(s)
	if err != nil {
		d.err = err
		return nil
	}
	wrapped, err := json.Marshal(struct {
		Schema string          `json:"schema_url"`
		Scope  json.RawMessage `json:"scope"`
	}{schema, raw})
	if err != nil {
		d.err = err
	}
	return wrapped
}

func (d *decoder) id(value []byte, length int, required bool) string {
	if len(value) != 0 && len(value) != length {
		d.err = fmt.Errorf("invalid %d-byte identifier", length)
		return ""
	}
	zero := true
	for _, b := range value {
		if b != 0 {
			zero = false
			break
		}
	}
	if len(value) == 0 || zero {
		if required {
			d.err = fmt.Errorf("missing required %d-byte identifier", length)
		}
		return ""
	}
	return hex.EncodeToString(value)
}

func (d *decoder) timestamp(value uint64) int64 {
	if value > math.MaxInt64 {
		d.err = fmt.Errorf("timestamp exceeds signed nanoseconds")
		return 0
	}
	return int64(value)
}

func (d *decoder) add(signal model.Signal, message proto.Message) {
	if d.err != nil {
		return
	}
	raw, err := canonical(message)
	if err != nil {
		d.err = err
		return
	}
	if len(signal.Raw) == 0 {
		signal.Raw = raw
	}
	if signal.End == 0 {
		signal.End = signal.TS
	}
	if signal.ID == "" {
		identity, err := json.Marshal(struct {
			Kind, Resource string
			Scope, Raw     json.RawMessage
			Parent, Trace  string
		}{signal.Kind, signal.Resource.ID, signal.Scope, signal.Raw, signal.SpanID, signal.TraceID})
		if err != nil {
			d.err = err
			return
		}
		hash := sha256.Sum256(identity)
		base := signal.Kind + ":" + hex.EncodeToString(hash[:])
		ordinal := d.occurrences[base]
		d.occurrences[base]++
		signal.ID = base + "/" + strconv.Itoa(ordinal)
	}
	signal.RunID, _ = runs.Identity(signal)
	d.batch.Signals = append(d.batch.Signals, signal)
}

func decode(message proto.Message, n *runs.Normalizer) (model.Batch, error) {
	d := &decoder{occurrences: map[string]int{}}
	switch request := message.(type) {
	case *tracescol.ExportTraceServiceRequest:
		for _, rs := range request.GetResourceSpans() {
			r := d.makeResource(rs.GetResource(), rs.GetSchemaUrl())
			for _, ss := range rs.GetScopeSpans() {
				scope := d.scope(ss.GetScope(), ss.GetSchemaUrl())
				for _, span := range ss.GetSpans() {
					s := model.Signal{Kind: "span", Resource: r, Scope: scope, TraceID: d.id(span.GetTraceId(), 16, true), SpanID: d.id(span.GetSpanId(), 8, true), ParentID: d.id(span.GetParentSpanId(), 8, false), Name: span.GetName(), TS: d.timestamp(span.GetStartTimeUnixNano()), End: d.timestamp(span.GetEndTimeUnixNano()), Attrs: attributes(span.GetAttributes())}
					if s.End < s.TS {
						d.err = fmt.Errorf("span ends before it starts")
					}
					s.ID = "span:" + s.TraceID + "/" + s.SpanID
					if span.GetStatus().GetCode() == 2 {
						s.Status = "error"
					} else {
						s.Status = "ok"
					}
					d.add(s, span)
					for _, event := range span.GetEvents() {
						e := model.Signal{Kind: "event", TraceID: s.TraceID, SpanID: s.SpanID, Name: event.GetName(), TS: d.timestamp(event.GetTimeUnixNano()), Resource: r, Scope: scope, Attrs: attributes(event.GetAttributes())}
						d.add(e, event)
					}
				}
			}
		}
	case *logscol.ExportLogsServiceRequest:
		for _, rl := range request.GetResourceLogs() {
			r := d.makeResource(rl.GetResource(), rl.GetSchemaUrl())
			for _, sl := range rl.GetScopeLogs() {
				scope := d.scope(sl.GetScope(), sl.GetSchemaUrl())
				for _, record := range sl.GetLogRecords() {
					ts := record.GetTimeUnixNano()
					if ts == 0 {
						ts = record.GetObservedTimeUnixNano()
					}
					body := anyValue(record.GetBody())
					bodyText, ok := body.(string)
					if !ok {
						data, err := json.Marshal(body)
						if err != nil {
							d.err = err
						}
						bodyText = string(data)
					}
					s := model.Signal{Kind: "log", Resource: r, Scope: scope, TraceID: d.id(record.GetTraceId(), 16, false), SpanID: d.id(record.GetSpanId(), 8, false), Name: record.GetSeverityText(), TS: d.timestamp(ts), Attrs: attributes(record.GetAttributes()), Body: bodyText, Template: n.Template(bodyText)}
					d.add(s, record)
				}
			}
		}
	case *metricscol.ExportMetricsServiceRequest:
		for _, rm := range request.GetResourceMetrics() {
			r := d.makeResource(rm.GetResource(), rm.GetSchemaUrl())
			for _, sm := range rm.GetScopeMetrics() {
				scope := d.scope(sm.GetScope(), sm.GetSchemaUrl())
				for _, metric := range sm.GetMetrics() {
					d.metric(metric, r, scope)
				}
			}
		}
	default:
		return model.Batch{}, fmt.Errorf("unsupported OTLP message")
	}
	return d.batch, d.err
}

func (d *decoder) metric(metric *metrics.Metric, r model.Resource, scope json.RawMessage) {
	metadata := map[string]any{"name": metric.GetName(), "description": metric.GetDescription(), "unit": metric.GetUnit()}
	extra, err := canonical(&metrics.Metric{Metadata: metric.GetMetadata()})
	if err != nil {
		d.err = err
		return
	}
	metadata["extra"] = extra
	add := func(point proto.Message, ts uint64, attrs []*common.KeyValue, value *float64) {
		raw, err := canonical(point)
		if err != nil {
			d.err = err
			return
		}
		wrapped, err := json.Marshal(struct {
			Metric map[string]any  `json:"metric"`
			Point  json.RawMessage `json:"point"`
		}{metadata, raw})
		if err != nil {
			d.err = err
			return
		}
		s := model.Signal{Kind: "metric", Name: metric.GetName(), TS: d.timestamp(ts), Attrs: attributes(attrs), Resource: r, Scope: scope, Raw: wrapped, Value: value}
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			s.Value = nil
			if math.IsNaN(*value) {
				s.ValueText = "NaN"
			} else if math.IsInf(*value, 1) {
				s.ValueText = "Infinity"
			} else {
				s.ValueText = "-Infinity"
			}
		}
		d.add(s, point)
	}
	number := func(point *metrics.NumberDataPoint) {
		var value *float64
		switch v := point.GetValue().(type) {
		case *metrics.NumberDataPoint_AsDouble:
			x := v.AsDouble
			value = &x
		case *metrics.NumberDataPoint_AsInt:
			x := float64(v.AsInt)
			value = &x
		}
		add(point, point.GetTimeUnixNano(), point.GetAttributes(), value)
	}
	switch data := metric.GetData().(type) {
	case *metrics.Metric_Gauge:
		metadata["type"] = "gauge"
		for _, p := range data.Gauge.GetDataPoints() {
			number(p)
		}
	case *metrics.Metric_Sum:
		metadata["type"] = "sum"
		metadata["aggregation_temporality"] = int32(data.Sum.GetAggregationTemporality())
		metadata["is_monotonic"] = data.Sum.GetIsMonotonic()
		for _, p := range data.Sum.GetDataPoints() {
			number(p)
		}
	case *metrics.Metric_Histogram:
		metadata["type"] = "histogram"
		metadata["aggregation_temporality"] = int32(data.Histogram.GetAggregationTemporality())
		for _, p := range data.Histogram.GetDataPoints() {
			add(p, p.GetTimeUnixNano(), p.GetAttributes(), nil)
		}
	case *metrics.Metric_ExponentialHistogram:
		metadata["type"] = "exponential_histogram"
		metadata["aggregation_temporality"] = int32(data.ExponentialHistogram.GetAggregationTemporality())
		for _, p := range data.ExponentialHistogram.GetDataPoints() {
			add(p, p.GetTimeUnixNano(), p.GetAttributes(), nil)
		}
	case *metrics.Metric_Summary:
		metadata["type"] = "summary"
		for _, p := range data.Summary.GetDataPoints() {
			add(p, p.GetTimeUnixNano(), p.GetAttributes(), nil)
		}
	}
}
