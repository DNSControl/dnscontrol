package namedotcom

import "testing"

func TestInitializeAPIURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"default", "", "https://api.name.com"},
		{"legacy hostname", "api.dev.name.com", "https://api.dev.name.com"},
		{"explicit HTTPS", "https://api.dev.name.com", "https://api.dev.name.com"},
		{"explicit HTTP", "http://localhost:8080", "http://localhost:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &namedotcomProvider{}
			err := provider.Initialize(map[string]string{
				"apiuser": "user", "apikey": "token", "apiurl": tt.url,
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if provider.client.Server != tt.want || provider.APIUrl != tt.want {
				t.Errorf("API URL = %q, client server = %q; want %q", provider.APIUrl, provider.client.Server, tt.want)
			}
		})
	}
}
