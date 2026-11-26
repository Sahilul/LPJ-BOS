package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OM adalah map JSON yang menjaga urutan kunci (seperti array asosiatif PHP).
type OM []KV

type KV struct {
	K string
	V any
}

func (o OM) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, kv := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		k, _ := jsonMarshal(kv.K)
		b.Write(k)
		b.WriteByte(':')
		v, err := jsonMarshal(kv.V)
		if err != nil {
			return nil, err
		}
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// McpArgs adalah argumen tool hasil decode JSON.
type McpArgs map[string]any

func (a McpArgs) S(k string) string {
	switch v := a[k].(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "1"
		}
		return ""
	case json.Number:
		return v.String()
	default:
		return toStr(v)
	}
}

func (a McpArgs) I(k string) int64   { return toInt(a.S(k)) }
func (a McpArgs) F(k string) float64 { return toFloat(a.S(k)) }
func (a McpArgs) Has(k string) bool  { _, ok := a[k]; return ok }

// McpTool adalah satu tool MCP.
type McpTool struct {
	Nama      string
	Akses     string
	Deskripsi string
	Skema     any
	Handler   func(a McpArgs) (any, error)
}

var mcpGrup [][]McpTool

// RegisterMcp dipanggil dari init() file tools; urutan grup mengikuti order.
func RegisterMcp(order int, tools func() []McpTool) {
	mcpRegs = append(mcpRegs, mcpReg{order, tools})
}

type mcpReg struct {
	order int
	fn    func() []McpTool
}

var (
	mcpRegs  []mcpReg
	mcpTools []McpTool
	mcpIndex map[string]int
)

func mcpSemua() []McpTool {
	if mcpIndex != nil {
		return mcpTools
	}
	regs := append([]mcpReg(nil), mcpRegs...)
	for i := 1; i < len(regs); i++ {
		for j := i; j > 0 && regs[j].order < regs[j-1].order; j-- {
			regs[j], regs[j-1] = regs[j-1], regs[j]
		}
	}
	mcpIndex = map[string]int{}
	for _, r := range regs {
		for _, t := range r.fn() {
			if i, ok := mcpIndex[t.Nama]; ok {
				mcpTools[i] = t
				continue
			}
			mcpIndex[t.Nama] = len(mcpTools)
			mcpTools = append(mcpTools, t)
		}
	}
	return mcpTools
}

func mcpCari(nama string) *McpTool {
	mcpSemua()
	if i, ok := mcpIndex[nama]; ok {
		return &mcpTools[i]
	}
	return nil
}

// Pembantu skema JSON.
func mcpObj(props OM, wajib ...string) OM {
	var p any = props
	if len(props) == 0 {
		p = struct{}{}
	}
	s := OM{{"type", "object"}, {"properties", p}}
	if len(wajib) > 0 {
		s = append(s, KV{"required", wajib})
	}
	return s
}

func mcpS(desc string) OM { return OM{{"type", "string"}, {"description", desc}} }
func mcpI(desc string) OM { return OM{{"type", "integer"}, {"description", desc}} }
func mcpN(desc string) OM { return OM{{"type", "number"}, {"description", desc}} }
func mcpB(desc string) OM { return OM{{"type", "boolean"}, {"description", desc}} }

// mcpError adalah galat biasa (setara \RuntimeException di PHP).
func mcpError(format string, a ...any) error { return fmt.Errorf(format, a...) }

func init() {
	McpHandler = http.HandlerFunc(mcpHTTP)
}

func mcpJSON(w http.ResponseWriter, kode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(kode)
	b, _ := jsonMarshal(data)
	_, _ = w.Write(b)
}

func mcpErrObj(id any, kode int, pesan string) OM {
	return OM{{"jsonrpc", "2.0"}, {"id", id}, {"error", OM{{"code", kode}, {"message", pesan}}}}
}

func mcpHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		mcpJSON(w, 405, OM{{"jsonrpc", "2.0"}, {"id", nil}, {"error", OM{
			{"code", -32000},
			{"message", "Gunakan POST."},
			{"data", OM{{"jumlah_tool", len(mcpSemua())}}},
		}}})
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	defer lockApp(true)()
	token := VerifyToken(TokenDariHeader(r))
	if token == nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="lpjbos"`)
		mcpJSON(w, 401, mcpErrObj(nil, -32001, "Token API tidak valid."))
		return
	}
	SentuhToken(token)

	raw = bytes.TrimSpace(raw)
	var body any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		mcpJSON(w, 400, mcpErrObj(nil, -32700, "JSON tidak valid."))
		return
	}
	var pesan []any
	batch := false
	switch b := body.(type) {
	case []any:
		batch, pesan = true, b
	case map[string]any:
		pesan = []any{b}
	default:
		mcpJSON(w, 400, mcpErrObj(nil, -32700, "JSON tidak valid."))
		return
	}

	jawaban := []any{}
	for _, m := range pesan {
		if msg, ok := m.(map[string]any); ok {
			if res := mcpHandle(token, msg); res != nil {
				jawaban = append(jawaban, res)
			}
		}
	}
	if len(jawaban) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if batch {
		mcpJSON(w, 200, jawaban)
		return
	}
	mcpJSON(w, 200, jawaban[0])
}

const mcpVersi = "2024-11-05"

func mcpOk(id any, result any) OM {
	return OM{{"jsonrpc", "2.0"}, {"id", id}, {"result", result}}
}

func mcpHandle(token Row, msg map[string]any) any {
	id := msg["id"]
	method, _ := msg["method"].(string)
	params, _ := msg["params"].(map[string]any)
	if method == "" {
		return mcpErrObj(id, -32600, "Pesan bukan permintaan JSON-RPC.")
	}
	_, adaID := msg["id"]

	switch method {
	case "initialize":
		return mcpOk(id, OM{
			{"protocolVersion", mcpVersi},
			{"capabilities", OM{{"tools", OM{{"listChanged", false}}}}},
			{"serverInfo", OM{{"name", "lpjbos"}, {"version", "1.0.0"}}},
			{"instructions", "Server MCP LPJ BOS: baca dan catat data RKAS, penerimaan, belanja, mutasi kas, BKU, undangan & notulen rapat, dan laporan BOSP."},
		})
	case "ping":
		return mcpOk(id, struct{}{})
	case "tools/list":
		out := []any{}
		for _, t := range mcpSemua() {
			skema := t.Skema
			if skema == nil {
				skema = mcpObj(nil)
			}
			out = append(out, OM{{"name", t.Nama}, {"description", t.Deskripsi}, {"inputSchema", skema}})
		}
		return mcpOk(id, OM{{"tools", out}})
	case "tools/call":
		return mcpPanggil(token, id, params)
	}
	if !adaID {
		return nil
	}
	return mcpErrObj(id, -32601, "Metode tidak dikenal: "+method)
}

func mcpHasil(id any, data any, galat bool) OM {
	teks, ok := data.(string)
	if !ok {
		teks = jsonPretty(data)
	}
	return mcpOk(id, OM{{"content", []any{OM{{"type", "text"}, {"text", teks}}}}, {"isError", galat}})
}

func mcpPanggil(token Row, id any, params map[string]any) (res any) {
	nama, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}
	tool := mcpCari(nama)
	if tool == nil {
		LogAPI(token, nama, args, false, "Tool tidak dikenal")
		return mcpErrObj(id, -32602, "Tool tidak dikenal: "+nama)
	}
	akses := token.S("akses")
	if akses == "" {
		akses = AksesBaca
	}
	if tool.Akses == AksesTulis && akses != AksesTulis {
		LogAPI(token, nama, args, false, "Token hanya baca")
		return mcpHasil(id, "Token ini hanya boleh membaca. Buat token dengan akses tulis untuk mencatat data.", true)
	}

	gagal := func(err error) OM {
		var ve *ValidationError
		if errors.As(err, &ve) {
			pesan := "Isian belum valid: " + strings.Join(validationMessages(ve), " ")
			LogAPI(token, nama, args, false, pesan)
			return mcpHasil(id, pesan, true)
		}
		LogAPI(token, nama, args, false, err.Error())
		return mcpHasil(id, "Gagal: "+err.Error(), true)
	}
	defer func() {
		if p := recover(); p != nil {
			if err, ok := p.(error); ok {
				res = gagal(err)
				return
			}
			res = gagal(fmt.Errorf("%v", p))
		}
	}()

	data, err := tool.Handler(McpArgs(args))
	if err != nil {
		return gagal(err)
	}
	LogAPI(token, nama, args, true, "")
	return mcpHasil(id, data, false)
}
