package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ClientIP returns the visitor's address. Proxy headers are trusted only when
// the direct peer is on a private network (our reverse proxy); a request that
// reaches the server directly cannot spoof its IP with X-Forwarded-For.
func ClientIP(c *fiber.Ctx) string {
	peer := c.Context().RemoteIP()
	if peer == nil || !isInternal(peer) {
		return c.IP()
	}

	if ip := net.ParseIP(strings.TrimSpace(c.Get("X-Real-IP"))); ip != nil && !isInternal(ip) {
		return ip.String()
	}

	// X-Forwarded-For is "client, proxy1, proxy2": the right-most public address
	// is the one our own proxy saw, so earlier (client-supplied) entries can't win.
	hops := strings.Split(c.Get(fiber.HeaderXForwardedFor), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		if ip := net.ParseIP(strings.TrimSpace(hops[i])); ip != nil && !isInternal(ip) {
			return ip.String()
		}
	}
	return peer.String()
}

func isInternal(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
