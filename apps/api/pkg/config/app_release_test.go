package config

import "testing"

func TestLoadAppRelease(t *testing.T) {
	for _, tc := range []struct {
		why     string
		min     string
		latest  string
		wantErr bool
	}{
		{"defaults", "", "", false},
		{"min below latest", "0.9.12", "0.10.0", false},
		{"min above latest", "0.2.0", "0.1.9", true},
		{"prefixed", "v0.1.0", "0.1.0", true},
		{"two parts", "0.1", "0.1.0", true},
	} {
		t.Setenv("KUN_APP_MIN_VERSION", tc.min)
		t.Setenv("KUN_APP_LATEST_VERSION", tc.latest)
		cfg, err := loadAppRelease()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.why, err, tc.wantErr)
		}
		if err == nil && cfg.Downloads.Android == "" {
			t.Errorf("%s: android download must default to the download page", tc.why)
		}
	}
}
