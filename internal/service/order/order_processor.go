package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DenisPavlov/go-musthave-diploma/internal/client/accrual"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/DenisPavlov/go-musthave-diploma/internal/model"
)

type OrdersStorage interface {
	GetNewOrders(ctx context.Context, limit int) ([]model.Order, error)
	UpdateOrder(ctx context.Context, order model.Order) error
}

type AccrualClient interface {
	GetOrder(ctx context.Context, oderNum string) (*model.AccrualOrder, error)
}

type Processor struct {
	log           *slog.Logger
	accrualClient AccrualClient
	ordersStorage OrdersStorage
}

func NewProcessor(log *slog.Logger, accrualClient AccrualClient, ordersStorage OrdersStorage) *Processor {
	return &Processor{
		log:           log.With(slog.String("component", "order_processor")),
		accrualClient: accrualClient,
		ordersStorage: ordersStorage,
	}
}

func (s *Processor) ProcessNewOrders(ctx context.Context, batchSize int, workers int) error {
	ordersChan := make(chan model.Order, batchSize)
	doneChan := make(chan model.Order, batchSize)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.getStatusWorker(ctx, ordersChan, doneChan, errChan)
		}()
	}

	go func() {
		wg.Wait()
		close(doneChan)
		close(errChan)
	}()

	go s.updateOrders(ctx, doneChan, errChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			s.log.ErrorContext(ctx, "error while processing orders", logger.Err(err))
		default:
			orders, err := s.fetchNewOrders(ctx, batchSize)
			if err != nil {
				s.log.ErrorContext(ctx, "error while fetching orders", logger.Err(err))
			}

			if len(orders) == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second): // todo - вынести время в конфиг
					continue
				}
			}

			for _, order := range orders {
				select {
				case ordersChan <- order:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

func (s *Processor) updateOrders(ctx context.Context, doneChan <-chan model.Order, errChan chan<- error) {
	for order := range doneChan {
		err := s.ordersStorage.UpdateOrder(ctx, order)
		if err != nil {
			select {
			case errChan <- err:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *Processor) fetchNewOrders(ctx context.Context, batchSize int) ([]model.Order, error) {
	return s.ordersStorage.GetNewOrders(ctx, batchSize)
}

func (s *Processor) getStatusWorker(ctx context.Context, ordersChan <-chan model.Order, doneChan chan<- model.Order, errChan chan<- error) {
	for order := range ordersChan {
		accOrder, err := s.accrualClient.GetOrder(ctx, order.Number)
		if err != nil {
			if errors.Is(err, accrual.ErrOrderNotRegistered) {
				s.log.DebugContext(ctx, "accrual order not registered", logger.Err(err))
			} else {
				select {
				case errChan <- err:
				case <-ctx.Done():
					return
				}
			}
			continue
		}

		var status model.Status

		switch accOrder.Status {
		case model.AccrualStatusRegistered, model.AccrualStatusProcessing:
			status = model.StatusProcessing
		case model.AccrualStatusInvalid:
			status = model.StatusInvalid
		case model.AccrualStatusProcessed:
			status = model.StatusProcessed
		default:
			select {
			case errChan <- fmt.Errorf("unknown accrual status for oder: %s %s", order.Number, accOrder.Status):
			case <-ctx.Done():
				return
			}
			continue
		}

		order.Status = status
		if order.Status == model.StatusProcessed {
			order.Accrual = accOrder.Accrual
		}

		select {
		case doneChan <- order:
		case <-ctx.Done():
			return
		}
	}
}
