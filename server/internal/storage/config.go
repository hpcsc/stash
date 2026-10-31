package storage

// Config selects and configures the storage backend.
type Config struct {
	Backend string `env:"STORAGE_BACKEND" envDefault:"disk"`
	Dir     string `env:"STORAGE_DIR" envDefault:"./data"`

	Bucket    string `env:"S3_BUCKET"`
	Region    string `env:"S3_REGION"`
	Endpoint  string `env:"S3_ENDPOINT"`
	AccessKey string `env:"S3_ACCESS_KEY_ID"`
	SecretKey string `env:"S3_SECRET_ACCESS_KEY"`
	PathStyle bool   `env:"S3_PATH_STYLE"`
}
