package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DenisPavlov/go-musthave-diploma/internal/config"
	getBalance "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/balance/get"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/withdrawals"
	"github.com/DenisPavlov/go-musthave-diploma/internal/model"
	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrOrderWasAddedByCurrentUser = errors.New("order was added by current user")
	ErrOrderWasAddedByOtherUsers  = errors.New("order was added by other user")
	ErrUserAlreadyExists          = errors.New("user already exists")
	ErrUserNotFound               = errors.New("user not found")
	ErrWrongPassword              = errors.New("wrong password")
	ErrNotEnough                  = errors.New("not enough coins")
	ErrWrongOrderNumber           = errors.New("wrong order number")
)

type Storage struct {
	log *slog.Logger
	db  *sql.DB
}

func InitStorage(log *slog.Logger, cfg *config.Config) (*Storage, error) {
	const op = "storage.InitStorage"

	db, err := sql.Open("postgres", cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %s %w", op, err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %s %w", op, err)
	}

	if err := ApplyMigrations(db, log); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %s %w", op, err)
	}

	return &Storage{
		log: log.With("component", "storage"),
		db:  db,
	}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) GetAllOrders(ctx context.Context, username string) ([]model.Order, error) {
	op := "storage.GetAllOrders"
	s.log.DebugContext(ctx, "getting all orders", slog.String("username", username))

	var orders []model.Order

	stmt, err := s.db.Prepare("SELECT number, status, uploaded_at, accrual FROM orders WHERE username=$1")
	if err != nil {
		return orders, fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryContext(ctx, username)
	if err != nil {
		return orders, fmt.Errorf("failed to query statement: %s %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.Number, &order.Status, &order.UploadedAt, &order.Accrual); err != nil {
			return orders, fmt.Errorf("failed to scan row: %s %w", op, err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return orders, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return orders, nil
}

func (s *Storage) AddOrder(ctx context.Context, orderNum string, username string) error {
	op := "storage.AddOrder"

	s.log.DebugContext(ctx, "adding order", slog.String("orderNum", orderNum), slog.String("username", username))
	stmt, err := s.db.Prepare("INSERT INTO orders (number, username, status, uploaded_at) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, orderNum, username, model.StatusNew, time.Now())
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Debug("pg error has occurred", slog.String("error", pgErr.Detail))
			errorRowLogin, err := s.findUsernameByOrderNum(ctx, orderNum)
			if err != nil {
				return fmt.Errorf("failed to find username by order number: %s %w", op, err)
			}

			if errorRowLogin == username {
				return ErrOrderWasAddedByCurrentUser
			} else {
				return ErrOrderWasAddedByOtherUsers
			}
		}
		return fmt.Errorf("failed to execute statement: %s %w", op, err)
	}

	return nil
}

func (s *Storage) UpdateOrder(ctx context.Context, order model.Order) error {
	op := "storage.UpdateOrder"

	s.log.DebugContext(ctx, "updating order", slog.Any("order", order))
	stmt, err := s.db.Prepare("UPDATE orders SET status = $1, accrual = $2 WHERE number = $3")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, order.Status, order.Accrual, order.Number)
	if err != nil {
		return fmt.Errorf("failed to execute statement: %s %w", op, err)
	}
	return nil
}

func (s *Storage) GetNewOrders(ctx context.Context, limit int) ([]model.Order, error) {
	op := "storage.GetNewOrders"
	s.log.DebugContext(ctx, "getting new orders", slog.Int("limit", limit))

	var orders []model.Order

	stmt, err := s.db.Prepare("SELECT number, status, uploaded_at, accrual FROM orders WHERE status=$1")
	if err != nil {
		return orders, fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryContext(ctx, model.StatusNew)
	if err != nil {
		return orders, fmt.Errorf("failed to query statement: %s %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.Number, &order.Status, &order.UploadedAt, &order.Accrual); err != nil {
			return orders, fmt.Errorf("failed to scan row: %s %w", op, err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return orders, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return orders, nil

}

func (s *Storage) AddUser(ctx context.Context, username string, password string) error {
	op := "storage.AddUser"
	s.log.Debug("adding user", slog.String("username", username))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %s %w", op, err)
	}

	stmt, err := s.db.Prepare("INSERT INTO users (username, password_hash) VALUES ($1, $2)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, username, hashedPassword)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Debug("pg error has occurred", slog.String("error", pgErr.Detail))
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to execute statement: %s %w", op, err)
	}
	return nil
}

func (s *Storage) Login(ctx context.Context, username string, password string) error {
	op := "storage.Login"
	s.log.Debug("login user", slog.String("username", username))

	stmt, err := s.db.Prepare("SELECT password_hash FROM users WHERE username = $1")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	row := stmt.QueryRowContext(ctx, username)
	var hashStr string
	if err = row.Scan(&hashStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to scan row: %s %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(password)); err != nil {
		return ErrWrongPassword
	}
	return nil
}

func (s *Storage) GetBalance(ctx context.Context, username string) (getBalance.Balance, error) {
	op := "storage.GetBalance"
	s.log.Debug("getting balance", slog.String("username", username))

	var balance getBalance.Balance

	accrualSum, err := s.getAccrualSum(ctx, username)
	if err != nil {
		return balance, err
	}

	var withdrawnSum float32
	stmt, err := s.db.Prepare("SELECT COALESCE(SUM(sum), 0) FROM withdrawal WHERE username = $1")
	if err != nil {
		return balance, fmt.Errorf("failed to prepare sum statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()
	row := stmt.QueryRowContext(ctx, username)
	if err := row.Scan(&withdrawnSum); err != nil {
		return balance, fmt.Errorf("failed to scan sum row: %s %w", op, err)
	}

	balance = getBalance.Balance{
		Current:   accrualSum - withdrawnSum,
		Withdrawn: withdrawnSum,
	}

	return balance, nil
}

func (s *Storage) getAccrualSum(ctx context.Context, username string) (float32, error) {
	op := "storage.getAccrualSum"

	var accrualSum float32
	stmt, err := s.db.Prepare("SELECT SUM(accrual) FROM orders WHERE username = $1")
	if err != nil {
		return accrualSum, fmt.Errorf("failed to prepare accural statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()
	row := stmt.QueryRowContext(ctx, username)
	if err := row.Scan(&accrualSum); err != nil {
		return accrualSum, fmt.Errorf("failed to scan accural row: %s %w", op, err)
	}
	return accrualSum, nil
}

func (s *Storage) Withdraw(ctx context.Context, username string, orderNum string, sum float32) error {
	op := "storage.Withdraw"
	s.log.Debug("withdrawing", slog.String("username", username), slog.String("order", orderNum), slog.Float64("sum", float64(sum)))

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var accrualSum float32
	stmt, err := tx.Prepare("SELECT SUM(accrual) FROM orders WHERE username = $1")
	if err != nil {
		return fmt.Errorf("failed to prepare accural statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()
	row := stmt.QueryRowContext(ctx, username)
	if err := row.Scan(&accrualSum); err != nil {
		return fmt.Errorf("failed to scan accural row: %s %w", op, err)
	}

	if accrualSum < sum {
		return ErrNotEnough
	}

	stmt, err = tx.Prepare("INSERT INTO withdrawal (order_number, sum, username) VALUES ($1, $2, $3)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, orderNum, sum, username)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Debug("pg error has occurred", slog.String("error", pgErr.Detail))
			return ErrWrongOrderNumber
		}
		return fmt.Errorf("failed to execute statement: %s %w", op, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %s %w", op, err)
	}

	return nil
}

func (s *Storage) GetWithdrawals(ctx context.Context, username string) ([]withdrawals.Withdrawal, error) {
	op := "storage.GetWithdrawals"

	var result []withdrawals.Withdrawal
	stmt, err := s.db.Prepare("SELECT order_number, sum, processed_at FROM withdrawal WHERE username = $1 ORDER BY processed_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	defer func() {
		_ = stmt.Close()
	}()
	rows, err := stmt.QueryContext(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %s %w", op, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var withdrawal withdrawals.Withdrawal
		if err := rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return result, fmt.Errorf("failed to scan row: %s %w", op, err)
		}
		result = append(result, withdrawal)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("failed to scan row: %s %w", op, err)
	}
	return result, nil
}

func (s *Storage) findUsernameByOrderNum(ctx context.Context, number string) (string, error) {
	op := "storage.findLoginByOrder"
	s.log.Debug("finding login by order", slog.String("orderNum", number))
	stmt, err := s.db.Prepare("SELECT username FROM orders WHERE number = $1")
	if err != nil {
		return "", fmt.Errorf("failed to prepare statement: %s %w", op, err)
	}
	row := stmt.QueryRowContext(ctx, number)
	var username string
	if err := row.Scan(&username); err != nil {
		return "", fmt.Errorf("failed to query row: %s %w", op, err)
	}
	s.log.Debug("found login by order", slog.String("orderNum", number), slog.String("username", username))
	return username, nil
}
