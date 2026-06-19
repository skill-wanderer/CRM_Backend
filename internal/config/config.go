package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	GinMode            string
	CORSAllowedOrigins []string
	Database           DatabaseConfig
	Keycloak           KeycloakConfig
	Tenancy            TenancyConfig
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	TimeZone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	AutoMigrate     bool
}

type KeycloakConfig struct {
	BaseURL      string
	Admin        RealmConfig
	Client       RealmConfig
	AdminRole    string
	JWKSCacheTTL time.Duration
	TokenSkew    time.Duration
}

type RealmConfig struct {
	Name     string
	Issuer   string
	Audience string
}

type TenancyConfig struct {
	Header string
}

func Load() (*Config, error) {
	dbAutoMigrate, err := envBool("DB_AUTO_MIGRATE", true)
	if err != nil {
		return nil, err
	}

	maxOpenConns, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return nil, err
	}

	maxIdleConns, err := envInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return nil, err
	}

	connMaxLifetime, err := envDuration("DB_CONN_MAX_LIFETIME", time.Hour)
	if err != nil {
		return nil, err
	}

	jwksCacheTTL, err := envDuration("JWKS_CACHE_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	tokenSkew, err := envDuration("TOKEN_CLOCK_SKEW", time.Minute)
	if err != nil {
		return nil, err
	}

	baseURL, err := requiredEnv("KEYCLOAK_BASE_URL")
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimRight(baseURL, "/")

	adminRealm, err := requiredEnv("KEYCLOAK_ADMIN_REALM")
	if err != nil {
		return nil, err
	}

	clientRealm, err := requiredEnv("KEYCLOAK_CLIENT_REALM")
	if err != nil {
		return nil, err
	}

	adminIssuer := envString("KEYCLOAK_ADMIN_ISSUER", issuerURL(baseURL, adminRealm))
	clientIssuer := envString("KEYCLOAK_CLIENT_ISSUER", issuerURL(baseURL, clientRealm))

	dbUser, err := requiredEnv("DB_USER")
	if err != nil {
		return nil, err
	}
	dbPassword, err := requiredEnv("DB_PASSWORD")
	if err != nil {
		return nil, err
	}
	dbName, err := requiredEnv("DB_NAME")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:               envString("PORT", "8080"),
		GinMode:            envString("GIN_MODE", "debug"),
		CORSAllowedOrigins: envList("CORS_ALLOWED_ORIGINS", []string{"*"}),
		Database: DatabaseConfig{
			Host:            envString("DB_HOST", "localhost"),
			Port:            envString("DB_PORT", "5432"),
			User:            dbUser,
			Password:        dbPassword,
			Name:            dbName,
			SSLMode:         envString("DB_SSLMODE", "disable"),
			TimeZone:        envString("DB_TIMEZONE", "UTC"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
			AutoMigrate:     dbAutoMigrate,
		},
		Keycloak: KeycloakConfig{
			BaseURL: baseURL,
			Admin: RealmConfig{
				Name:     adminRealm,
				Issuer:   adminIssuer,
				Audience: strings.TrimSpace(os.Getenv("KEYCLOAK_ADMIN_AUDIENCE")),
			},
			Client: RealmConfig{
				Name:     clientRealm,
				Issuer:   clientIssuer,
				Audience: strings.TrimSpace(os.Getenv("KEYCLOAK_CLIENT_AUDIENCE")),
			},
			AdminRole:    envString("KEYCLOAK_ADMIN_REQUIRED_ROLE", "CRM"),
			JWKSCacheTTL: jwksCacheTTL,
			TokenSkew:    tokenSkew,
		},
		Tenancy: TenancyConfig{
			Header: envString("TENANT_HEADER", "X-Tenant-ID"),
		},
	}

	return cfg, nil
}

func issuerURL(baseURL, realm string) string {
	return fmt.Sprintf("%s/realms/%s", strings.TrimRight(baseURL, "/"), strings.Trim(realm, "/"))
}

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable %s", key)
	}
	return value, nil
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envList(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func envBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}
