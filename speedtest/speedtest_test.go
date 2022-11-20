package speedtest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("DefaultDoer", func(t *testing.T) {
		c := New()

		if c.doer == nil {
			t.Error("doer is nil by")
		}
	})

	t.Run("CustomDoer", func(t *testing.T) {
		doer := &http.Client{}

		c := New(WithDoer(doer))
		if c.doer != doer {
			t.Error("doer is not the same")
		}
	})
}

func TestUserAgent(t *testing.T) {
	testServer := func(expectedUserAgent string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.UserAgent() == "" {
				t.Error("Did not receive User-Agent header")
			} else if r.UserAgent() != expectedUserAgent {
				t.Errorf("Incorrect User-Agent header: %s, expected: %s", r.UserAgent(), expectedUserAgent)
			}
		}))
	}

	t.Run("DefaultUserAgent", func(t *testing.T) {
		c := New()
		s := testServer(defaultUserAgent)
		c.doer.Get(s.URL)
	})

	t.Run("CustomUserAgent", func(t *testing.T) {
		testAgent := "asdf1234"
		s := testServer(testAgent)
		c := New(WithUserAgent(testAgent))
		c.doer.Get(s.URL)
	})

	// Test that With
	t.Run("CustomUserAgentAndDoer", func(t *testing.T) {
		testAgent := "asdf2345"
		doer := &http.Client{}
		s := testServer(testAgent)
		c := New(WithDoer(doer), WithUserAgent(testAgent))
		if c.doer != doer {
			t.Error("doer is not the same")
		}
		c.doer.Get(s.URL)
	})
}
