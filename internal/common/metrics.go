package common

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(port int) *http.Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: promhttp.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return srv
}
