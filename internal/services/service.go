package services

type Service struct {
	Payment interface {
		Send(correlationID string, amount float64) error
	}
	Health interface {
		Start()
	}
}

func NewServices() *Service {
	return &Service{
		Payment: &PaymentService{},
	}
}
