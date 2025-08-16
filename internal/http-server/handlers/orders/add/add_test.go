package add

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mocks "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/orders/add/mocks"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/stretchr/testify/require"
)

func TestAddHandler(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		respStatus  int
		orderNum    string
	}{
		{
			name:        "Success",
			contentType: "text/plain",
			respStatus:  http.StatusOK,
			orderNum:    "79927398713",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			login := "login"
			ctx := context.WithValue(context.Background(), auth.UsernameKey, login)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(c.orderNum)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", c.contentType)

			orderAdderMock := mocks.NewMockOrderAdder(t)

			orderAdderMock.EXPECT().AddOrder(ctx, c.orderNum, login).Once().Return(nil)

			handler := New(slog.Default(), orderAdderMock)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, rr.Code, c.respStatus)
		})
	}
}
