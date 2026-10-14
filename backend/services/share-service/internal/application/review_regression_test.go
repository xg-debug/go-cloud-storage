package application

import (
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"

	contracts "go-cloud-storage/backend/pkg/contracts"
	repository "go-cloud-storage/backend/services/share-service/internal/adapters/repository"
	domain "go-cloud-storage/backend/services/share-service/internal/domain"
)

type shareListRepo struct {
	repository.ShareRepository
	shares []*domain.Share
}

func (r shareListRepo) GetUserShares(int, int, int) ([]*domain.Share, int64, error) {
	return r.shares, int64(len(r.shares)), nil
}

type shareFiles struct{ ShareFileReader }

func (shareFiles) GetFileByIds([]string) ([]contracts.FileMetadata, error) {
	return []contracts.FileMetadata{{Id: "f", UserId: 1, ShareVersion: 2}}, nil
}
func TestShareListHidesInvalidatedVersionBeforeEventDelivery(t *testing.T) {
	s := NewShareService(shareListRepo{shares: []*domain.Share{{Id: 1, FileId: "f", FileVersion: 1}, {Id: 2, FileId: "f", FileVersion: 2}}}, shareFiles{}, nil, nil)
	items, _, err := s.GetUserShares(1, 1, 20)
	if err != nil || len(items) != 1 || items[0].Id != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
}

// The other request wins between the initial lookup and INSERT.
type racedShareRepo struct {
	repository.ShareRepository
	lookups int
	winner  *domain.Share
}

func (r *racedShareRepo) IsShared(string) (bool, *domain.Share) {
	r.lookups++
	return r.lookups > 1, r.winner
}
func (r *racedShareRepo) CreateShare(*domain.Share) (*domain.Share, error) {
	return nil, &mysqlDriver.MySQLError{Number: 1062, Message: "duplicate file"}
}
func (shareFiles) GetUserFileByID(int, string) (*contracts.FileMetadata, error) {
	return &contracts.FileMetadata{Id: "f", UserId: 1, ShareVersion: 2}, nil
}
func TestConcurrentShareCreationReturnsExistingWinner(t *testing.T) {
	winner := &domain.Share{Id: 8, FileId: "f", FileVersion: 2}
	repo := &racedShareRepo{winner: winner}
	got, err := NewShareService(repo, shareFiles{}, nil, nil).CreateShare(1, "f", 1, "")
	if err != nil || got != winner {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
