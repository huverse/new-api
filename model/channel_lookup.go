package model

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func channelLookupModelCandidates(modelName string) []string {
	if modelName == "" {
		return nil
	}
	candidates := make([]string, 0, 4)
	add := func(name string) {
		if name == "" {
			return
		}
		for _, existing := range candidates {
			if existing == name {
				return
			}
		}
		candidates = append(candidates, name)
	}

	add(modelName)
	add(ratio_setting.FormatMatchingModelName(modelName))

	if strings.HasSuffix(modelName, ratio_setting.CompactModelSuffix) {
		baseModel := strings.TrimSuffix(modelName, ratio_setting.CompactModelSuffix)
		add(baseModel)
		add(ratio_setting.FormatMatchingModelName(baseModel))
	}

	return candidates
}
