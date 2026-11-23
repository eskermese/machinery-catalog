package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/pkg/metrics/mock_metrics"
)

func TestGetLatencyCallback(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	cases := []struct {
		name     string
		mockFunc func(t *testing.T) HistogramVec
		err      error
	}{
		{
			name: "успешный случай",
			mockFunc: func(t *testing.T) HistogramVec {
				t.Helper()

				h := mock_metrics.NewMockHistogramVec(ctrl)
				o := mock_metrics.NewMockObserver(ctrl)

				h.EXPECT().WithLabelValues(Success).Return(o)
				o.EXPECT().Observe(gomock.Any())

				return h
			},
			err: nil,
		},
		{
			name: "неуспешный случай",
			mockFunc: func(t *testing.T) HistogramVec {
				t.Helper()

				h := mock_metrics.NewMockHistogramVec(ctrl)
				o := mock_metrics.NewMockObserver(ctrl)

				h.EXPECT().WithLabelValues(Failure).Return(o)
				o.EXPECT().Observe(gomock.Any())

				return h
			},
			err: assert.AnError,
		},
	}

	for _, s := range cases {
		s := s

		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			histogramVec := s.mockFunc(t)

			GetLatencyCallback(time.Now(), histogramVec)(s.err)
		})
	}
}
