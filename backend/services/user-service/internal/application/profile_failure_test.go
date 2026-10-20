package application

import (
	"errors"
	repository "go-cloud-storage/backend/services/user-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/user-service/internal/domain"
	"gorm.io/gorm"
	"testing"
)

type profileRepoFailure struct {
	repository.UserRepository
	err error
}

func (r profileRepoFailure) GetUserInfoById(int) (*domain.User, error) { return nil, r.err }
func TestProfilePreservesRepositoryFailureCause(t *testing.T) {
	for _, cause := range []error{gorm.ErrRecordNotFound, errors.New("connection unavailable")} {
		service := &userService{userRepo: profileRepoFailure{err: cause}}
		_, err := service.GetProfile(1)
		if !errors.Is(err, cause) {
			t.Fatalf("lost cause: %v", err)
		}
	}
}
