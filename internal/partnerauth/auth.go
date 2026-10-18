package partnerauth

import (
	"MyKitchen/internal/domain"
	"context"
	"crypto/sha256"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidKey = errors.New("invalid partner key")

type Repository interface {
	FindActivePartnerByKeyHash(context.Context, [32]byte) (domain.PartnerID, error)
	OwnsRestaurant(context.Context, domain.PartnerID, domain.RestaurantID) (bool, error)
}

type Authenticator interface {
	Authenticate(context.Context, string) (domain.PartnerID, error)
	OwnsRestaurant(context.Context, domain.PartnerID, domain.RestaurantID) (bool, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Authenticate(ctx context.Context, key string) (domain.PartnerID, error) {
	if key == "" {
		return 0, ErrInvalidKey
	}
	id, err := s.repo.FindActivePartnerByKeyHash(ctx, sha256.Sum256([]byte(key)))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidKey
	}
	return id, err
}

func (s *Service) OwnsRestaurant(ctx context.Context, partnerID domain.PartnerID, restaurantID domain.RestaurantID) (bool, error) {
	if partnerID <= 0 || restaurantID <= 0 {
		return false, nil
	}
	return s.repo.OwnsRestaurant(ctx, partnerID, restaurantID)
}

type contextKey struct{}

func WithPartnerID(ctx context.Context, id domain.PartnerID) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func PartnerID(ctx context.Context) domain.PartnerID {
	id, _ := ctx.Value(contextKey{}).(domain.PartnerID)
	return id
}
