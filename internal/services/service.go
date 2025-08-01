package services

type Service struct {
	Payment interface {
		Send(correlationID string, amount float64) error
	}
}

func NewServices() *Service {
	return &Service{
		Payment: &PaymentService{},
	}
}
