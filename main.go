package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "patroni_state", Help: "Current Patroni state"}, []string{"state"})

var metricRole = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "patroni_role", Help: "Current database role"}, []string{"role"})

var metricXlogReceivedLocation = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "patroni_xlog_received_location", Help: "Current xlog received location"})

var metricXlogReplayedLocation = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "patroni_xlog_replayed_location", Help: "Current xlog replayed location"})

type XlogStatus struct {
	ReceivedLocation float64 `json:"received_location"`
	ReplayedLocation float64 `json:"replayed_location"`
}

type PatroniStatus struct {
	State string     `json:"state"`
	Role  string     `json:"role"`
	Xlog  XlogStatus `json:"xlog"`
}

var POSSIBLE_STATES = []string{"running", "rejecting connections", "not responding", "unknown"}

func setState(status PatroniStatus) {
	for _, state := range POSSIBLE_STATES {
		if status.State == state {
			metricState.WithLabelValues(state).Set(1)
		} else {
			metricState.WithLabelValues(state).Set(0)
		}
	}
}

var POSSIBLE_ROLES = []string{"master", "replica"}

func setRole(status PatroniStatus) {
	for _, role := range POSSIBLE_ROLES {
		if status.Role == role {
			metricRole.WithLabelValues(role).Set(1)
		} else {
			metricRole.WithLabelValues(role).Set(0)
		}
	}
}

func setXlogMetrics(status PatroniStatus) {
	metricXlogReceivedLocation.Set(status.Xlog.ReceivedLocation)
	metricXlogReplayedLocation.Set(status.Xlog.ReplayedLocation)
}

func updateMetrics(httpClient http.Client, url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	status := PatroniStatus{}
	jsonErr := json.Unmarshal(body, &status)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	setState(status)
	setRole(status)
	setXlogMetrics(status)
}

func updateLoop() {
	url := "http://localhost:8008/patroni"
	httpClient := http.Client{Timeout: time.Second * 2}

	for {
		updateMetrics(httpClient, url)

		time.Sleep(time.Duration(5) * time.Second)
	}
}

func main() {
	prometheus.MustRegister(metricState)
	prometheus.MustRegister(metricRole)
	prometheus.MustRegister(metricXlogReceivedLocation)
	prometheus.MustRegister(metricXlogReplayedLocation)

	go updateLoop()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9394", nil)
}
