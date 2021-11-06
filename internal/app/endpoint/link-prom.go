package endpoint

import (
	"github.com/prometheus/client_golang/prometheus"
)

//PromIf - Prometh interface
type PromIf interface {
	New() PromIf
	UpdateHist(method string, dtime float64)
	UpdateCtr()
}

// префикс перед лейблами
const (
	Namespace   = "weblinkmetrics"
	LabelMethod = "method"
	LabelStatus = "status"
)

//Prom - prometheus counters struct
type Prom struct {
	latencyHistogram *prometheus.HistogramVec
	authCounter      prometheus.Counter
}

//New - init counters
func (p *Prom) New() PromIf {

	prom := &Prom{}
	// prometheus type: histogram
	prom.latencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "latency",
			Help:      "The distribution of the latencies",
			Buckets:   []float64{0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000},
		},
		[]string{LabelMethod})

	// prometheus type: counter
	prom.authCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "tokens_issued",
			Help:      "The number of authentifications it shows user activity",
		})

	prometheus.MustRegister(prom.latencyHistogram)
	prometheus.MustRegister(prom.authCounter)

	return prom
}

//UpdateHist - update prom histogram
func (p *Prom) UpdateHist(method string, dtime float64) {
	p.latencyHistogram.With(prometheus.Labels{LabelMethod: method}).Observe(dtime)
}

//UpdateCtr - update prom auth counter
func (p *Prom) UpdateCtr() {
	p.authCounter.Inc()
}
