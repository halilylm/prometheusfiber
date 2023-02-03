
## Usage/Examples

Each option is optional. Use what you need, if you don't use, it will fall back to default urls.
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

// publish metrics on metric server.
middleware.SetMetricsPath(metricApp)

// run metric app on different server.
go func() { metricApp.Listen(":9090") }()

// run main application.
app.Listen(":8080")
```