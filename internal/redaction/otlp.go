package redaction

import (
	"encoding/json"

	common "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// OTLP scrubs the decoded protobuf BEFORE canonical/raw copies, hashes, templates
// or resource identities are created. Unknown protobuf wire fields are removed.
func (p *Policy) OTLP(message proto.Message, profile string) {
	var walk func(protoreflect.Message, string)
	walk = func(m protoreflect.Message, scope string) {
		m.SetUnknown(nil)
		if m.Descriptor().Name() == "AnyValue" {
			m.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool { m.Clear(fd); return true })
			return
		}
		switch m.Descriptor().Name() {
		case "Resource":
			scope = "resource."
		case "InstrumentationScope":
			scope = "scope."
		}
		m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if fd.IsList() && fd.Message() != nil && fd.Message().Name() == "KeyValue" {
				list := v.List()
				var kept []protoreflect.Value
				for i := 0; i < list.Len(); i++ {
					item := list.Get(i)
					kv := item.Message().Interface().(*common.KeyValue)
					kv.ProtoReflect().SetUnknown(nil)
					key := kv.GetKey()
					if key == "signalscout.record.json" {
						kv.Value = &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: p.Envelope(profile, kv.GetValue().GetStringValue())}}
						kept = append(kept, item)
						continue
					}
					var x any
					b, err := protojson.Marshal(kv.GetValue())
					if err != nil {
						continue
					}
					var val map[string]any
					if json.Unmarshal(b, &val) != nil {
						continue
					}
					for _, name := range []string{"stringValue", "boolValue", "doubleValue"} {
						if a, ok := val[name]; ok {
							x = a
						}
					}
					if text, ok := val["intValue"].(string); ok {
						n := json.Number(text)
						if f, err := n.Float64(); err == nil {
							x = f
						}
					}
					clean, ok := p.Value(profile, scope+"attributes."+key, x)
					if !ok {
						continue
					}
					var value any
					switch x := clean.(type) {
					case string:
						value = map[string]any{"stringValue": x}
					case bool:
						value = map[string]any{"boolValue": x}
					case float64:
						value = map[string]any{"doubleValue": x}
					default:
						continue
					}
					data, _ := json.Marshal(value)
					kv.Value = &common.AnyValue{}
					if protojson.Unmarshal(data, kv.Value) == nil {
						kept = append(kept, item)
					}
				}
				list.Truncate(0)
				for _, item := range kept {
					list.Append(item)
				}
				return true
			}
			if fd.IsList() {
				if fd.Message() != nil {
					list := v.List()
					for i := 0; i < list.Len(); i++ {
						walk(list.Get(i).Message(), scope)
					}
				} else {
					m.Clear(fd) // Repeated scalar payloads are capture-OFF too.
				}
				return true
			}
			switch fd.Kind() {
			case protoreflect.MessageKind:
				walk(v.Message(), scope)
			case protoreflect.StringKind:
				clean, ok := p.Value(profile, scope+"otlp."+string(fd.Name()), v.String())
				if text, yes := clean.(string); ok && yes {
					m.Set(fd, protoreflect.ValueOfString(text))
				} else {
					m.Clear(fd)
				}
			case protoreflect.BytesKind:
				if fd.Name() != "trace_id" && fd.Name() != "span_id" && fd.Name() != "parent_span_id" {
					m.Clear(fd)
				}
			case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Uint32Kind, protoreflect.Uint64Kind, protoreflect.Fixed32Kind, protoreflect.Fixed64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind, protoreflect.DoubleKind, protoreflect.FloatKind:
				name := string(fd.Name())
				if name != "time_unix_nano" && name != "start_time_unix_nano" && name != "end_time_unix_nano" && name != "observed_time_unix_nano" && name != "flags" {
					m.Clear(fd)
				}
			}
			return true
		})
	}
	walk(message.ProtoReflect(), "")
}
