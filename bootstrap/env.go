package bootstrap

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

type Env struct {
	Host             string
	Port             string
	Username         string
	Password         string
	DBName           string
	DBDate           string
	DBSchema         string
	BlobAccountName  string
	ServerAddress    string
	SslMode          string
	ContextTimeout   int
	MobileAppVersion string
	// Connection pool settings for parallel query workloads
	DBMaxOpenConns int
	DBMaxIdleConns int
	UserPmcs       UserPmcsConfig
}

type UserPmcsConfig struct {
	MaxOwnedChecklists     int
	MaxActiveSubscriptions int
	MaxChecklistModels     int
	MaxSections            int
	MaxSectionModels       int
	MaxSectionModelsTotal  int
	MaxItemsPerSection     int
	MaxItemsTotal          int
	MaxNoticesPerItem      int
	MaxNoticesTotal        int
	MaxStepsPerItem        int
	MaxStepsTotal          int
	MaxMutationBodyBytes   int64
	MaxDeltaResponseBytes  int
	DeltaDefaultLimit      int
	DeltaMaxLimit          int
	UpdatesDefaultLimit    int
	UpdatesMaxLimit        int
	CommunityDefaultLimit  int
	CommunityMaxLimit      int
	TransactionMaxAttempts int
}

func NewEnv() *Env {
	env := Env{}
	var err error
	if os.Getenv("DEBUG") == "true" {
		log.Println("Debug build: Loading .env file")
		err = godotenv.Load(".env")
		env.SslMode = "disable"
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		log.Println("Production build: Skipping .env file")
		env.SslMode = "require"
	}
	// Database
	env.Host = os.Getenv("DB_HOST")
	env.Port = os.Getenv("DB_PORT")
	env.Username = os.Getenv("DB_USERNAME")
	env.Password = os.Getenv("DB_PASSWORD")
	env.DBName = os.Getenv("DB_NAME")
	env.DBDate = os.Getenv("DB_DATE")
	env.DBSchema = os.Getenv("DB_SCHEMA")
	env.MobileAppVersion = os.Getenv("MOBILE_APP_VERSION")
	// Connection pool settings (defaults optimized for parallel query workloads)
	env.DBMaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", 50)
	env.DBMaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", 25)
	env.UserPmcs, err = newUserPmcsConfigFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	if err := env.UserPmcs.validate(); err != nil {
		log.Fatal(err)
	}
	// Blob Storage
	env.BlobAccountName = os.Getenv("BLOB_ACCOUNT_NAME")

	log.Printf("DB_HOST: %s", env.Host)
	log.Printf("DB_PORT: %s", env.Port)
	log.Printf("DB_USERNAME: %s", env.Username)
	log.Printf("DB_NAME: %s", env.DBName)
	log.Printf("DB_DATE: %s", env.DBDate)
	log.Printf("DB_SCHEMA: %s", env.DBSchema)
	log.Printf("SSL_MODE: %s", env.SslMode)
	log.Printf("MOBILE_APP_VERSION: %s", env.MobileAppVersion)
	return &env

}

func newUserPmcsConfigFromEnvironment() (UserPmcsConfig, error) {
	config := UserPmcsConfig{}
	values := []struct {
		key          string
		defaultValue int
		assign       func(int)
	}{
		{"USER_PMCS_MAX_OWNED_CHECKLISTS", 250, func(value int) { config.MaxOwnedChecklists = value }},
		{"USER_PMCS_MAX_ACTIVE_SUBSCRIPTIONS", 500, func(value int) { config.MaxActiveSubscriptions = value }},
		{"USER_PMCS_MAX_CHECKLIST_MODELS", 100, func(value int) { config.MaxChecklistModels = value }},
		{"USER_PMCS_MAX_SECTIONS", 100, func(value int) { config.MaxSections = value }},
		{"USER_PMCS_MAX_SECTION_MODELS_PER_SECTION", 100, func(value int) { config.MaxSectionModels = value }},
		{"USER_PMCS_MAX_SECTION_MODELS_TOTAL", 1000, func(value int) { config.MaxSectionModelsTotal = value }},
		{"USER_PMCS_MAX_ITEMS_PER_SECTION", 500, func(value int) { config.MaxItemsPerSection = value }},
		{"USER_PMCS_MAX_ITEMS_TOTAL", 2000, func(value int) { config.MaxItemsTotal = value }},
		{"USER_PMCS_MAX_NOTICES_PER_ITEM", 100, func(value int) { config.MaxNoticesPerItem = value }},
		{"USER_PMCS_MAX_NOTICES_TOTAL", 4000, func(value int) { config.MaxNoticesTotal = value }},
		{"USER_PMCS_MAX_STEPS_PER_ITEM", 250, func(value int) { config.MaxStepsPerItem = value }},
		{"USER_PMCS_MAX_STEPS_TOTAL", 10000, func(value int) { config.MaxStepsTotal = value }},
		{"USER_PMCS_MAX_MUTATION_BODY_BYTES", 8 * 1024 * 1024, func(value int) { config.MaxMutationBodyBytes = int64(value) }},
		{"USER_PMCS_MAX_DELTA_RESPONSE_BYTES", 20 * 1024 * 1024, func(value int) { config.MaxDeltaResponseBytes = value }},
		{"USER_PMCS_DELTA_DEFAULT_LIMIT", 10, func(value int) { config.DeltaDefaultLimit = value }},
		{"USER_PMCS_DELTA_MAX_LIMIT", 25, func(value int) { config.DeltaMaxLimit = value }},
		{"USER_PMCS_UPDATES_DEFAULT_LIMIT", 50, func(value int) { config.UpdatesDefaultLimit = value }},
		{"USER_PMCS_UPDATES_MAX_LIMIT", 100, func(value int) { config.UpdatesMaxLimit = value }},
		{"USER_PMCS_COMMUNITY_DEFAULT_LIMIT", 20, func(value int) { config.CommunityDefaultLimit = value }},
		{"USER_PMCS_COMMUNITY_MAX_LIMIT", 50, func(value int) { config.CommunityMaxLimit = value }},
		{"USER_PMCS_TRANSACTION_MAX_ATTEMPTS", 3, func(value int) { config.TransactionMaxAttempts = value }},
	}

	for _, value := range values {
		parsed, err := getUserPmcsEnvAsInt(value.key, value.defaultValue)
		if err != nil {
			return UserPmcsConfig{}, err
		}
		value.assign(parsed)
	}
	return config, nil
}

func getUserPmcsEnvAsInt(key string, defaultValue int) (int, error) {
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func (config UserPmcsConfig) validate() error {
	values := []struct {
		name  string
		value int64
	}{
		{"USER_PMCS_MAX_OWNED_CHECKLISTS", int64(config.MaxOwnedChecklists)},
		{"USER_PMCS_MAX_ACTIVE_SUBSCRIPTIONS", int64(config.MaxActiveSubscriptions)},
		{"USER_PMCS_MAX_CHECKLIST_MODELS", int64(config.MaxChecklistModels)},
		{"USER_PMCS_MAX_SECTIONS", int64(config.MaxSections)},
		{"USER_PMCS_MAX_SECTION_MODELS_PER_SECTION", int64(config.MaxSectionModels)},
		{"USER_PMCS_MAX_SECTION_MODELS_TOTAL", int64(config.MaxSectionModelsTotal)},
		{"USER_PMCS_MAX_ITEMS_PER_SECTION", int64(config.MaxItemsPerSection)},
		{"USER_PMCS_MAX_ITEMS_TOTAL", int64(config.MaxItemsTotal)},
		{"USER_PMCS_MAX_NOTICES_PER_ITEM", int64(config.MaxNoticesPerItem)},
		{"USER_PMCS_MAX_NOTICES_TOTAL", int64(config.MaxNoticesTotal)},
		{"USER_PMCS_MAX_STEPS_PER_ITEM", int64(config.MaxStepsPerItem)},
		{"USER_PMCS_MAX_STEPS_TOTAL", int64(config.MaxStepsTotal)},
		{"USER_PMCS_MAX_MUTATION_BODY_BYTES", config.MaxMutationBodyBytes},
		{"USER_PMCS_MAX_DELTA_RESPONSE_BYTES", int64(config.MaxDeltaResponseBytes)},
		{"USER_PMCS_DELTA_DEFAULT_LIMIT", int64(config.DeltaDefaultLimit)},
		{"USER_PMCS_DELTA_MAX_LIMIT", int64(config.DeltaMaxLimit)},
		{"USER_PMCS_UPDATES_DEFAULT_LIMIT", int64(config.UpdatesDefaultLimit)},
		{"USER_PMCS_UPDATES_MAX_LIMIT", int64(config.UpdatesMaxLimit)},
		{"USER_PMCS_COMMUNITY_DEFAULT_LIMIT", int64(config.CommunityDefaultLimit)},
		{"USER_PMCS_COMMUNITY_MAX_LIMIT", int64(config.CommunityMaxLimit)},
		{"USER_PMCS_TRANSACTION_MAX_ATTEMPTS", int64(config.TransactionMaxAttempts)},
	}
	for _, value := range values {
		if value.value <= 0 {
			return fmt.Errorf("%s must be positive", value.name)
		}
	}
	return nil
}

// Ensure env file isn't loaded in production -- make sure dockerfile uses production
