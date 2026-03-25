package logtrace

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaginatorFromRequest(t *testing.T) {
	t.Run("uses defaults when no query params are set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		p := PaginatorFromRequest(req)

		require.Equal(t, int64(1), p.Page)
		require.Equal(t, int64(defaultNumOfItemPerPage), p.PerPage)
	})

	t.Run("parses valid page and per_page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=2&per_page=15", nil)

		p := PaginatorFromRequest(req)

		require.Equal(t, int64(2), p.Page)
		require.Equal(t, int64(15), p.PerPage)
	})

	t.Run("rejects non-positive values and keeps defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=0&per_page=0", nil)

		p := PaginatorFromRequest(req)

		require.Equal(t, int64(1), p.Page)
		require.Equal(t, int64(defaultNumOfItemPerPage), p.PerPage)
	})
}
