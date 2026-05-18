package scheduler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type CreditPaymentProcessor interface {
	ProcessDuePayments(ctx context.Context) (processed int, paid int, overdue int, err error)
}

type CreditPaymentScheduler struct {
	processor CreditPaymentProcessor
	interval  time.Duration
	logger    *logrus.Logger
}

func NewCreditPaymentScheduler(
	processor CreditPaymentProcessor,
	interval time.Duration,
	logger *logrus.Logger,
) *CreditPaymentScheduler {
	if interval <= 0 {
		interval = 12 * time.Hour
	}

	return &CreditPaymentScheduler{
		processor: processor,
		interval:  interval,
		logger:    logger,
	}
}

func (s *CreditPaymentScheduler) Start(ctx context.Context) {
	if s.logger != nil {
		s.logger.Infof("credit payment scheduler started, interval: %s", s.interval)
	}

	s.process(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if s.logger != nil {
				s.logger.Info("credit payment scheduler stopped")
			}
			return
		case <-ticker.C:
			s.process(ctx)
		}
	}
}

func (s *CreditPaymentScheduler) process(ctx context.Context) {
	if s.processor == nil {
		if s.logger != nil {
			s.logger.Warn("credit payment scheduler has no processor")
		}
		return
	}

	processed, paid, overdue, err := s.processor.ProcessDuePayments(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Errorf("failed to process credit payments: %v", err)
		}
		return
	}

	if s.logger != nil {
		s.logger.Infof(
			"credit payments processed: total=%d paid=%d overdue=%d",
			processed,
			paid,
			overdue,
		)
	}
}
