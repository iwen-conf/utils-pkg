package url

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignatureV2_GETAndPOST(t *testing.T) {
	built, err := NewURLBuilder("https://example.com/path", testURLSecret).
		AddParam("q", "1").
		Build()
	if err != nil {
		t.Fatalf("Build GET: %v", err)
	}
	if !strings.Contains(built, "_sigv=2") {
		t.Fatalf("expected _sigv=2 in %s", built)
	}
	ok, err := ValidateSignature(built, testURLSecret, 0)
	if err != nil || !ok {
		t.Fatalf("GET validate: ok=%v err=%v", ok, err)
	}

	post, err := NewURLBuilder("https://example.com/path", testURLSecret).
		SetMethod("post").
		AddParam("q", "1").
		Build()
	if err != nil {
		t.Fatalf("Build POST: %v", err)
	}
	ok, err = ValidateSignatureWithMethod(post, "POST", testURLSecret, 0)
	if err != nil || !ok {
		t.Fatalf("POST validate: ok=%v err=%v", ok, err)
	}
	if _, err := ValidateSignatureWithMethod(post, "GET", testURLSecret, 0); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("GET on POST URL: %v", err)
	}
}

func TestSignatureV2_FragmentNotSigned(t *testing.T) {
	built, err := NewURLBuilder("https://example.com/path", testURLSecret).
		SetFragment("one").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	parsed, err := url.Parse(built)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	parsed.Fragment = "other"
	ok, err := ValidateSignature(parsed.String(), testURLSecret, 0)
	if err != nil || !ok {
		t.Fatalf("fragment is not part of the HTTP request target: ok=%v err=%v", ok, err)
	}
}

func TestSignatureV2_TamperMethodHostPathQuery(t *testing.T) {
	built, err := NewURLBuilder("https://example.com:8443/a%20b", testURLSecret).
		AddParam("q", "1").
		AddParam("q", "2").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cases := []struct {
		name string
		mut  func(*url.URL)
	}{
		{"scheme", func(u *url.URL) { u.Scheme = "http" }},
		{"host", func(u *url.URL) { u.Host = "evil.example" }},
		{"port", func(u *url.URL) { u.Host = "example.com:9443" }},
		{"path", func(u *url.URL) { u.Path = "/other" }},
		{"query value", func(u *url.URL) {
			q := u.Query()
			q.Set("q", "x")
			u.RawQuery = q.Encode()
		}},
		{"query add", func(u *url.URL) {
			q := u.Query()
			q.Add("extra", "1")
			u.RawQuery = q.Encode()
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := url.Parse(built)
			if err != nil {
				t.Fatal(err)
			}
			tc.mut(parsed)
			if _, err := ValidateSignature(parsed.String(), testURLSecret, 0); !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("expected ErrInvalidSignature, got %v", err)
			}
		})
	}

	moved, err := url.Parse(built)
	if err != nil {
		t.Fatal(err)
	}
	moved.Path = "/moved"
	if _, err := ValidateSignature(moved.String(), testURLSecret, 0); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("moved path with same query: %v", err)
	}
}

func TestSignatureV2_InputAndTimeRules(t *testing.T) {
	if _, err := NewURLBuilder("https://example.com", strings.Repeat("a", 31)).Build(); !errors.Is(err, ErrSecretKeyTooShort) {
		t.Fatalf("31-byte secret: %v", err)
	}
	hexKey := "0123456789abcdef0123456789abcdef"
	if _, err := NewURLBuilder("https://example.com", hexKey).Build(); err != nil {
		t.Fatalf("32-byte hex secret: %v", err)
	}

	if _, err := NewURLBuilder("/relative", testURLSecret).Build(); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("relative: %v", err)
	}
	if _, err := NewURLBuilder("mailto:a@b.com", testURLSecret).Build(); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("mailto: %v", err)
	}
	if _, err := NewURLBuilder("https://user:pass@example.com", testURLSecret).Build(); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("userinfo: %v", err)
	}

	if _, err := NewURLBuilder("https://example.com", testURLSecret).SetMethod("POST\n").Build(); !errors.Is(err, ErrInvalidMethod) {
		t.Fatalf("method newline: %v", err)
	}
	if _, err := NewURLBuilder("https://example.com", testURLSecret).AddParam("_sign", "x").Build(); !errors.Is(err, ErrReservedQueryParamConflict) {
		t.Fatalf("reserved param: %v", err)
	}

	built, err := NewURLBuilder("https://example.com", testURLSecret).SetExpiration(30).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	ok, err := ValidateSignature(built, testURLSecret, 300)
	if err != nil || !ok {
		t.Fatalf("maxAge larger than exp must not widen: %v", err)
	}

	tight, err := NewURLBuilder("https://example.com", testURLSecret).SetExpiration(60).Build()
	if err != nil {
		t.Fatalf("Build tight: %v", err)
	}
	ok, err = ValidateSignature(tight, testURLSecret, 30)
	if err != nil || !ok {
		t.Fatalf("maxAge smaller than exp should still pass now: %v", err)
	}

	legacy := strings.Replace(built, "_sigv=2", "_sigv=1", 1)
	if _, err := ValidateSignature(legacy, testURLSecret, 0); !errors.Is(err, ErrUnsupportedSignatureVersion) {
		t.Fatalf("legacy v1: %v", err)
	}

	parsed, _ := url.Parse(built)
	q := parsed.Query()
	q.Add("_sign", "dup")
	parsed.RawQuery = q.Encode()
	if _, err := ValidateSignature(parsed.String(), testURLSecret, 0); !errors.Is(err, ErrAmbiguousSignatureParams) {
		t.Fatalf("duplicate reserved: %v", err)
	}

	missing := strings.Replace(built, "_sigv=2&", "", 1)
	if _, err := ValidateSignature(missing, testURLSecret, 0); !errors.Is(err, ErrMissingSignatureVersion) {
		t.Fatalf("missing sigv: %v", err)
	}

	future, err := NewURLBuilder("https://example.com", testURLSecret).
		SetTimestamp(time.Now().Unix() + allowedTimeDrift + 10).
		Build()
	if err != nil {
		t.Fatalf("future build: %v", err)
	}
	if _, err := ValidateSignature(future, testURLSecret, 0); !errors.Is(err, ErrFutureTimestamp) {
		t.Fatalf("future ts: %v", err)
	}
}

func TestCreateSignedURLAndBatch(t *testing.T) {
	signed, err := CreateSignedURL("https://example.com/api", testURLSecret, map[string]string{"id": "1"}, 60)
	if err != nil {
		t.Fatalf("CreateSignedURL: %v", err)
	}
	results := BatchValidateSignatures([]string{signed}, testURLSecret, 0)
	if results[signed] != nil {
		t.Fatalf("batch: %v", results[signed])
	}
}
