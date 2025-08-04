package add

import (
	"bytes"
	"context"
	mocks "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/orders/add/mocks"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
todo - add more tests
1. Тест на Content-Type: text/plain - 400
2. Тест на пустое тело сообщения - 422
3. Валидация номера заказа - 422
4.Тест на корректность по алгоритму Луна
*/

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

/*
func TestRequesterMockRun(t *testing.T) {
    m := NewMockRequester(t)
    m.EXPECT().Get(mock.Anything).Return("", nil)
    m.EXPECT().Get(mock.Anything).Run(func(path string) {
        fmt.Printf("Side effect! Argument is: %s", path)
    })
    retString, err := m.Get("hello")
    assert.NoError(t, err)
    assert.Equal(t, retString, "")
}

*/

/*
func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		respError string
		mockError error
	}{
		{
			name:  "Success",
			alias: "test_alias",
			url:   "https://google.com",
		},
		{
			name:  "Empty alias",
			alias: "",
			url:   "https://google.com",
		},
		{
			name:      "Empty URL",
			url:       "",
			alias:     "some_alias",
			respError: "field URL is a required field",
		},
		{
			name:      "Invalid URL",
			url:       "some invalid URL",
			alias:     "some_alias",
			respError: "field URL is not a valid URL",
		},
		{
			name:      "SaveURL Error",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "failed to add url",
			mockError: errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, rr.Code, http.StatusOK)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

			// TODO: add more checks
		})
	}
}

*/
