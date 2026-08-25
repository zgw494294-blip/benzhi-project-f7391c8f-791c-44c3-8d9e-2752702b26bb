package application

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"seed-vigor-gate/internal/domain"
)

type Clock interface{ Now() time.Time }
type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type IDGenerator interface{ New(string) string }
type randomIDs struct{}

func (randomIDs) New(prefix string) string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return prefix + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + hex.EncodeToString(raw[:])
}

type caseListKey struct {
	status string
	limit  int
}

type Service struct {
	repository Repository
	clock      Clock
	ids        IDGenerator
	listMu     sync.RWMutex
	caseLists  map[caseListKey][]domain.QualificationCase
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, clock: realClock{}, ids: randomIDs{}, caseLists: make(map[caseListKey][]domain.QualificationCase)}
}
func NewServiceWithDependencies(repository Repository, clock Clock, ids IDGenerator) *Service {
	return &Service{repository: repository, clock: clock, ids: ids, caseLists: make(map[caseListKey][]domain.QualificationCase)}
}

func validateMeta(meta WriteMeta) error {
	if meta.ExpectedVersion < 1 {
		return fieldError("expectedVersion", "expectedVersion 必须为正整数")
	}
	if len(strings.TrimSpace(meta.IdempotencyKey)) < 8 || len(meta.IdempotencyKey) > 120 {
		return fieldError("idempotencyKey", "幂等键长度须在 8 至 120 之间")
	}
	if strings.TrimSpace(meta.Actor) == "" {
		return fieldError("actor", "操作人不能为空")
	}
	return nil
}
