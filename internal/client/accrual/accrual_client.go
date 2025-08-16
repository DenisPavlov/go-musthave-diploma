package accrual

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/DenisPavlov/go-musthave-diploma/internal/config"
	"github.com/DenisPavlov/go-musthave-diploma/internal/model"
	"github.com/go-chi/render"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	ErrOrderNotRegistered = errors.New("order not registered")
	ErrRateLimit          = errors.New("rate limit exceeded")
)

type Client struct {
	log        *slog.Logger
	baseURL    string
	httpClient *http.Client
	pauseCh    chan struct{}
	resumeCh   chan struct{}
	isPaused   bool
	mu         sync.Mutex
}

func NewClient(log *slog.Logger, cfg *config.Config) *Client {
	logger := log.With(slog.String("client", "accrual"))
	return &Client{
		log:        logger,
		baseURL:    cfg.AccrualSystemAddress,
		httpClient: configureHTTPClient(cfg, logger),
		pauseCh:    make(chan struct{}),
		resumeCh:   make(chan struct{}),
	}
}

func (c *Client) Shutdown() {
	close(c.pauseCh)
	close(c.resumeCh)
	c.httpClient.CloseIdleConnections()
}

func (c *Client) GetOrder(ctx context.Context, oderNum string) (*model.AccrualOrder, error) {
	c.waitIfPaused(ctx)

	log := c.log.With(
		slog.String("component", "accrual.get_order"),
	)
	op := "accrual.GetOrder"

	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, oderNum)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s %w", op, err)
	}

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
		return nil, c.handleRateLimit(ctx, resp)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("server error: %s", op)
	default:
		return nil, fmt.Errorf("unexpected status code: %s %d", op, resp.StatusCode)
	}
}

func parseSuccessfulResponse(resp *http.Response) (*model.AccrualOrder, error) {
	op := "accrual.parseSuccessfulResponse"
	var orderResp model.AccrualOrder
	err := render.DecodeJSON(resp.Body, &orderResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %s %w", op, err)
	}
	return &orderResp, nil
}

func (c *Client) handleRateLimit(ctx context.Context, resp *http.Response) error {
	op := "accrual.handleRateLimit"
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return fmt.Errorf("rate limit exceeded, retry after unknown: %s", op)
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return fmt.Errorf("rate limit exceeded, failed to parse Retry-After: %w", err)
	}
	c.pause(ctx, time.Duration(seconds)*time.Second)
	return ErrRateLimit
}

func configureHTTPClient(cfg *config.Config, log *slog.Logger) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.HTTPClient.RetryMax
	retryClient.HTTPClient.Timeout = cfg.HTTPClient.Timeout
	retryClient.Logger = log
	retryClient.CheckRetry = customRetryPolicy

	return retryClient.StandardClient()
}

func customRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return true, nil
	}

	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

func (c *Client) pause(ctx context.Context, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isPaused {
		return // Уже на паузе
	}

	c.isPaused = true

	// Отправляем сигнал паузы
	close(c.pauseCh)
	c.pauseCh = make(chan struct{})

	// Запускаем таймер для автоматического возобновления
	go func() {
		for {
			select {
			case <-ctx.Done():
			case <-time.After(duration):
				c.resume()
			}
		}
	}()
}

func (c *Client) resume() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isPaused {
		return
	}

	c.isPaused = false
	close(c.resumeCh)
	c.resumeCh = make(chan struct{})
}

func (c *Client) waitIfPaused(ctx context.Context) {
	c.mu.Lock()
	if !c.isPaused {
		c.mu.Unlock()
		return
	}
	pauseCh := c.pauseCh
	resumeCh := c.resumeCh
	c.mu.Unlock()

	select {
	case <-ctx.Done():
	case <-pauseCh:
		select {
		case <-ctx.Done():
		case <-resumeCh:
		}
	case <-resumeCh:
	}
}
