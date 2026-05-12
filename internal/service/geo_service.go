package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"

	"github.com/ramisoul84/rami-server/pkg/logger"
)

type GeoService interface {
	// Lookup returns the ISO country code and (with a City database) the city name.
	Lookup(ip string) (country, city string)
	Close() error
}

type geoService struct {
	db     *geoip2.Reader
	hasCty bool // GeoLite2-City carries cities; GeoLite2-Country only countries
	log    *logger.Logger
}

func NewGeoService(dbPath string, log *logger.Logger) (GeoService, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("geo: open mmdb: %w", err)
	}
	return &geoService{
		db:     db,
		hasCty: strings.Contains(db.Metadata().DatabaseType, "City"),
		log:    log,
	}, nil
}

func (s *geoService) Lookup(ip string) (string, string) {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified() {
		return "", ""
	}

	if s.hasCty {
		record, err := s.db.City(parsed)
		if err != nil {
			s.log.Warn("geo lookup failed", "ip", ip, "err", err)
			return "", ""
		}
		return record.Country.IsoCode, record.City.Names["en"]
	}

	record, err := s.db.Country(parsed)
	if err != nil {
		s.log.Warn("geo lookup failed", "ip", ip, "err", err)
		return "", ""
	}
	return record.Country.IsoCode, ""
}

func (s *geoService) Close() error { return s.db.Close() }
