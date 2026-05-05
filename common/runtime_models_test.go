package common

import (
	"reflect"
	"testing"
)

func TestEndpointOnlyImageModelsAreConfigurable(t *testing.T) {
	defer SetEndpointOnlyImageModelsForTest("gpt-image-2, future-image-model")()

	if !IsEndpointOnlyModel("GPT-IMAGE-2") {
		t.Fatal("expected endpoint-only model match to be case-insensitive")
	}
	if !IsEndpointOnlyModel("future-image-model") {
		t.Fatal("expected configured future image model to be endpoint-only")
	}
	if IsEndpointOnlyModel("gpt-5.5") {
		t.Fatal("did not expect chat model to be endpoint-only")
	}
}

func TestEndpointOnlyImageModelsAllowOnlyImagePaths(t *testing.T) {
	defer SetEndpointOnlyImageModelsForTest("future-image-model")()

	allowedPaths := []string{
		"/v1/images/generations",
		"/v1/images/generations/",
		"/v1/images/edits",
		"/v1/edits",
	}
	for _, path := range allowedPaths {
		if !IsAllowedEndpointOnlyModelPath("future-image-model", path) {
			t.Fatalf("expected endpoint-only image model to be allowed on %s", path)
		}
	}

	blockedPaths := []string{
		"/v1/chat/completions",
		"/v1/responses",
		"/v1/completions",
	}
	for _, path := range blockedPaths {
		if IsAllowedEndpointOnlyModelPath("future-image-model", path) {
			t.Fatalf("expected endpoint-only image model to be blocked on %s", path)
		}
	}

	if !IsAllowedEndpointOnlyModelPath("gpt-5.5", "/v1/chat/completions") {
		t.Fatal("non endpoint-only models should not be blocked by this guard")
	}
}

func TestEndpointOnlyImageModelsReturnsSortedConfiguredNames(t *testing.T) {
	defer SetEndpointOnlyImageModelsForTest("z-image,a-image")()

	got := EndpointOnlyImageModels()
	want := []string{"a-image", "z-image"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EndpointOnlyImageModels() = %#v, want %#v", got, want)
	}
}
