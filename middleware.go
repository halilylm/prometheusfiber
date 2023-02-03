package prometheusfiber

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"

	"log"
	"strconv"
	"time"
)

var defaultMetricPath = "/metrics"
var defaultSubSystem = "fiber"

const (
	_          = iota
	KB float64 = 1 << (10 * iota)
	MB
)

// reqDurBuckets is the buckets for request duration.
var reqDurBuckets = prometheus.DefBuckets

// reqSizeBuckets is the buckets for request size.
var reqSizeBuckets = []float64{1.0 * KB, 2.0 * KB, 5.0 * KB, 10.0 * KB, 100 * KB, 500 * KB, 1.0 * MB, 2.5 * MB, 5.0 * MB, 10.0 * MB}

// resSizeBuckets is the buckets for response size.
var resSizeBuckets = []float64{1.0 * KB, 2.0 * KB, 5.0 * KB, 10.0 * KB, 100 * KB, 500 * KB, 1.0 * MB, 2.5 * MB, 5.0 * MB, 10.0 * MB}

// Metric defines an individual metric collection.
type Metric struct {
	Collector   prometheus.Collector
	ID          string
	Name        string
	Description string
	Type        string
	Args        []string
	Buckets     []float64
}

// reqCount collects request metrics.
var reqCount = Metric{
	ID:          "reqCount",
	Name:        "requests_total",
	Description: "Total number of requests by status code and HTTP method",
	Type:        "counter_vec",
	Args:        []string{"code", "method", "host", "url"},
}

// reqDur collects request latency metrics.
var reqDur = Metric{
	ID:          "reqDur",
	Name:        "request_duration_seconds",
	Description: "The HTTP request latencies in seconds.",
	Args:        []string{"code", "method", "url"},
	Type:        "histogram_vec",
	Buckets:     reqDurBuckets,
}

// respSize collects response size metrics.
var respSize = Metric{
	ID:          "respSize",
	Name:        "response_size_bytes",
	Description: "The HTTP response sizes in bytes.",
	Args:        []string{"code", "method", "url"},
	Type:        "histogram_vec",
	Buckets:     resSizeBuckets,
}

// reqSize collects request size metrics.
var reqSize = Metric{
	ID:          "reqSize",
	Name:        "request_size_bytes",
	Description: "The HTTP request sizes in bytes.",
	Args:        []string{"code", "method", "url"},
	Type:        "histogram_vec",
	Buckets:     reqSizeBuckets,
}

// defaultMetrics consists of default metrics.
var defaultMetrics = []*Metric{
	&reqCount,
	&reqDur,
	&respSize,
	&reqSize,
}

// Prometheus contains metric collection instruments.
type Prometheus struct {
	reqCount      *prometheus.CounterVec
	reqDur        *prometheus.HistogramVec
	respSize      *prometheus.HistogramVec
	reqSize       *prometheus.HistogramVec
	router        *fiber.App
	listenAddress string

	metricsList []*Metric
	metricsPath string
	subsystem   string
	skip        []string
}

// Options defines
type Options struct {
	SubSystem  string
	MetricPath string
	Skip       []string
}

type Option func(o *Options)

// WithSubSystem defines subsystem.
func WithSubSystem(subsystem string) Option {
	return func(o *Options) {
		o.SubSystem = subsystem
	}
}

// WithMetricPath define path where metric will be published.
func WithMetricPath(path string) Option {
	return func(o *Options) {
		o.MetricPath = path
	}
}

// WithSkipURL will skip urls
func WithSkipURL(urls ...string) Option {
	return func(o *Options) {
		o.Skip = urls
	}
}

// NewOptions is a factory function to generate Options.
func NewOptions(opts ...Option) Options {
	options := Options{
		SubSystem:  defaultSubSystem,
		MetricPath: defaultMetricPath,
		Skip:       make([]string, 0),
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

// NewPrometheus is a factory function for prometheus.
func NewPrometheus(opts ...Option) *Prometheus {
	options := NewOptions(opts...)

	metricsList := make([]*Metric, 0, len(defaultMetrics))
	metricsList = append(metricsList, defaultMetrics...)

	p := &Prometheus{
		metricsList: metricsList,
		metricsPath: options.MetricPath,
		subsystem:   options.SubSystem,
		skip:        options.Skip,
	}

	p.registerMetrics()

	return p
}

// NewMetric  is a factory function to create an individual metric.
func NewMetric(m *Metric, subsystem string) prometheus.Collector {
	var metric prometheus.Collector

	switch m.Type {
	case "counter_vec":
		metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "counter":
		metric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "gauge_vec":
		metric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "gauge":
		metric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "histogram_vec":
		metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
				Buckets:   m.Buckets,
			},
			m.Args,
		)
	case "histogram":
		metric = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
				Buckets:   m.Buckets,
			},
		)
	case "summary_vec":
		metric = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "summary":
		metric = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	}

	return metric
}

// Middleware is the prometheus middleware.
func (ps *Prometheus) Middleware(ctx *fiber.Ctx) error {
	if ctx.Path() == ps.metricsPath {
		return ctx.Next()
	}

	for _, skip := range ps.skip {
		if ctx.Path() == skip {
			return ctx.Next()
		}
	}

	start := time.Now()
	reqSize := computeApproximateRequestSize(ctx.Request())
	method := ctx.Route().Method

	err := ctx.Next()

	status := fiber.StatusInternalServerError
	if err != nil {
		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
		}
	} else {
		status = ctx.Response().StatusCode()
	}

	elapsed := float64(time.Since(start)) / float64(time.Second)

	url := ctx.Route().Path

	statusStr := strconv.Itoa(status)
	ps.reqDur.WithLabelValues(statusStr, method, url).Observe(elapsed)
	ps.reqCount.WithLabelValues(statusStr, method, ctx.Hostname(), url).Inc()
	ps.reqSize.WithLabelValues(statusStr, method, url).Observe(float64(reqSize))
	ps.respSize.WithLabelValues(statusStr, method, url).Observe(float64(computeApproximateResponseSize(ctx.Response())))

	return err
}

// SetMetricsPath sets metric path.
func (ps *Prometheus) SetMetricsPath(app *fiber.App) {
	if ps.listenAddress != "" {
		ps.router.Get(ps.metricsPath, adaptor.HTTPHandler(promhttp.Handler()))
		ps.runServer()
	} else {
		app.Get(ps.metricsPath, adaptor.HTTPHandler(promhttp.Handler()))
	}
}

// runServer publish metrics in a different server.
func (ps *Prometheus) runServer() {
	if ps.listenAddress != "" {
		go func() {
			if err := ps.router.Listen(ps.listenAddress); err != nil {
				log.Fatalln(err)
			}
		}()
	}
}

// Use registers a prometheus middleware on a fiber app.
func (ps *Prometheus) Use(app *fiber.App) {
	app.Use(ps.Middleware)
	ps.SetMetricsPath(app)
}

// registerMetrics register metrics on prometheus.
func (ps *Prometheus) registerMetrics() {
	for _, metricDef := range ps.metricsList {
		metric := NewMetric(metricDef, ps.subsystem)
		if err := prometheus.Register(metric); err != nil {
			return
		}
		switch metricDef {
		case &reqCount:
			ps.reqCount = metric.(*prometheus.CounterVec)
		case &reqDur:
			ps.reqDur = metric.(*prometheus.HistogramVec)
		case &respSize:
			ps.respSize = metric.(*prometheus.HistogramVec)
		case &reqSize:
			ps.reqSize = metric.(*prometheus.HistogramVec)
		}
		metricDef.Collector = metric
	}
}

// computeApproximateRequestSize calculates size of the request body.
func computeApproximateRequestSize(r *fasthttp.Request) int {
	size := len(r.Body()) + 2
	r.Header.VisitAll(func(key, value []byte) {
		size += len(key) + len(value) + 2
	})
	return size
}

// computeApproximateResponseSize calculates size of the response body.
func computeApproximateResponseSize(r *fasthttp.Response) int {
	size := len(r.Body()) + 2
	r.Header.VisitAll(func(key, value []byte) {
		size += len(key) + len(value) + 2
	})
	return size
}
