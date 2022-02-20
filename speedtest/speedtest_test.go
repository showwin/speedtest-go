package speedtest

import (
	"net/http"
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
