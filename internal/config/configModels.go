package config

import "time"

type Config struct {
	Env            string           `yaml:"env" env-default:"local"`
	HttpServer     HttpServerConfig `yaml:"httpServer" env-required:"true"`
	DBConfig       DBConfig         `yaml:"db" env-required:"true"`
	BotConfig      BotConfig        `yaml:"bot" env-required:"true"`
	ScraperConfig  ScraperConfig    `yaml:"scraper" env-required:"true"`
	ConfigFilePath string           `yaml:"configFilePath" env:"CONFIG_FILEPATH" env-default:""`
	ConfigFileName string           `yaml:"configFileName" env:"CONFIG_FILENAME" env-default:""`
	configPath     string
}

type HttpServerConfig struct {
	Address string        `yaml:"address" env-required:"true" env-default:"localhost"`
	Port    string        `yaml:"port" env-required:"true" env-default:"8080"`
	Timeout time.Duration `yaml:"timeout" env-default:"5"`
	Secret  string        `yaml:"secret" env-required:"true" env-default:"secret"`
}

type DBConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"DB_PORT" env-default:"5432"`
	Name     string `yaml:"name" env:"DB_NAME" env-default:"postgres"`
	User     string `yaml:"user" env:"DB_USER" env-default:"user"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"password"`
}

type AIConfig struct {
	Timeout            int     `yaml:"timeout" env:"AI_TIMEOUT" env-required:"true" env-default:"600"` //in seconds
	ModelName          string  `yaml:"modelName" env:"AI_MODEL_NAME" env-required:"true"`
	AIApiToken         string  `yaml:"aiapitoken" env:"AI_API_TOKEN" env-required:"true"`
	SystemRolePrompt   string  `yaml:"systemRolePrompt" env-default:""`
	PromptFilePath     string  `yaml:"promptFilePath" env:"PROMPT_FILEPATH" env-required:"true" env-default:""`
	PromptFileName     string  `yaml:"promptFileName" env:"PROMPT_FILENAME" env-required:"true" env-default:""`
	AiResponseFilePath string  `yaml:"aiResponseFilePath" env:"AI_RESPONSE_FILEPATH" env-required:"true" env-default:""`
	MaxTokens          int     `yaml:"maxTokens" env-default:"65000"`
	Temperature        float32 `yaml:"temperature" env-default:"0.5"`
	N                  int     `yaml:"n" env-default:"1"`
	JobBufferSize      int     `yaml:"jobBufferSize" env:"AI_BUFFER_SIZE" env-default:"10"`
	WorkersCount       int     `yaml:"workersCount" env:"AI_WORKERS_COUNT" env-default:"1"`
}

type BotConfig struct {
	Admins        []string `yaml:"admins" env-default:"KrAssor"`
	TgbotApiToken string   `yaml:"tgbot_apitoken" env:"TGBOT_APITOKEN" env-required:"true"`
	AI            AIConfig `yaml:"AI"`
}

// SiteConfig описывает сайт для скрапинга.
type SiteConfig struct {
	Name string `yaml:"name"` // Имя скрапера (например, "lococlub")
	URL  string `yaml:"url"`  // URL страницы для скрапинга
}

type ScraperConfig struct {
	JobBufferSize int          `yaml:"jobBufferSize" env:"SCRAPER_JOB_BUFFER_SIZE" env-default:"10"`
	WorkersCount  int          `yaml:"workersCount" env:"SCRAPER_WORKERS_COUNT" env-default:"3"`
	Timeout       int          `yaml:"timeout" env:"SCRAPER_TIMEOUT" env-default:"600"` //in seconds
	Sites         []SiteConfig `yaml:"sites"`                                           // Список сайтов для скрапинга
}
