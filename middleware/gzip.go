package middleware

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
)

type readCloser struct {
	io.Reader
	closeFn func() error
}

func (rc *readCloser) Close() error {
	if rc.closeFn != nil {
		return rc.closeFn()
	}
	return nil
}

func DecompressRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		maxMB := constant.MaxRequestBodyMB
		if maxMB <= 0 {
			maxMB = 32
		}
		maxBytes := int64(maxMB) << 20

		origBody := c.Request.Body
		wrapMaxBytes := func(body io.ReadCloser) io.ReadCloser {
			return http.MaxBytesReader(c.Writer, body, maxBytes)
		}

		decodedBody, err := decodeRequestBody(origBody, c.GetHeader("Content-Encoding"))
		if err != nil {
			_ = origBody.Close()
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid compressed request body: %v", err),
			})
			return
		}

		if decodedBody == origBody {
			// Even for uncompressed bodies, enforce a max size to avoid huge request allocations.
			c.Request.Body = wrapMaxBytes(origBody)
		} else {
			// Replace the request body with the decompressed data, and enforce a max size (post-decompression).
			c.Request.Body = wrapMaxBytes(decodedBody)
			c.Request.Header.Del("Content-Encoding")
			c.Request.Header.Del("Content-Length")
			c.Request.ContentLength = -1
		}

		// Continue processing the request
		c.Next()
	}
}

type peekableBody struct {
	*bufio.Reader
	closer io.Closer
}

func (p *peekableBody) Close() error {
	return p.closer.Close()
}

func decodeRequestBody(body io.ReadCloser, contentEncoding string) (io.ReadCloser, error) {
	encodings := parseContentEncodings(contentEncoding)
	if len(encodings) == 0 {
		return body, nil
	}
	for i := len(encodings) - 1; i >= 0; i-- {
		var err error
		body, err = wrapDecodedBody(body, encodings[i])
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

func parseContentEncodings(contentEncoding string) []string {
	if contentEncoding == "" {
		return nil
	}
	parts := strings.Split(contentEncoding, ",")
	encodings := make([]string, 0, len(parts))
	for _, raw := range parts {
		encoding := strings.TrimSpace(strings.ToLower(raw))
		switch encoding {
		case "", "identity":
			continue
		default:
			encodings = append(encodings, encoding)
		}
	}
	return encodings
}

func wrapDecodedBody(body io.ReadCloser, encoding string) (io.ReadCloser, error) {
	switch encoding {
	case "gzip", "x-gzip":
		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			_ = body.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return &readCloser{
			Reader: gzipReader,
			closeFn: func() error {
				_ = gzipReader.Close()
				return body.Close()
			},
		}, nil
	case "deflate":
		return newDeflateReadCloser(body)
	case "br":
		reader := brotli.NewReader(body)
		return &readCloser{
			Reader:  reader,
			closeFn: body.Close,
		}, nil
	case "zstd", "zst":
		decoder, err := zstd.NewReader(body)
		if err != nil {
			_ = body.Close()
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
		return &readCloser{
			Reader: decoder,
			closeFn: func() error {
				decoder.Close()
				return body.Close()
			},
		}, nil
	default:
		_ = body.Close()
		return nil, fmt.Errorf("unsupported content encoding %q", encoding)
	}
}

func newDeflateReadCloser(body io.ReadCloser) (io.ReadCloser, error) {
	pb := &peekableBody{Reader: bufio.NewReader(body), closer: body}
	if looksLikeZlib(pb) {
		zlibReader, err := zlib.NewReader(pb)
		if err != nil {
			_ = pb.Close()
			return nil, fmt.Errorf("failed to create zlib deflate reader: %w", err)
		}
		return &readCloser{
			Reader: zlibReader,
			closeFn: func() error {
				_ = zlibReader.Close()
				return pb.Close()
			},
		}, nil
	}
	deflateReader := flate.NewReader(pb)
	return &readCloser{
		Reader: deflateReader,
		closeFn: func() error {
			_ = deflateReader.Close()
			return pb.Close()
		},
	}, nil
}

func looksLikeZlib(r interface{ Peek(int) ([]byte, error) }) bool {
	header, err := r.Peek(2)
	if err != nil {
		return false
	}
	cmf, flg := int(header[0]), int(header[1])
	return cmf&0x0f == 8 && cmf>>4 <= 7 && (cmf*256+flg)%31 == 0
}
