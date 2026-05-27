package analytics

import (
	"testing"

	"shortpress-server/internal/model"
)

func TestApplyMetaAttributionProperties(t *testing.T) {
	props := map[string]interface{}{"order_id": "txn_1"}
	meta := ResolvedMetaClick{
		Fbc:            "fb.1.123.abc",
		Fbp:            "fb.1.123.999",
		EventSourceURL: "https://example.com/create",
	}
	snapshot := model.JSONMap{"meta_fbclid": "abc"}

	ApplyMetaAttributionProperties(props, meta, snapshot)

	if props["fbc"] != meta.Fbc {
		t.Errorf("fbc: got %v", props["fbc"])
	}
	if props["fbp"] != meta.Fbp {
		t.Errorf("fbp: got %v", props["fbp"])
	}
	if props["event_source_url"] != meta.EventSourceURL {
		t.Errorf("event_source_url: got %v", props["event_source_url"])
	}
	if props["fbclid"] != "abc" {
		t.Errorf("fbclid: got %v", props["fbclid"])
	}
}

func TestApplyMetaAttributionProperties_skipsEmpty(t *testing.T) {
	props := map[string]interface{}{"order_id": "txn_1"}
	ApplyMetaAttributionProperties(props, ResolvedMetaClick{}, nil)
	for _, key := range []string{"fbc", "fbp", "fbclid", "event_source_url"} {
		if _, ok := props[key]; ok {
			t.Errorf("unexpected key %s", key)
		}
	}
}
