package opencodeadapter

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestACPFrameBoundRejected(t *testing.T) {
	big := "{\"jsonrpc\":\"2.0\",\"method\":\"session/update\",\"params\":{\"pad\":\"" + strings.Repeat("x", acpMaxMessage) + "\"}}\n"
	r := bufio.NewReaderSize(strings.NewReader(big), acpMaxMessage+2)
	if _, err := readACP(r); err == nil {
		t.Fatal("accepted oversized frame")
	}
}

func TestACPFrameMidMessageEOFRejected(t *testing.T) {
	r := bufio.NewReaderSize(strings.NewReader(`{"jsonrpc":"2.0","method":"x"`), acpMaxMessage+2)
	if _, err := readACP(r); err == nil || err.Error() == "EOF" {
		t.Fatalf("mid-message EOF not rejected as partial: %v", err)
	}
}

func TestACPFrameInvalidJSONAndWrongVersion(t *testing.T) {
	for _, bad := range []string{"not json\n", `{"jsonrpc":"1.0","id":1,"result":{}}` + "\n"} {
		r := bufio.NewReaderSize(strings.NewReader(bad), acpMaxMessage+2)
		if _, err := readACP(r); err == nil {
			t.Fatalf("accepted invalid frame %q", bad)
		}
	}
}

func TestACPRoundTripPreservesIDAndParams(t *testing.T) {
	var buf bytes.Buffer
	if err := writeACP(&buf, rpcRequest{JSONRPC: "2.0", ID: 7, Method: "session/prompt", Params: map[string]any{"sessionId": "ses_1"}}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > acpMaxMessage {
		t.Fatal("bounded write exceeded limit")
	}
	r := bufio.NewReaderSize(&buf, acpMaxMessage+2)
	m, err := readACP(r)
	if err != nil {
		t.Fatal(err)
	}
	if !m.isRequest() || !m.idAsInt(7) || m.Method != "session/prompt" {
		t.Fatalf("round trip mismatch: %+v", m)
	}
}

// newlineMarshaler emits a JSON document containing a literal newline, which a
// newline-delimited framing must reject rather than corrupt the stream.
type newlineMarshaler struct{}

func (newlineMarshaler) MarshalJSON() ([]byte, error) { return []byte("{\"a\":\"x\ny\"}"), nil }

func TestACPWriteRejectsMultiline(t *testing.T) {
	var buf bytes.Buffer
	if err := writeACP(&buf, newlineMarshaler{}); err == nil {
		t.Fatal("accepted embedded newline")
	}
}
