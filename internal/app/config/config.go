package config

import (
	"flag"
	"fmt"
	"sync"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"
)

// Configuration содержит все настройки приложения, сгруппированные по логическому назначению
type Configuration struct {
	// Настройки сервера
	Server ServerConfig

	// Настройки базы данных
	Database DatabaseConfig

	// Настройки внешних сервисов
	External ExternalConfig

	// Настройки безопасности
	Security SecurityConfig
}

// ServerConfig содержит настройки HTTP-сервера
type ServerConfig struct {
	// Addr - адрес и порт для запуска сервиса
	// Формат: "host:port" или ":port" для всех интерфейсов
	// По умолчанию: ":8080"
	Addr string `env:"RUN_ADDRESS" envDefault:":8080"`
}

// DatabaseConfig содержит настройки подключения к базе данных
type DatabaseConfig struct {
	// DSN - строка подключения к базе данных PostgreSQL
	// Формат: "postgres://user:password@host:port/dbname?sslmode=disable"
	// Обязательное поле
	DSN string `env:"DATABASE_URI"`
}

// ExternalConfig содержит настройки внешних сервисов
type ExternalConfig struct {
	// AccrualURL - адрес системы расчёта начислений
	// Формат: "http://host:port" или "https://host:port"
	// Обязательное поле
	AccrualURL string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

// SecurityConfig содержит настройки безопасности
type SecurityConfig struct {
	// TokenSecret - секретный ключ для подписи JWT токенов
	// Рекомендуется изменить в продакшене
	// По умолчанию: "The Little Man Who Wasn't There"
	TokenSecret string `env:"TOKEN_SIGN_KEY" envDefault:"The Little Man Who Wasn't There"`
}

var (
	instance *Configuration
	once     sync.Once
)

// Load инициализирует и возвращает конфигурацию (синглтон)
func Load() *Configuration {
	once.Do(func() {
		instance = &Configuration{}

		// Парсим переменные окружения
		if err := env.Parse(&instance.Server); err != nil {
			zap.L().Fatal("Ошибка парсинга настроек сервера", zap.Error(err))
		}

		if err := env.Parse(&instance.Database); err != nil {
			zap.L().Fatal("Ошибка парсинга настроек БД", zap.Error(err))
		}

		if err := env.Parse(&instance.External); err != nil {
			zap.L().Fatal("Ошибка парсинга настроек внешних сервисов", zap.Error(err))
		}

		if err := env.Parse(&instance.Security); err != nil {
			zap.L().Fatal("Ошибка парсинга настроек безопасности", zap.Error(err))
		}

		// Парсим флаги командной строки (переопределяют env переменные)
		parseFlags(instance)

		// Валидируем конфигурацию
		if err := instance.Validate(); err != nil {
			zap.L().Fatal("Неверная конфигурация", zap.Error(err))
		}

		// Логируем конфигурацию (без чувствительных данных)
		instance.LogConfig()
	})

	return instance
}

// parseFlags парсит флаги командной строки
func parseFlags(cfg *Configuration) {
	flag.StringVar(&cfg.Server.Addr, "a", cfg.Server.Addr,
		"адрес и порт запуска сервиса")

	flag.StringVar(&cfg.Database.DSN, "d", cfg.Database.DSN,
		"строка подключения к базе данных")

	flag.StringVar(&cfg.External.AccrualURL, "r", cfg.External.AccrualURL,
		"адрес системы расчёта начислений")

	flag.Parse()
}

// Validate проверяет корректность конфигурации
func (c *Configuration) Validate() error {
	// Проверка обязательных полей
	if c.Database.DSN == "" {
		return fmt.Errorf("DATABASE_URI обязателен")
	}

	if c.External.AccrualURL == "" {
		return fmt.Errorf("ACCRUAL_SYSTEM_ADDRESS обязателен")
	}

	// Проверка безопасности
	if c.Security.TokenSecret == "The Little Man Who Wasn't There" {
		zap.L().Warn("Используется токен по умолчанию, смените его в продакшене!")
	}

	return nil
}

// LogConfig логирует конфигурацию (без чувствительных данных)
func (c *Configuration) LogConfig() {
	zap.L().Info("Конфигурация загружена",
		zap.String("server.addr", c.Server.Addr),
		zap.Bool("database.configured", c.Database.DSN != ""),
		zap.String("external.accrual_url", c.External.AccrualURL),
		zap.Bool("security.token_configured", c.Security.TokenSecret != ""),
	)
}

// ==========================================
// Методы для удобного доступа (для обратной совместимости)
// ==========================================

// RunAddress возвращает адрес сервера (для обратной совместимости)
func (c *Configuration) RunAddress() string {
	return c.Server.Addr
}

// DatabaseURI возвращает DSN базы данных (для обратной совместимости)
func (c *Configuration) DatabaseURI() string {
	return c.Database.DSN
}

// AccrualSystemAddress возвращает URL системы начислений (для обратной совместимости)
func (c *Configuration) AccrualSystemAddress() string {
	return c.External.AccrualURL
}

// SecretToken возвращает секретный токен (для обратной совместимости)
func (c *Configuration) SecretToken() string {
	return c.Security.TokenSecret
}
