package proto

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestEncodeExecMcpStateResultUsesCurrentCursorSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"patchText":{"type":"string"}},"required":["patchText"]}`)
	payload := EncodeExecMcpStateResult(7, "exec-7", []string{"cliproxyapi-plus"}, []McpToolDef{{
		Name:        "apply_patch",
		Description: "Apply a patch",
		InputSchema: schema,
	}})

	execClient := requireBytesField(t, payload, 2)
	result := requireBytesField(t, execClient, ECM_McpStateExecResult)
	success := requireBytesField(t, result, 1)
	server := requireBytesField(t, success, 1)

	if got := string(requireBytesField(t, server, 1)); got != "cliproxyapi-plus" {
		t.Fatalf("server_name = %q", got)
	}
	if got := string(requireBytesField(t, server, 2)); got != "cliproxyapi-plus" {
		t.Fatalf("server_identifier = %q", got)
	}
	tool := requireBytesField(t, server, 5)
	if got := string(requireBytesField(t, server, 7)); got != "connected" {
		t.Fatalf("status = %q", got)
	}
	if hasField(server, 3) || hasField(server, 4) {
		t.Fatal("tools or status encoded with obsolete McpStateServer field numbers")
	}

	if got := string(requireBytesField(t, tool, 1)); got != "apply_patch" {
		t.Fatalf("tool name = %q", got)
	}
	if got := string(requireBytesField(t, tool, 4)); got != "cliproxyapi-plus" {
		t.Fatalf("provider_identifier = %q", got)
	}
	if got := string(requireBytesField(t, tool, 5)); got != "apply_patch" {
		t.Fatalf("tool_name = %q", got)
	}
	if got := string(requireBytesField(t, tool, 6)); got != string(schema) {
		t.Fatalf("input_schema_json = %q", got)
	}
	decoded, err := ProtobufValueBytesToJSON(requireBytesField(t, tool, 3))
	if err != nil {
		t.Fatalf("input_schema is not a protobuf Value: %v", err)
	}
	decodedObject, ok := decoded.(map[string]any)
	if !ok || decodedObject["type"] != "object" {
		t.Fatalf("decoded input_schema = %#v", decoded)
	}
}

func requireBytesField(t *testing.T, message []byte, wanted protowire.Number) []byte {
	t.Helper()
	for len(message) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(message)
		if tagLength < 0 {
			t.Fatalf("invalid protobuf tag: %v", protowire.ParseError(tagLength))
		}
		message = message[tagLength:]
		if wireType == protowire.BytesType {
			value, valueLength := protowire.ConsumeBytes(message)
			if valueLength < 0 {
				t.Fatalf("invalid bytes field %d: %v", number, protowire.ParseError(valueLength))
			}
			if number == wanted {
				return value
			}
			message = message[valueLength:]
			continue
		}
		fieldLength := protowire.ConsumeFieldValue(number, wireType, message)
		if fieldLength < 0 {
			t.Fatalf("invalid field %d: %v", number, protowire.ParseError(fieldLength))
		}
		message = message[fieldLength:]
	}
	t.Fatalf("missing bytes field %d", wanted)
	return nil
}

func hasField(message []byte, wanted protowire.Number) bool {
	for len(message) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(message)
		if tagLength < 0 {
			return false
		}
		message = message[tagLength:]
		if number == wanted {
			return true
		}
		fieldLength := protowire.ConsumeFieldValue(number, wireType, message)
		if fieldLength < 0 {
			return false
		}
		message = message[fieldLength:]
	}
	return false
}
