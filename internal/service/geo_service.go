package service

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"

	"github.com/ramisoul84/rami-server/pkg/logger"
)

type GeoService interface {
	Lookup(ip string) (country, city string)
}

type geoService struct {
	db  *geoip2.Reader
	log *logger.Logger
}

func NewGeoService(dbPath string, log *logger.Logger) (GeoService, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("geo: open mmdb: %w", err)
	}
	return &geoService{db: db, log: log}, nil
}

func (s *geoService) Lookup(ip string) (string, string) {
	if ip == "" {
		return "", ""
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", ""
	}

	if parsed.IsLoopback() || parsed.IsPrivate() {
		return "Local", ""
	}

	record, err := s.db.City(parsed)
	if err != nil {
		s.log.Warn("geo lookup failed", "ip", ip, "err", err)
		return "", ""
	}

	country := record.Country.IsoCode
	if country == "" && len(record.Country.Names) > 0 {
		country = record.Country.Names["en"]
	}

	city := ""
	if len(record.City.Names) > 0 {
		city = record.City.Names["en"]
	}

	return country, city
}

func (s *geoService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
