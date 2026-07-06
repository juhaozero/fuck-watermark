package urlutil

import "testing"

func TestValidateParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "empty", url: "", wantErr: true},
		{name: "valid https", url: "https://v.douyin.com/abc/", wantErr: false},
		{name: "invalid scheme", url: "ftp://example.com", wantErr: true},
		{name: "localhost", url: "http://localhost/video", wantErr: true},
		{name: "private ip", url: "http://192.168.1.1/", wantErr: true},
		{name: "metadata ip", url: "http://169.254.169.254/latest/meta-data", wantErr: true},
		{name: "bilibili", url: "https://www.bilibili.com/video/BV1xx", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateParseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
