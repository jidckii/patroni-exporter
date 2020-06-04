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

var metricXlogLocation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "patroni_xlog_location",
	Help: "Current xlog location (only applicable to masters)"}, []string{"role"})

var metricXlogReceivedLocation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "patroni_xlog_received_location",
	Help: "Current xlog received location (only applicable to replicas)"}, []string{"role"})

var metricXlogReplayedLocation = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "patroni_xlog_replayed_location",
	Help: "Current xlog replayed location (only applicable to replicas)"}, []string{"role"})

type XlogStatus struct {
	Location         float64 `json:"location"`
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
			metricState.DeleteLabelValues(state)
		}
	}
}

var POSSIBLE_ROLES = []string{"master", "replica"}

func setRole(status PatroniStatus) {
	for _, role := range POSSIBLE_ROLES {
		if status.Role == role {
			metricRole.WithLabelValues(role).Set(1)
		} else {
			metricRole.DeleteLabelValues(role)
		}
	}
}

func setXlogMetrics(status PatroniStatus) {
	if status.Role == "master" {
		metricXlogLocation.WithLabelValues(status.Role).Set(status.Xlog.Location)
		metricXlogReceivedLocation.DeleteLabelValues("replica")
		metricXlogReplayedLocation.DeleteLabelValues("replica")
	} else {
		metricXlogLocation.DeleteLabelValues("master")
		metricXlogReceivedLocation.WithLabelValues(status.Role).Set(status.Xlog.ReceivedLocation)
		metricXlogReplayedLocation.WithLabelValues(status.Role).Set(status.Xlog.ReplayedLocation)
	}
}

func updateMetrics(httpClient http.Client, url string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Print(getErr)
		return;
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
	prometheus.MustRegister(metricXlogLocation)
	prometheus.MustRegister(metricXlogReceivedLocation)
	prometheus.MustRegister(metricXlogReplayedLocation)

	go updateLoop()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9394", nil)
}
