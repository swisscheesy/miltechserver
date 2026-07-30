package shared

import (
	"fmt"

	"miltechserver/bootstrap"
)

type Config struct {
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

func DefaultConfig() Config {
	return Config{
		MaxOwnedChecklists:     250,
		MaxActiveSubscriptions: 500,
		MaxChecklistModels:     100,
		MaxSections:            100,
		MaxSectionModels:       100,
		MaxSectionModelsTotal:  1000,
		MaxItemsPerSection:     500,
		MaxItemsTotal:          2000,
		MaxNoticesPerItem:      100,
		MaxNoticesTotal:        4000,
		MaxStepsPerItem:        250,
		MaxStepsTotal:          10000,
		MaxMutationBodyBytes:   8 * 1024 * 1024,
		MaxDeltaResponseBytes:  20 * 1024 * 1024,
		DeltaDefaultLimit:      10,
		DeltaMaxLimit:          25,
		UpdatesDefaultLimit:    50,
		UpdatesMaxLimit:        100,
		CommunityDefaultLimit:  20,
		CommunityMaxLimit:      50,
		TransactionMaxAttempts: 3,
	}
}

func ConfigFromEnv(env *bootstrap.Env) (Config, error) {
	if env == nil {
		return Config{}, fmt.Errorf("user PMCS configuration requires bootstrap environment")
	}

	config := Config(env.UserPmcs)
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) validate() error {
	values := []struct {
		name  string
		value int64
	}{
		{"MaxOwnedChecklists", int64(config.MaxOwnedChecklists)},
		{"MaxActiveSubscriptions", int64(config.MaxActiveSubscriptions)},
		{"MaxChecklistModels", int64(config.MaxChecklistModels)},
		{"MaxSections", int64(config.MaxSections)},
		{"MaxSectionModels", int64(config.MaxSectionModels)},
		{"MaxSectionModelsTotal", int64(config.MaxSectionModelsTotal)},
		{"MaxItemsPerSection", int64(config.MaxItemsPerSection)},
		{"MaxItemsTotal", int64(config.MaxItemsTotal)},
		{"MaxNoticesPerItem", int64(config.MaxNoticesPerItem)},
		{"MaxNoticesTotal", int64(config.MaxNoticesTotal)},
		{"MaxStepsPerItem", int64(config.MaxStepsPerItem)},
		{"MaxStepsTotal", int64(config.MaxStepsTotal)},
		{"MaxMutationBodyBytes", config.MaxMutationBodyBytes},
		{"MaxDeltaResponseBytes", int64(config.MaxDeltaResponseBytes)},
		{"DeltaDefaultLimit", int64(config.DeltaDefaultLimit)},
		{"DeltaMaxLimit", int64(config.DeltaMaxLimit)},
		{"UpdatesDefaultLimit", int64(config.UpdatesDefaultLimit)},
		{"UpdatesMaxLimit", int64(config.UpdatesMaxLimit)},
		{"CommunityDefaultLimit", int64(config.CommunityDefaultLimit)},
		{"CommunityMaxLimit", int64(config.CommunityMaxLimit)},
		{"TransactionMaxAttempts", int64(config.TransactionMaxAttempts)},
	}
	for _, value := range values {
		if value.value <= 0 {
			return fmt.Errorf("user PMCS configuration %s must be positive", value.name)
		}
	}
	return nil
}
