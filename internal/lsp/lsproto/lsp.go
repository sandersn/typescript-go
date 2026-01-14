package lsproto

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/microsoft/typescript-go/internal/jsonrpc"
	"github.com/microsoft/typescript-go/internal/tspath"
)

type DocumentUri string // !!!
func (uri DocumentUri) FileName() string {
	if strings.HasPrefix(string(uri), "file://") {
		// First, normalize the URI by replacing backslashes with forward slashes
		uriStr := string(uri)
		uriStr = strings.ReplaceAll(uriStr, "\\", "/")

		// Handle malformed URIs with only 2 slashes (file://c:/path)
		// by ensuring we have 3 slashes for absolute paths (file:///c:/path)
		if strings.HasPrefix(uriStr, "file://") && !strings.HasPrefix(uriStr, "file:///") {
			// Check if what follows looks like an absolute path (drive letter on Windows, or / on Unix)
			rest := uriStr[7:] // Skip "file://"
			if len(rest) > 0 && (rest[0] == '/' || (len(rest) >= 2 && isLetter(rest[0]) && rest[1] == ':')) {
				// It's an absolute path, add the missing slash
				uriStr = "file:///" + rest
			}
		}

		parsed, err := url.Parse(uriStr)
		if err != nil {
			panic(fmt.Sprintf("invalid file URI: %s", uri))
		}

		// Unescape the path to handle URL-encoded characters like %3A for ":"
		path := parsed.Path
		if unescaped, err := url.PathUnescape(path); err == nil {
			path = unescaped
		}

		if parsed.Host != "" {
			return "//" + parsed.Host + path
		}
		return fixWindowsURIPath(path)
	}

	// Leave all other URIs escaped so we can round-trip them.

	scheme, path, ok := strings.Cut(string(uri), ":")
	if !ok {
		panic(fmt.Sprintf("invalid URI: %s", uri))
	}

	authority := "ts-nul-authority"
	if rest, ok := strings.CutPrefix(path, "//"); ok {
		authority, path, ok = strings.Cut(rest, "/")
		if !ok {
			panic(fmt.Sprintf("invalid URI: %s", uri))
		}
	}

	return "^/" + scheme + "/" + authority + "/" + path
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (uri DocumentUri) Path(useCaseSensitiveFileNames bool) tspath.Path {
	fileName := uri.FileName()
	return tspath.ToPath(fileName, "", useCaseSensitiveFileNames)
}

func fixWindowsURIPath(path string) string {
	if rest, ok := strings.CutPrefix(path, "/"); ok {
		if volume, rest, ok := tspath.SplitVolumePath(rest); ok {
			return volume + rest
		}
	}
	return path
}

type HasTextDocumentURI interface {
	TextDocumentURI() DocumentUri
}

type HasTextDocumentPosition interface {
	HasTextDocumentURI
	TextDocumentPosition() Position
}

type HasLocations interface {
	GetLocations() *[]Location
}

type HasLocation interface {
	GetLocation() Location
}

type URI string // !!!

type Method string

func unmarshalPtrTo[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %T: %w", (*T)(nil), err)
	}
	return &v, nil
}

func unmarshalValue[T any](data []byte) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return *new(T), fmt.Errorf("failed to unmarshal %T: %w", (*T)(nil), err)
	}
	return v, nil
}

func unmarshalAny(data []byte) (any, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal any: %w", err)
	}
	return v, nil
}

func unmarshalEmpty(data []byte) (any, error) {
	if len(data) != 0 {
		return nil, fmt.Errorf("expected empty, got: %s", string(data))
	}
	return nil, nil
}

func assertOnlyOne(message string, values ...bool) {
	count := 0
	for _, v := range values {
		if v {
			count++
		}
	}
	if count != 1 {
		panic(message)
	}
}

func assertAtMostOne(message string, values ...bool) {
	count := 0
	for _, v := range values {
		if v {
			count++
		}
	}
	if count > 1 {
		panic(message)
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

type requiredProp bool

func (v *requiredProp) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	*v = true
	return dec.SkipValue()
}

// Inspired by https://www.youtube.com/watch?v=dab3I-HcTVk

type RequestInfo[Params, Resp any] struct {
	_      [0]Params
	_      [0]Resp
	Method Method
}

func (info RequestInfo[Params, Resp]) UnmarshalResult(result any) (Resp, error) {
	if r, ok := result.(Resp); ok {
		return r, nil
	}

	raw, ok := result.(jsontext.Value)
	if !ok {
		return *new(Resp), fmt.Errorf("expected jsontext.Value, got %T", result)
	}

	r, err := unmarshalResult(info.Method, raw)
	if err != nil {
		return *new(Resp), err
	}
	return r.(Resp), nil
}

func (info RequestInfo[Params, Resp]) NewRequestMessage(id *jsonrpc.ID, params Params) *RequestMessage {
	return &RequestMessage{
		ID:     id,
		Method: info.Method,
		Params: params,
	}
}

type NotificationInfo[Params any] struct {
	_      [0]Params
	Method Method
}

func (info NotificationInfo[Params]) NewNotificationMessage(params Params) *RequestMessage {
	return &RequestMessage{
		Method: info.Method,
		Params: params,
	}
}

type Null struct{}

func (Null) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	data, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if string(data) != "null" {
		return fmt.Errorf("expected null, got %s", data)
	}
	return nil
}

func (Null) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.Null)
}

type clientCapabilitiesKey struct{}

func WithClientCapabilities(ctx context.Context, caps *ResolvedClientCapabilities) context.Context {
	return context.WithValue(ctx, clientCapabilitiesKey{}, caps)
}

func GetClientCapabilities(ctx context.Context) *ResolvedClientCapabilities {
	if caps, _ := ctx.Value(clientCapabilitiesKey{}).(*ResolvedClientCapabilities); caps != nil {
		return caps
	}
	return &ResolvedClientCapabilities{}
}

// PreferredMarkupKind returns the first (most preferred) markup kind from the given formats,
// or MarkupKindPlainText if the slice is empty.
func PreferredMarkupKind(formats []MarkupKind) MarkupKind {
	if len(formats) > 0 {
		return formats[0]
	}
	return MarkupKindPlainText
}
