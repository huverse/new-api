package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestChannelLookupModelCandidatesIncludesCompactBaseModel(t *testing.T) {
	got := channelLookupModelCandidates("gpt-5.5" + ratio_setting.CompactModelSuffix)
	want := []string{"gpt-5.5" + ratio_setting.CompactModelSuffix, "gpt-5.5"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidates = %#v, want %#v", got, want)
		}
	}
}

func TestGetRandomSatisfiedChannelFallsBackToCompactBaseModel(t *testing.T) {
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldGroup2Model2Channels := group2model2channels
	oldChannelsIDM := channelsIDM
	defer func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		group2model2channels = oldGroup2Model2Channels
		channelsIDM = oldChannelsIDM
	}()

	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"svip": {
			"gpt-5.5": {42},
		},
	}
	channelsIDM = map[int]*Channel{
		42: {
			Id:     42,
			Status: common.ChannelStatusEnabled,
			Name:   "paid",
			Group:  "vip,svip",
			Models: "gpt-5.5",
		},
	}

	channel, err := GetRandomSatisfiedChannel("svip", "gpt-5.5"+ratio_setting.CompactModelSuffix, 0)
	if err != nil {
		t.Fatalf("GetRandomSatisfiedChannel error: %v", err)
	}
	if channel == nil {
		t.Fatal("expected channel, got nil")
	}
	if channel.Id != 42 {
		t.Fatalf("channel id = %d, want 42", channel.Id)
	}
}
