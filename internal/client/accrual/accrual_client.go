package accrual

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/DenisPavlov/go-musthave-diploma/internal/config"
	"github.com/DenisPavlov/go-musthave-diploma/internal/model"
	"github.com/go-chi/render"
)

var (
	ErrOrderNotRegistered = errors.New("order not registered")
)

type Client struct {
	log        *slog.Logger
	baseURL    string
	httpClient *http.Client
}

func NewClient(log *slog.Logger, cfg *config.Config) *Client {
	return &Client{
		log:     log.With(slog.String("client", "accrual")),
		baseURL: cfg.AccrualSystemAddress,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // todo -вынести в конфиг
		},
	}
}

func (c *Client) GetOrder(ctx context.Context, oderNum string) (*model.AccrualOrder, error) {
	log := c.log.With(
		slog.String("component", "accrual.get_order"),
	)
	op := "accrual.GetOrder"

	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, oderNum)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s %w", op, err)
	}
	log.DebugContext(ctx, "request url", slog.String("url", url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s %w", op, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.ErrorContext(ctx, "failed to close response body", slog.String("url", url))
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		return parseSuccessfulResponse(resp)
	case http.StatusNoContent:
		return nil, ErrOrderNotRegistered
	case http.StatusTooManyRequests:
		return nil, handleRateLimit(resp)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("server error: %s", op)
	default:
		return nil, fmt.Errorf("unexpected status code: %s %d", op, resp.StatusCode)
	}
}

func parseSuccessfulResponse(resp *http.Response) (*model.AccrualOrder, error) {
	op := "accrual.GetOrder"
	var orderResp model.AccrualOrder
	err := render.DecodeJSON(resp.Body, &orderResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %s %w", op, err)
	}
	return &orderResp, nil
}

// todo - обработать ошибку из этой функции с учетом seconds
func handleRateLimit(resp *http.Response) error {
	op := "accrual.GetOrder"
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return fmt.Errorf("rate limit exceeded, retry after unknown: %s", op)
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return fmt.Errorf("rate limit exceeded, failed to parse Retry-After: %w", err)
	}

	return fmt.Errorf("rate limit exceeded, retry after %d seconds", seconds)
}
