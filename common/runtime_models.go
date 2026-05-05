package common

import (
	"sort"
	"strings"
	"sync"
)

const (
	EndpointOnlyImageModelsEnv     = "ENDPOINT_ONLY_IMAGE_MODELS"
	defaultEndpointOnlyImageModels = "gpt-image-2"
)

var (
	endpointOnlyImageModelsMu sync.RWMutex
	endpointOnlyImageModels   = parseRuntimeModelCSV(defaultEndpointOnlyImageModels)
)

func InitRuntimeModelsFromEnv() {
	SetEndpointOnlyImageModelsFromCSV(GetEnvOrDefaultString(EndpointOnlyImageModelsEnv, defaultEndpointOnlyImageModels))
}

func SetEndpointOnlyImageModelsFromCSV(modelCSV string) {
	endpointOnlyImageModelsMu.Lock()
	defer endpointOnlyImageModelsMu.Unlock()

	endpointOnlyImageModels = parseRuntimeModelCSV(modelCSV)
}

func SetEndpointOnlyImageModelsForTest(modelCSV string) func() {
	endpointOnlyImageModelsMu.Lock()
	previous := cloneRuntimeModelSet(endpointOnlyImageModels)
	endpointOnlyImageModels = parseRuntimeModelCSV(modelCSV)
	endpointOnlyImageModelsMu.Unlock()

	return func() {
		endpointOnlyImageModelsMu.Lock()
		endpointOnlyImageModels = previous
		endpointOnlyImageModelsMu.Unlock()
	}
}

func EndpointOnlyImageModels() []string {
	endpointOnlyImageModelsMu.RLock()
	defer endpointOnlyImageModelsMu.RUnlock()

	models := make([]string, 0, len(endpointOnlyImageModels))
	for _, modelName := range endpointOnlyImageModels {
		models = append(models, modelName)
	}
	sort.Strings(models)
	return models
}

// IsEndpointOnlyModel reports models that are routable only through
// endpoint-specific APIs and should not appear in the general /v1/models list.
func IsEndpointOnlyModel(modelName string) bool {
	return IsEndpointOnlyImageModel(modelName)
}

func IsEndpointOnlyImageModel(modelName string) bool {
	normalized := normalizeRuntimeModelName(modelName)
	if normalized == "" {
		return false
	}

	endpointOnlyImageModelsMu.RLock()
	defer endpointOnlyImageModelsMu.RUnlock()
	_, ok := endpointOnlyImageModels[normalized]
	return ok
}

func IsAllowedEndpointOnlyModelPath(modelName, path string) bool {
	if !IsEndpointOnlyImageModel(modelName) {
		return true
	}
	switch normalizeRuntimePath(path) {
	case "/v1/images/generations", "/v1/images/edits", "/v1/edits":
		return true
	default:
		return false
	}
}

func parseRuntimeModelCSV(modelCSV string) map[string]string {
	models := map[string]string{}
	for _, item := range strings.FieldsFunc(modelCSV, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	}) {
		modelName := strings.TrimSpace(item)
		if modelName == "" {
			continue
		}
		models[normalizeRuntimeModelName(modelName)] = modelName
	}
	return models
}

func cloneRuntimeModelSet(models map[string]string) map[string]string {
	cloned := make(map[string]string, len(models))
	for key, value := range models {
		cloned[key] = value
	}
	return cloned
}

func normalizeRuntimeModelName(modelName string) string {
	return strings.ToLower(strings.TrimSpace(modelName))
}

func normalizeRuntimePath(path string) string {
	normalized := strings.ToLower(strings.TrimSpace(path))
	if len(normalized) > 1 {
		normalized = strings.TrimRight(normalized, "/")
	}
	return normalized
}
