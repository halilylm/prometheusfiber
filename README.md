
# Prometheus Middleware

Prometheus middleware for Fiber v2.

## Installation

```bash
go get github.com/halilylm/prometheusfiber@v0.1.0
```

## Usage/Examples

Each option is optional. Use what you need, if you don't use, it will fall back to default values.
```go
app := fiber.New()
middleware := prometheusfiber.NewPrometheus(
    prometheusfiber.WithSubSystem("fiber"), // define subsystem, default "fiber"
    prometheusfiber.WithMetricPath("/metrics"), // where metric will be publisher, default "/metrics"
    prometheusfiber.WithSkipURL(skipURL)) // urls to skip 
middleware.Use(app)
```
Running metrics in different server.
```go
// main server
app := fiber.New()

// creating a server to publish metrics.
metricApp := fiber.New()
middleware := prometheusfiber.NewPrometheus(
    prometheusfiber.WithSubSystem("fiber"),     // define subsystem, default "fiber"
    prometheusfiber.WithMetricPath("/metrics"), // where metric will be publisher, default "/metrics"
    prometheusfiber.WithSkipURL("/skip")) // urls to skip

// collect metrics on main server
app.Use(middleware.Middleware)

// publisher metrics on metric server.
middleware.SetMetricsPath(metricApp)

go func() { metricApp.Listen(":9090") }()

// run main application.
app.Listen(":8080")
```

    
