package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Logger   LoggerConfig
	HTTP     HTTPConfig
	DB       DatabaseConfig
	Telegram TelegramConfig
	Auth     AuthConfig
	Geo      GeoConfig
}

type AppConfig struct {
	Name        string
	Version     string
	Environment string
}

type LoggerConfig struct {
	Level      string
	Format     string
	Output     string
	FilePath   string
	Service    string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
	Timezone        string
}

type TelegramConfig struct {
	BotToken string
	Admins   []string
	// SiteURL is the public site root the bot links to and fetches the resume from.
	SiteURL string
	// DigestHour is the local hour (DB_TIMEZONE) of the daily traffic summary; -1 disables it.
	DigestHour int
}

type AuthConfig struct {
	AdminUsername     string
	AdminPasswordHash string
	JWTSecret         string
	AccessDuration    time.Duration
}

type GeoConfig struct {
	MMDBPath string
}

func Load() (*Config, error) {
	env := getEnv("APP_ENV", "development")
	loadDotenv(env)

	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "rami-server"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: env,
		},
		Logger: LoggerConfig{
			Level:      getEnv("LOG_LEVEL", defaultLogLevel(env)),
			Format:     getEnv("LOG_FORMAT", defaultLogFormat(env)),
			Output:     getEnv("LOG_OUTPUT", defaultLogOutput(env)),
			FilePath:   getEnv("LOG_FILE_PATH", "logs/app.log"),
			Service:    getEnv("LOG_SERVICE", "analytics-server"),
			MaxSizeMB:  getEnvInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
			MaxAgeDays: getEnvInt("LOG_MAX_AGE_DAYS", 30),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		},
		HTTP: HTTPConfig{
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
			CORSOrigins:     getEnv("HTTP_CORS_ORIGINS", "http://localhost:4200"),
		},
		DB: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "analytics"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        getEnvInt("DB_MAX_CONNS", 20),
			MinConns:        getEnvInt("DB_MIN_CONNS", 5),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
			ConnectTimeout:  getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			Timezone:        getEnv("DB_TIMEZONE", "Europe/Moscow"),
		},
		Telegram: TelegramConfig{
			BotToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
			Admins:     parseCSV(getEnv("TELEGRAM_ADMINS", "")),
			SiteURL:    strings.TrimSuffix(getEnv("TELEGRAM_SITE_URL", "https://ramisuliman.com"), "/"),
			DigestHour: getEnvInt("TELEGRAM_DIGEST_HOUR", 21),
		},
		Auth: AuthConfig{
			AdminUsername:     getEnv("AUTH_ADMIN_USERNAME", "admin"),
			AdminPasswordHash: getEnv("AUTH_ADMIN_PASSWORD_HASH", ""),
			JWTSecret:         getEnv("AUTH_JWT_SECRET", ""),
			AccessDuration:    getEnvDuration("AUTH_ACCESS_DURATION", 24*time.Hour),
		},
		Geo: GeoConfig{
			MMDBPath: getEnv("GEO_MMDB_PATH", "./data/GeoLite2-Country.mmdb"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	var errs []string

	if !isValidEnvironment(c.App.Environment) {
		errs = append(errs, fmt.Sprintf("APP_ENV invalid: %q", c.App.Environment))
	}
	if strings.TrimSpace(c.App.Name) == "" {
		errs = append(errs, "APP_NAME must not be empty")
	}
	if !isValidLogLevel(c.Logger.Level) {
		errs = append(errs, fmt.Sprintf("LOG_LEVEL invalid: %q", c.Logger.Level))
	}
	if !isValidLogFormat(c.Logger.Format) {
		errs = append(errs, fmt.Sprintf("LOG_FORMAT must be json or text, got %q", c.Logger.Format))
	}
	if !isValidLogOutput(c.Logger.Output) {
		errs = append(errs, fmt.Sprintf("LOG_OUTPUT must be stdout, file or both, got %q", c.Logger.Output))
	}
	if !isValidPort(c.HTTP.Port) {
		errs = append(errs, fmt.Sprintf("HTTP_PORT invalid: %q", c.HTTP.Port))
	}
	if !isValidSSLMode(c.DB.SSLMode) {
		errs = append(errs, fmt.Sprintf("DB_SSL_MODE invalid: %q", c.DB.SSLMode))
	}
	if !isValidLogFormat(c.Logger.Format) {
		errs = append(errs, fmt.Sprintf("LOG_FORMAT invalid: %q", c.Logger.Format))
	}
	if !isValidLogOutput(c.Logger.Output) {
		errs = append(errs, fmt.Sprintf("LOG_OUTPUT invalid: %q", c.Logger.Output))
	}
	if !isValidPort(c.HTTP.Port) {
		errs = append(errs, fmt.Sprintf("HTTP_PORT invalid: %q", c.HTTP.Port))
	}
	if c.HTTP.ReadTimeout <= 0 || c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 {
		errs = append(errs, "HTTP timeouts must be positive")
	}
	if c.HTTP.ShutdownTimeout <= 0 {
		errs = append(errs, "HTTP_SHUTDOWN_TIMEOUT must be positive")
	}
	if strings.TrimSpace(c.HTTP.CORSOrigins) == "" {
		errs = append(errs, "HTTP_CORS_ORIGINS must not be empty")
	}
	if strings.TrimSpace(c.DB.Host) == "" {
		errs = append(errs, "DB_HOST must not be empty")
	}
	if !isValidPort(c.DB.Port) {
		errs = append(errs, fmt.Sprintf("DB_PORT invalid: %q", c.DB.Port))
	}
	if strings.TrimSpace(c.DB.User) == "" {
		errs = append(errs, "DB_USER must not be empty")
	}
	if strings.TrimSpace(c.DB.Name) == "" {
		errs = append(errs, "DB_NAME must not be empty")
	}
	if !isValidSSLMode(c.DB.SSLMode) {
		errs = append(errs, fmt.Sprintf("DB_SSL_MODE invalid: %q", c.DB.SSLMode))
	}
	if c.DB.MaxConns < c.DB.MinConns {
		errs = append(errs, "DB_MAX_CONNS must be >= DB_MIN_CONNS")
	}
	if c.DB.MaxConns <= 0 {
		errs = append(errs, "DB_MAX_CONNS must be > 0")
	}

	tokenSet := strings.TrimSpace(c.Telegram.BotToken) != ""
	adminsSet := len(c.Telegram.Admins) > 0
	if tokenSet != adminsSet {
		errs = append(errs, "TELEGRAM_BOT_TOKEN and TELEGRAM_ADMINS must both be set or both be empty")
	}
	if tokenSet && !isValidTelegramToken(c.Telegram.BotToken) {
		errs = append(errs, "TELEGRAM_BOT_TOKEN format invalid")
	}
	if tokenSet && !strings.HasPrefix(c.Telegram.SiteURL, "https://") && !strings.HasPrefix(c.Telegram.SiteURL, "http://") {
		errs = append(errs, "TELEGRAM_SITE_URL must be an http(s) URL")
	}
	if c.Telegram.DigestHour < -1 || c.Telegram.DigestHour > 23 {
		errs = append(errs, "TELEGRAM_DIGEST_HOUR must be -1 (off) or 0-23")
	}
	if _, err := time.LoadLocation(c.DB.Timezone); err != nil {
		errs = append(errs, "DB_TIMEZONE is not a valid IANA timezone")
	}

	if strings.TrimSpace(c.Auth.AdminUsername) == "" {
		errs = append(errs, "AUTH_ADMIN_USERNAME must not be empty")
	}
	if strings.TrimSpace(c.Auth.AdminPasswordHash) == "" {
		errs = append(errs, "AUTH_ADMIN_PASSWORD_HASH must not be empty")
	}
	if strings.TrimSpace(c.Auth.JWTSecret) == "" {
		errs = append(errs, "AUTH_JWT_SECRET must not be empty")
	}
	if c.Auth.AccessDuration <= 0 {
		errs = append(errs, "AUTH_ACCESS_DURATION must be positive")
	}

	if c.IsProduction() {
		if len(c.Auth.JWTSecret) < 32 {
			errs = append(errs, "AUTH_JWT_SECRET must be at least 32 chars in production")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Config) IsProduction() bool { return c.App.Environment == "production" }

// DSN is a postgres:// URL, so credentials with spaces or quotes stay intact.
func (c *Config) DSN() string {
	q := url.Values{}
	q.Set("sslmode", c.DB.SSLMode)
	q.Set("TimeZone", c.DB.Timezone)
	q.Set("connect_timeout", strconv.Itoa(int(c.DB.ConnectTimeout.Seconds())))
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DB.User, c.DB.Password),
		Host:     net.JoinHostPort(c.DB.Host, c.DB.Port),
		Path:     c.DB.Name,
		RawQuery: q.Encode(),
	}
	return u.String()
}

func isValidEnvironment(env string) bool {
	return env == "development" || env == "staging" || env == "production"
}

func isValidLogLevel(l string) bool {
	return l == "debug" || l == "info" || l == "warn" || l == "error"
}

func isValidLogFormat(f string) bool { return f == "json" || f == "text" }

func isValidLogOutput(o string) bool {
	return o == "stdout" || o == "file" || o == "both"
}

func isValidSSLMode(m string) bool {
	switch m {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	}
	return false
}

func isValidPort(p string) bool {
	n, err := strconv.Atoi(p)
	return err == nil && n > 0 && n < 65536
}

func isValidTelegramToken(token string) bool {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || len(parts[0]) < 6 || len(parts[1]) != 35 {
		return false
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return false
		}
	}
	for _, r := range parts[1] {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func defaultLogLevel(env string) string {
	if env == "production" {
		return "info"
	}
	return "debug"
}

func defaultLogFormat(env string) string {
	if env == "production" {
		return "json"
	}
	return "text"
}

func defaultLogOutput(env string) string {
	if env == "production" {
		return "both"
	}
	return "stdout"
}
