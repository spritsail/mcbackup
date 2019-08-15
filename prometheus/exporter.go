package prometheus

import (
	"fmt"
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/provider"
)

var log = logrus.WithField("prefix", "prometheus")

func Serve(addr string, opts config.Options, prov provider.Provider) error {
	// Collect metrics for the provided backup provider
	collector := newBackupCollector(prov, opts)
	prom.MustRegister(collector)

	log.Info("serving metrics at " + addr)

	promHandler := promhttp.Handler()
	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		log.WithField("client", r.RemoteAddr).
			WithField("path", r.URL.Path).
			Info(fmt.Sprintf("request %s %s", r.Method, r.URL.Path))
		promHandler.ServeHTTP(w, r)
	}

	http.Handle("/metrics", handler)
	return http.ListenAndServe(addr, nil)
}
