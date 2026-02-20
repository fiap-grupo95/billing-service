package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"mecanica_xpto/internal/domain/entities"
	mock_interfaces "mecanica_xpto/internal/usecase/interfaces/mocks"

	"go.uber.org/mock/gomock"
)

func TestEstimateUseCase_CalculateEstimate(t *testing.T) {
	t.Run("invalid os id", func(t *testing.T) {
		uc := NewEstimateUseCase(nil)
		_, err := uc.CalculateEstimate(context.Background(), "   ", 10)
		if !errors.Is(err, ErrInvalidOSID) {
			t.Fatalf("expected ErrInvalidOSID, got %v", err)
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		uc := NewEstimateUseCase(nil)
		_, err := uc.CalculateEstimate(context.Background(), "os-1", 0)
		if !errors.Is(err, ErrInvalidEstimateVal) {
			t.Fatalf("expected ErrInvalidEstimateVal, got %v", err)
		}
	})

	t.Run("repo get by os id error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)

		repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(entities.Estimate{}, errors.New("db"))

		_, err := uc.CalculateEstimate(context.Background(), "os-1", 10)
		if err == nil || err.Error() != "db" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)

		repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(entities.Estimate{ID: "existing"}, nil)

		_, err := uc.CalculateEstimate(context.Background(), "os-1", 10)
		if !errors.Is(err, ErrEstimateAlreadyExists) {
			t.Fatalf("expected ErrEstimateAlreadyExists, got %v", err)
		}
	})

	t.Run("create success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)

		repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(entities.Estimate{}, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(entities.Estimate{})).DoAndReturn(
			func(_ context.Context, e entities.Estimate) (entities.Estimate, error) {
				if e.ID == "" || e.OSID != "os-1" || e.Price != 125.5 || e.Status != entities.EstimateStatusPendente {
					t.Fatalf("unexpected estimate: %+v", e)
				}
				if e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
					t.Fatalf("expected timestamps")
				}
				return e, nil
			},
		)

		res, err := uc.CalculateEstimate(context.Background(), " os-1 ", 125.5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID == "" {
			t.Fatalf("expected generated id")
		}
	})
}

func TestEstimateUseCase_UpdateStatusByOSIDFlows(t *testing.T) {
	cases := []struct {
		name   string
		call   func(uc *EstimateUseCase, ctx context.Context, osID string) (entities.Estimate, error)
		status entities.EstimateStatus
	}{
		{name: "approve", call: (*EstimateUseCase).ApproveByOSID, status: entities.EstimateStatusAprovado},
		{name: "reject", call: (*EstimateUseCase).RejectByOSID, status: entities.EstimateStatusRejeitado},
		{name: "cancel", call: (*EstimateUseCase).CancelByOSID, status: entities.EstimateStatusCancelado},
	}

	for _, tc := range cases {
		t.Run(tc.name+" invalid os", func(t *testing.T) {
			uc := NewEstimateUseCase(nil)
			_, err := tc.call(uc, context.Background(), "")
			if !errors.Is(err, ErrInvalidOSID) {
				t.Fatalf("expected ErrInvalidOSID, got %v", err)
			}
		})

		t.Run(tc.name+" repo error", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().UpdateStatusByOSID(gomock.Any(), "os-1", tc.status).Return(entities.Estimate{}, errors.New("db"))

			_, err := tc.call(uc, context.Background(), "os-1")
			if err == nil || err.Error() != "db" {
				t.Fatalf("expected db error, got %v", err)
			}
		})

		t.Run(tc.name+" not found", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().UpdateStatusByOSID(gomock.Any(), "os-1", tc.status).Return(entities.Estimate{}, nil)

			_, err := tc.call(uc, context.Background(), "os-1")
			if !errors.Is(err, ErrEstimateNotFound) {
				t.Fatalf("expected ErrEstimateNotFound, got %v", err)
			}
		})

		t.Run(tc.name+" success", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			expected := entities.Estimate{ID: "id-1", OSID: "os-1", Status: tc.status}
			repo.EXPECT().UpdateStatusByOSID(gomock.Any(), "os-1", tc.status).Return(expected, nil)

			res, err := tc.call(uc, context.Background(), " os-1 ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Status != tc.status {
				t.Fatalf("expected %s got %s", tc.status, res.Status)
			}
		})
	}
}

func TestEstimateUseCase_UpdateEstimatePrice(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		uc := NewEstimateUseCase(nil)
		_, err := uc.UpdateEstimatePrice(context.Background(), " ", 10)
		if !errors.Is(err, ErrInvalidEstimateID) {
			t.Fatalf("expected ErrInvalidEstimateID, got %v", err)
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		uc := NewEstimateUseCase(nil)
		_, err := uc.UpdateEstimatePrice(context.Background(), "id-1", 0)
		if !errors.Is(err, ErrInvalidEstimateVal) {
			t.Fatalf("expected ErrInvalidEstimateVal, got %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)
		repo.EXPECT().UpdatePriceByID(gomock.Any(), "id-1", 10.5).Return(entities.Estimate{}, errors.New("db"))

		_, err := uc.UpdateEstimatePrice(context.Background(), "id-1", 10.5)
		if err == nil || err.Error() != "db" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)
		repo.EXPECT().UpdatePriceByID(gomock.Any(), "id-1", 10.5).Return(entities.Estimate{}, nil)

		_, err := uc.UpdateEstimatePrice(context.Background(), "id-1", 10.5)
		if !errors.Is(err, ErrEstimateNotFound) {
			t.Fatalf("expected ErrEstimateNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewEstimateUseCase(repo)
		now := time.Now()
		expected := entities.Estimate{ID: "id-1", Price: 10.5, UpdatedAt: now}
		repo.EXPECT().UpdatePriceByID(gomock.Any(), "id-1", 10.5).Return(expected, nil)

		res, err := uc.UpdateEstimatePrice(context.Background(), " id-1 ", 10.5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID != "id-1" || res.Price != 10.5 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

func TestEstimateUseCase_Getters(t *testing.T) {
	t.Run("GetByID", func(t *testing.T) {
		t.Run("invalid id", func(t *testing.T) {
			uc := NewEstimateUseCase(nil)
			_, err := uc.GetByID(context.Background(), "")
			if !errors.Is(err, ErrInvalidEstimateID) {
				t.Fatalf("expected ErrInvalidEstimateID, got %v", err)
			}
		})

		t.Run("repo error", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(entities.Estimate{}, errors.New("db"))

			_, err := uc.GetByID(context.Background(), "id-1")
			if err == nil || err.Error() != "db" {
				t.Fatalf("expected db error, got %v", err)
			}
		})

		t.Run("not found", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(entities.Estimate{}, nil)

			_, err := uc.GetByID(context.Background(), "id-1")
			if !errors.Is(err, ErrEstimateNotFound) {
				t.Fatalf("expected ErrEstimateNotFound, got %v", err)
			}
		})

		t.Run("success", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			expected := entities.Estimate{ID: "id-1"}
			repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(expected, nil)

			res, err := uc.GetByID(context.Background(), " id-1 ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.ID != "id-1" {
				t.Fatalf("unexpected result: %+v", res)
			}
		})
	})

	t.Run("GetByOSID", func(t *testing.T) {
		t.Run("invalid os id", func(t *testing.T) {
			uc := NewEstimateUseCase(nil)
			_, err := uc.GetByOSID(context.Background(), "")
			if !errors.Is(err, ErrInvalidOSID) {
				t.Fatalf("expected ErrInvalidOSID, got %v", err)
			}
		})

		t.Run("repo error", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(entities.Estimate{}, errors.New("db"))

			_, err := uc.GetByOSID(context.Background(), "os-1")
			if err == nil || err.Error() != "db" {
				t.Fatalf("expected db error, got %v", err)
			}
		})

		t.Run("not found", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(entities.Estimate{}, nil)

			_, err := uc.GetByOSID(context.Background(), "os-1")
			if !errors.Is(err, ErrEstimateNotFound) {
				t.Fatalf("expected ErrEstimateNotFound, got %v", err)
			}
		})

		t.Run("success", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			uc := NewEstimateUseCase(repo)
			expected := entities.Estimate{ID: "id-1", OSID: "os-1"}
			repo.EXPECT().GetByOSID(gomock.Any(), "os-1").Return(expected, nil)

			res, err := uc.GetByOSID(context.Background(), " os-1 ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.OSID != "os-1" {
				t.Fatalf("unexpected result: %+v", res)
			}
		})
	})
}
