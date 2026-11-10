package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/apperror"
)

type ctxKey int

const requestIDKey ctxKey = iota

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			writeProblem(w, r, fmt.Errorf("panic: %v\n%s", v, debug.Stack()))
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		logger.LogAttrs(r.Context(), slog.LevelInfo, "requisicao",
			slog.String("request_id", requestIDFrom(r.Context())),
			slog.String("method", r.Method),
			slog.String("route", r.Pattern),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
		)
	})
}

type headerCapture struct {
	header http.Header
	status int
}

func (c *headerCapture) Header() http.Header {
	return c.header
}

func (c *headerCapture) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c *headerCapture) WriteHeader(code int) {
	c.status = code
}

func withRouteProblems(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}

		c := &headerCapture{header: http.Header{}}
		h.ServeHTTP(c, r)
		if c.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", c.header.Get("allow"))
			writeProblem(w, r, apperror.MethodNotAllowed("metodo nao permitido nesta rota"))
			return
		}
		writeProblem(w, r, apperror.NotFound("rota nao encontrada"))
	})
}
