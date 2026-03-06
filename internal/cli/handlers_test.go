package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"

	"github.com/andinger/vaultquery/internal/config"
	"github.com/andinger/vaultquery/internal/executor"
)

func newCmdWithFormatFlag(formatValue string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", formatValue, "")
	return cmd
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name       string
		flagValue  string
		cfgFormat  string
		want       string
		wantErr    bool
		errContain string
	}{
		{name: "default is json", want: "json"},
		{name: "config sets toon", cfgFormat: "toon", want: "toon"},
		{name: "config sets json", cfgFormat: "json", want: "json"},
		{name: "flag overrides config", flagValue: "json", cfgFormat: "toon", want: "json"},
		{name: "flag sets toon", flagValue: "toon", want: "toon"},
		{name: "invalid format from flag", flagValue: "xml", wantErr: true, errContain: "unsupported"},
		{name: "invalid format from config", cfgFormat: "yaml", wantErr: true, errContain: "unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdWithFormatFlag(tt.flagValue)
			cfg := &config.Config{Format: tt.cfgFormat}

			got, err := resolveFormat(cmd, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeResult_JSON(t *testing.T) {
	result := &executor.Result{
		Mode:    "LIST",
		Results: []map[string]any{{"file": "test.md"}},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := encodeResult(result, "json")

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var decoded executor.Result
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if decoded.Mode != "LIST" {
		t.Errorf("mode = %q, want LIST", decoded.Mode)
	}
	if len(decoded.Results) != 1 {
		t.Errorf("got %d results, want 1", len(decoded.Results))
	}
}

func TestEncodeResult_TOON(t *testing.T) {
	result := &executor.Result{
		Mode:    "LIST",
		Results: []map[string]any{{"file": "test.md"}},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := encodeResult(result, "toon")

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	output := buf.String()
	if output == "" {
		t.Fatal("expected non-empty TOON output")
	}

	// Verify it's valid TOON by decoding it
	decoded, err := toon.Decode([]byte(strings.TrimSpace(output)))
	if err != nil {
		t.Fatalf("invalid TOON output: %v\n%s", err, output)
	}

	m, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}
	if m["mode"] != "LIST" {
		t.Errorf("mode = %v, want LIST", m["mode"])
	}
}
