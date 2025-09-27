package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
)

type gzipReadCloser struct {
	*gzip.Reader
	io.Closer
}

func (gz gzipReadCloser) Close() error {
	return gz.Closer.Close()
}

// Decompressor middleware handles gzip-compressed requests
var Decompressor = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to decompress request", http.StatusInternalServerError)
			return
		}
		defer gzipReader.Close()

		r.Body = gzipReadCloser{gzipReader, r.Body}
		next.ServeHTTP(w, r)
	})
}
