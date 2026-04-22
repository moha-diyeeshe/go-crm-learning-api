package http // Keeps middleware in the same HTTP package as router wiring.

import (
	"log"      // Uses the standard logger to print request lines to terminal.
	"net/http" // Uses Handler, Request, and ResponseWriter types for middleware.
	"time"     // Uses time.Now and time.Since to measure request duration.
)

type statusRecorder struct { // Wraps ResponseWriter so we can capture final status code.
	http.ResponseWriter // Embeds net/http ResponseWriter so normal response writing still works.
	statusCode int      // Stores the HTTP status code we observe during WriteHeader.
}

func (r *statusRecorder) WriteHeader(code int) { // Overrides WriteHeader to record status before sending it.
	r.statusCode = code // Saves the status for logging after the handler finishes.
	r.ResponseWriter.WriteHeader(code) // Forwards the same status to the real client response writer.
}

func requestLogger(next http.Handler) http.Handler { // Builds middleware that wraps any downstream handler.
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { // Converts function into an http.Handler.
		startedAt := time.Now() // Captures start time so we can compute duration at the end.
		recorder := &statusRecorder{ // Creates our response wrapper to track status code.
			ResponseWriter: w,             // Keeps original writer so response body/headers still work.
			statusCode:     http.StatusOK, // Defaults to 200 if handler never explicitly calls WriteHeader.
		}

		next.ServeHTTP(recorder, req) // Passes request to next handler with wrapped writer.

		duration := time.Since(startedAt) // Computes total processing time for this request.
		log.Printf( // Prints one structured log line for easier debugging and tracing.
			"request method=%s path=%s status=%d duration=%s remote_addr=%s", // Defines consistent key=value style format.
			req.Method,           // Uses request method (GET/POST/etc.) from net/http Request.
			req.URL.Path,         // Uses URL path to show which endpoint was called.
			recorder.statusCode,  // Uses captured status so success/failure is visible.
			duration,             // Uses measured elapsed time to identify slow requests.
			req.RemoteAddr,       // Uses client remote address from the incoming TCP connection.
		)
	})
}
