package config

import (
	"testing"
)

func TestBaseSettings(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		env  string
		want AppSettings
	}{
		{
			name: "no env var set",
			env: "",
			want: BaseSettings(),
		},
		{
			name: "dev env",
			env: "dev",
			want: DevelopmentSettings(),
		},
		{
			name: "test env",
			env: "test",
			want: TestSettings(),
		},
		{
			name: "prod env",
			env: "prod",
			want: ProductionSettings(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSettings(tt.env)
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("BaseSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}
