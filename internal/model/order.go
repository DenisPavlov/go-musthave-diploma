package model

import "time"

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

type AccrualOrderStatus string

const (
	AccrualStatusRegistered AccrualOrderStatus = "REGISTERED"
	AccrualStatusInvalid    AccrualOrderStatus = "INVALID"
	AccrualStatusProcessing AccrualOrderStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualOrderStatus = "PROCESSED"
)

type Order struct {
	Number     string    `json:"number"`
	Status     Status    `json:"status"`
	Accrual    float32   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type AccrualOrder struct {
	Order   string             `json:"order"`
	Status  AccrualOrderStatus `json:"status"`
	Accrual float32            `json:"accrual,omitempty"`
}
