package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func loadDotenv(env string) {
	if strings.ToLower(env) == "production" {
		return
	}
	files := []string{
		fmt.Sprintf(".env.%s.local", env),
		fmt.Sprintf(".env.%s", env),
		".env.local",
		".env",
	}
	for _, file := range files {
		if path := findEnvFile(file); path != "" {
			if err := godotenv.Load(path); err != nil {
				log.Printf("warning: could not load %s: %v", path, err)
			} else {
				log.Printf("info: loaded env from %s", path)
				return
			}
		}
	}
}

func findEnvFile(filename string) string {
	if filepath.IsAbs(filename) {
		if fileExists(filename) {
			return filename
		}
		return ""
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, filename)
		if fileExists(path) {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		panic(fmt.Sprintf("env %q must be boolean, got %q", key, v))
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(fmt.Sprintf("env %q must be duration, got %q", key, v))
	}
	return d
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("env %q must be int, got %q", key, v))
	}
	return n
}

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
