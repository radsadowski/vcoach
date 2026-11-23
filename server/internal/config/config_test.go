package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		env     map[string]string
		want    Config
		wantErr bool
	}{
		"minimal config applies defaults": {
			env: map[string]string{"DATABASE_URL": "postgres://localhost/vcoach"},
			want: Config{
				DatabaseURL:  "postgres://localhost/vcoach",
				Port:         "8080",
				SyncInterval: 60 * time.Minute,
			},
		},
		"full config is passed through": {
			env: map[string]string{
				"DATABASE_URL":      "postgres://db/vcoach",
				"PORT":              "9090",
				"LLM_BASE_URL":      "https://openrouter.ai/api/v1",
				"LLM_API_KEY":       "secret",
				"LLM_MODEL":         "some/model",
				"GARMIN_API_KEY":    "garmin-secret",
				"SYNC_INTERVAL_MIN": "15",
			},
			want: Config{
				DatabaseURL:  "postgres://db/vcoach",
				Port:         "9090",
				LLMBaseURL:   "https://openrouter.ai/api/v1",
				LLMAPIKey:    "secret",
				LLMModel:     "some/model",
				GarminAPIKey: "garmin-secret",
				SyncInterval: 15 * time.Minute,
			},
		},
		"missing DATABASE_URL fails": {
			env:     map[string]string{},
			wantErr: true,
		},
		"zero sync interval fails": {
			env:     map[string]string{"DATABASE_URL": "postgres://localhost/vcoach", "SYNC_INTERVAL_MIN": "0"},
			wantErr: true,
		},
		"non-numeric sync interval fails": {
			env:     map[string]string{"DATABASE_URL": "postgres://localhost/vcoach", "SYNC_INTERVAL_MIN": "often"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			// given
			env := func(key string) string { return tt.env[key] }

			// when
			got, err := load(env)

			// then
			if tt.wantErr {
				r.Error(err)
				return
			}
			r.NoError(err)
			r.Equal(tt.want, got)
		})
	}
}

func TestLoadReadsProcessEnv(t *testing.T) {
	// Cannot use t.Parallel(): t.Setenv mutates process environment.
	r := require.New(t)

	// given
	t.Setenv("DATABASE_URL", "postgres://localhost/vcoach")
	t.Setenv("SYNC_INTERVAL_MIN", "30")

	// when
	got, err := Load()

	// then
	r.NoError(err)
	r.Equal("postgres://localhost/vcoach", got.DatabaseURL)
	r.Equal(30*time.Minute, got.SyncInterval)
}
