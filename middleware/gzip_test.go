package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
)

func TestDecompressRequestMiddlewareSupportedEncodings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const plaintext = `{"model":"gpt-5.5","input":"hello"}`

	tests := []struct {
		name            string
		contentEncoding string
		body            []byte
	}{
		{name: "identity", body: []byte(plaintext)},
		{name: "gzip", contentEncoding: "gzip", body: gzipBytes(t, []byte(plaintext))},
		{name: "zlib deflate", contentEncoding: "deflate", body: zlibDeflateBytes(t, []byte(plaintext))},
		{name: "raw deflate", contentEncoding: "deflate", body: rawDeflateBytes(t, []byte(plaintext))},
		{name: "brotli", contentEncoding: "br", body: brotliBytes(t, []byte(plaintext))},
		{name: "zstd", contentEncoding: "zstd", body: zstdBytes(t, []byte(plaintext))},
		{name: "multi layer", contentEncoding: "gzip, br", body: brotliBytes(t, gzipBytes(t, []byte(plaintext)))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(DecompressRequestMiddleware())
			router.POST("/v1/responses/compact", func(c *gin.Context) {
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					t.Fatalf("ReadAll request body: %v", err)
				}
				if string(body) != plaintext {
					t.Fatalf("body = %q, want %q", body, plaintext)
				}
				if tt.contentEncoding != "" && c.GetHeader("Content-Encoding") != "" {
					t.Fatalf("Content-Encoding should be stripped after decode")
				}
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", bytes.NewReader(tt.body))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestDecompressRequestMiddlewareRejectsUnsupportedEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DecompressRequestMiddleware())
	router.POST("/", func(c *gin.Context) {
		t.Fatal("handler should not run for unsupported content encoding")
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("payload")))
	req.Header.Set("Content-Encoding", "compress")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func gzipBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("gzip Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip Close: %v", err)
	}
	return buf.Bytes()
}

func zlibDeflateBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("zlib Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib Close: %v", err)
	}
	return buf.Bytes()
}

func rawDeflateBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		t.Fatalf("flate NewWriter: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("flate Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("flate Close: %v", err)
	}
	return buf.Bytes()
}

func brotliBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("brotli Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("brotli Close: %v", err)
	}
	return buf.Bytes()
}

func zstdBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd NewWriter: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("zstd Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zstd Close: %v", err)
	}
	return buf.Bytes()
}
