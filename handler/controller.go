package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const namespace = "test_app"

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "Number of get requests.",
	},
	[]string{"path"},
)

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "response_status",
		Help:      "Status of HTTP response",
	},
	[]string{"status"},
)

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "http_response_time_seconds",
	Help:      "Duration of HTTP requests.",
}, []string{"path"})

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL
		log.Println("requested path", path)
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path.Path))
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCode := rw.statusCode

		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		totalRequests.WithLabelValues(path.Path).Inc()

		t := timer.ObserveDuration()
		log.Println("response time:", t)

	})
}

func init() {
	err := prometheus.Register(totalRequests)
	if err != nil {
		log.Panic(err)
	}
	err = prometheus.Register(responseStatus)
	if err != nil {
		log.Panic(err)
	}
	//err = prometheus.Register(httpDuration)
	//log.Println(err)
}

func main() {
	router := mux.NewRouter()
	router.Use(prometheusMiddleware)

	router.HandleFunc("/getinfo", GetInfo).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Serving requests on port 9000")
	err := http.ListenAndServe(":9000", router)
	log.Fatal(err)
}

func GetInfo(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	t := rand.Intn(100)
	time.Sleep(time.Duration(t) * time.Millisecond)
	err := json.NewEncoder(w).Encode("calling test api")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, err.Error())
	}
}
