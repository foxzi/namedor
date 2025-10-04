package rest

import "testing"

func TestFQDN(t *testing.T) {
    tests := []struct{ name, zone, want string }{
        {"@", "example.com", "example.com."},
        {"", "example.com", "example.com."},
        {"www", "example.com.", "www.example.com."},
        {"WWW", "Example.Com", "www.example.com."},
    }
    for _, tt := range tests {
        if got := fqdn(tt.name, tt.zone); got != tt.want {
            t.Fatalf("fqdn(%q,%q)=%q want %q", tt.name, tt.zone, got, tt.want)
        }
    }
}

