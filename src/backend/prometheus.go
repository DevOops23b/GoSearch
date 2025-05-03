package main

import (
	"crypto/tls"
	//"crypto/x509" maybe use it later
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/cpu"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	cpuLoadPercentage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_load_percentage",
			Help: "Current cpu load in percent",
		},
	)

	certExpiryDays = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tls_certificate_expiry_days",
			Help: "Days until the tls certificate expires",
		},
		[]string{"domain"},
	)

	certValidity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tls_certificate_validity",
			Help: "Certificate validity (1 = valid, 0 = invalid)",
		},
		[]string{"domain"},
	)

	newUserCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "new_users_total_count",
			Help: "New users",
		},
		[]string{"hour_of_day", "day_of_week"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Custom response writer to track status
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     200,
		}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		// Record metrics after the request is processed
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)

		// Use actual status code
		httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(recorder.statusCode),
		).Inc()

	})
}

func startMonitoring() {

	// Start CPU monitoring
	go monitorCPU()

	// Start certificate monitoring
	go certificateMonitoring()
}

func monitorCPU() {
	for {
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			log.Printf("Error moitoring CPU: %v", err)
		} else if len(cpuPercent) > 0 {
			cpuLoadPercentage.Set(cpuPercent[0])
		}
		time.Sleep(30 * time.Second)
	}
}

func certificateMonitoring() {
	domains := []string{"gosearch.dk"}

	for {
		for _, domain := range domains {
			checkCertificate(domain)
		}
		time.Sleep(1 * time.Hour)
	}
}

func checkCertificate(domain string) {
	/* Can be added later for more advanced monitoring
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("Error loading system certification pool: %v", err)
		//rootCAs = x509.NewCertPool()
	}
	*/

	config := &tls.Config{
		InsecureSkipVerify:    true,
		ServerName: domain,
	}

	conn, err := tls.Dial("tcp", domain+":443", config)

	certValid := 0.0
	daysUntilExpiry := 0.0

	if err != nil {
		log.Printf("Certificate validation failed for %s: %v", domain, err)

	} else {
		defer conn.Close()

		if len(conn.ConnectionState().PeerCertificates) > 0 {
			cert := conn.ConnectionState().PeerCertificates[0]

			daysUntilExpiry = time.Until(cert.NotAfter).Hours() / 24

			if time.Now().After(cert.NotAfter) || time.Now().Before(cert.NotBefore) {
				log.Printf("Certificate for %s is outside validity period", domain)
			} else {
				if err := cert.VerifyHostname(domain); err != nil {
					log.Printf("Hostname verification failed for %s: %v", domain, err)
				} else {
					certValid = 1.0
				}
			}


			/*
			opts := x509.VerifyOptions{
				DNSName: domain,
				Roots:   rootCAs,
			}

			if _, err := cert.Verify(opts); err == nil {
				certValid = 1.0

			} else {
				log.Printf("Certificate chain validation failed for %s: %v", domain, err)
			}
		} else {
			log.Printf("No certificates found for %s", domain)
			*/
		} else {
			log.Printf("No certifcates found for %s", domain)
		}

	}

	certExpiryDays.WithLabelValues(domain).Set(daysUntilExpiry)
	certValidity.WithLabelValues(domain).Set(certValid)
}

// Updates the user counter with current hour and weekday
func incrementNewUserCounter() {
	now := time.Now()
	hourOfDay := strconv.Itoa(now.Hour())
	dayOfWeek := now.Weekday().String()

	newUserCounter.WithLabelValues(hourOfDay, dayOfWeek).Inc()
}
