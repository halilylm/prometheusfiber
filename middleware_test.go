package prometheusfiber_test

import (
	"github.com/halilylm/prometheusfiber"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestMiddleware(t *testing.T) {
	middlewarePath := "/metrics"
	skipURL := "/skip"
	app := fiber.New()
	middleware := prometheusfiber.NewPrometheus(
		prometheusfiber.WithSubSystem("random"),
		prometheusfiber.WithMetricPath(middlewarePath),
		prometheusfiber.WithSkipURL(skipURL))
	middleware.Use(app)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Get(skipURL, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	app.Get("/status/:code", func(c *fiber.Ctx) error {
		code, _ := strconv.Atoi(c.Params("code"))
		if code < 200 || code > 600 {
			t.Fatalf("%d is not valid http status code", code)
		}
		return c.SendStatus(code)
	})
	t.Run("successfully add metric", func(t *testing.T) {
		req := makeHttpGetRequest("/")
		res, _ := app.Test(req)
		assertStatusCode(t, http.StatusOK, res.StatusCode)
		req = makeHttpGetRequest("/status/500")
		res, _ = app.Test(req)
		assertStatusCode(t, http.StatusInternalServerError, res.StatusCode)
		req = makeHttpGetRequest("/status/404")
		res, _ = app.Test(req)
		assertStatusCode(t, http.StatusNotFound, res.StatusCode)
		req = makeHttpGetRequest(middlewarePath)
		res, _ = app.Test(req, -1)
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		assertContainsString(t, string(body), `random_requests_total{code="200",host="example.com",method="GET",url="/"} 1`)
		assertContainsString(t, string(body), `random_requests_total{code="404",host="example.com",method="GET",url="/status/:code"} 1`)
		assertContainsString(t, string(body), `random_requests_total{code="500",host="example.com",method="GET",url="/status/:code"} 1`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="500",method="GET",url="/status/:code"} 1`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="404",method="GET",url="/status/:code"} 1`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="200",method="GET",url="/"} 1`)
		assertContainsString(t, string(body), `random_request_duration_seconds_count{code="500",method="GET",url="/status/:code"} 1`)
	})
	t.Run("making a request to skips urls doesn't change metrics", func(t *testing.T) {
		req := makeHttpGetRequest(skipURL)
		res, _ := app.Test(req)
		assertStatusCode(t, http.StatusOK, res.StatusCode)
		req = makeHttpGetRequest("/")
		res, _ = app.Test(req)
		assertStatusCode(t, http.StatusOK, res.StatusCode)
		req = makeHttpGetRequest("/status/500")
		res, _ = app.Test(req)
		assertStatusCode(t, http.StatusInternalServerError, res.StatusCode)
		req = makeHttpGetRequest("/status/404")
		res, _ = app.Test(req)
		assertStatusCode(t, http.StatusNotFound, res.StatusCode)
		req = makeHttpGetRequest(middlewarePath)
		res, _ = app.Test(req, -1)
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		assertContainsString(t, string(body), `random_requests_total{code="200",host="example.com",method="GET",url="/"} 2`)
		assertContainsString(t, string(body), `random_requests_total{code="404",host="example.com",method="GET",url="/status/:code"} 2`)
		assertContainsString(t, string(body), `random_requests_total{code="500",host="example.com",method="GET",url="/status/:code"} 2`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="500",method="GET",url="/status/:code"} 2`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="404",method="GET",url="/status/:code"} 2`)
		assertContainsString(t, string(body), `random_request_size_bytes_count{code="200",method="GET",url="/"} 2`)
		assertContainsString(t, string(body), `random_request_duration_seconds_count{code="500",method="GET",url="/status/:code"} 2`)
	})
}

func assertStatusCode(t testing.TB, expected, got int) {
	t.Helper()
	if expected != got {
		t.Errorf("did not get correct status, expected %d, got %d", expected, got)
	}
}

func assertContainsString(t testing.TB, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("did not find the %s in %s", needle, haystack)
	}
}

func makeHttpGetRequest(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}
