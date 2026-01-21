package rest

import (
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ipACLMiddleware creates a middleware that restricts access based on client IP addresses
func ipACLMiddleware(allowedCIDRs []string) gin.HandlerFunc {
	// Parse all CIDRs once at middleware creation
	allowedNets := make([]*net.IPNet, 0, len(allowedCIDRs))
	for _, cidr := range allowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Printf("WARNING: Failed to parse CIDR %q: %v", cidr, err)
			continue
		}
		allowedNets = append(allowedNets, ipNet)
	}

	log.Printf("IP ACL enabled with %d allowed networks", len(allowedNets))

	return func(c *gin.Context) {
		// Use remote address to avoid trusting spoofed headers by default.
		ip := remoteIP(c.Request.RemoteAddr)
		if ip == nil {
			log.Printf("IP ACL: blocked invalid IP from %s", c.Request.RemoteAddr)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		clientIP := ip.String()

		// Check if IP is in any allowed network
		allowed := false
		for _, ipNet := range allowedNets {
			if ipNet.Contains(ip) {
				allowed = true
				break
			}
		}

		if !allowed {
			log.Printf("IP ACL: blocked %s from %s %s", clientIP, c.Request.Method, c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		c.Next()
	}
}

func remoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}
