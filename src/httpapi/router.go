package httpapi

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// NewRouter registers every route of the API.
func NewRouter(transfers *TransferHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/transfers", transfers.Create)
	mux.HandleFunc("GET /api/transfers/{transferId}", transfers.Get)
	mux.HandleFunc("POST /api/transfers/{transferId}/cancel", transfers.Cancel)

	// Unknown routes still answer with the required error shape.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "route "+r.Method+" "+r.URL.Path+" not found")
	})

	return logRequests(recoverPanics(mux))
}

// recoverPanics turns a panic into a 500 with the required error shape instead of a dropped connection.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic: %v\n%s", recovered, debug.Stack())
				writeError(w, http.StatusInternalServerError, codeInternal, "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// logRequests prints one line per request: method, path, status and duration.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, recorder.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
