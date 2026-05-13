package middleware

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name, peer, realIP, forwarded, want string
	}{
		{"direct request ignores spoofed headers", "203.0.113.7", "", "1.1.1.1", "203.0.113.7"},
		{"behind proxy: X-Real-IP", "172.18.0.5", "198.51.100.4", "", "198.51.100.4"},
		{"behind proxy: right-most public hop wins", "172.18.0.5", "", "6.6.6.6, 198.51.100.4", "198.51.100.4"},
		{"behind proxy: internal hops skipped", "172.18.0.5", "", "198.51.100.4, 10.0.0.2", "198.51.100.4"},
		{"behind proxy: no headers", "172.18.0.5", "", "", "172.18.0.5"},
	}
	app := fiber.New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req fasthttp.Request
			if tc.realIP != "" {
				req.Header.Set("X-Real-IP", tc.realIP)
			}
			if tc.forwarded != "" {
				req.Header.Set(fiber.HeaderXForwardedFor, tc.forwarded)
			}
			fctx := &fasthttp.RequestCtx{}
			fctx.Init(&req, &net.TCPAddr{IP: net.ParseIP(tc.peer), Port: 1234}, nil)
			c := app.AcquireCtx(fctx)
			defer app.ReleaseCtx(c)

			if got := ClientIP(c); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
