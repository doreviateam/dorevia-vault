package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "vault_http_requests_total",
		Help: "Total des requêtes HTTP par route/méthode/code",
	},
	[]string{"route", "method", "code"},
)

// safeRegister évite le panic si la métrique est déjà enregistrée
func safeRegister(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return
		}
		panic(err)
	}
}

func init() {
	safeRegister(httpRequestsTotal)
}

// PrometheusMiddleware comptabilise chaque requête par route/méthode/code
func PrometheusMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		code := c.Response().StatusCode()
		routePath := c.Route().Path
		if routePath == "" {
			routePath = c.Path() // Fallback si Route().Path est vide
		}
		httpRequestsTotal.WithLabelValues(routePath, c.Method(), fmt.Sprint(code)).Inc()
		return err
	}
}

