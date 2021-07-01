package service

import (
	"log"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
)

// repo имеет тип интерфейс
// cервисный интерфейс для спец. функций
type repo interface {
	Get(uid, key string) (model.DataEl, error)
	Put(uid, key string, value model.DataEl) error
	Del(uid, key string) error
	List(uid string) ([]string, error)
	GetUn(shortlink string) (model.DataEl, error)
}

// Service - содержит член repo
type Service struct {
	repo repo
}

// New - конструктор Service
func New(repo repo) *Service {
	return &Service{repo: repo}
}

// Put - when put to storage
func (s *Service) Put(uid, key string, value model.DataEl) error {
	if err := s.repo.Put(uid, key, value); err != nil {
		log.Printf("service/Put: put repo err: %v", err)
		return err
	}

	return nil
}

// Get - when get from storage
func (s *Service) Get(uid, key string) (model.DataEl, error) {
	value, err := s.repo.Get(uid, key)
	if err != nil {
		log.Printf("service/Get: get from repo err: %v", err)
		return model.DataEl{}, err
	}

	return value, nil
}

// Del when delete from storage
func (s *Service) Del(uid, key string) error {
	if err := s.repo.Del(uid, key); err != nil {
		log.Printf("service/Del: del repo err: %v", err)
		return err
	}

	return nil
}

// List - when get list of keys from storage
func (s *Service) List(uid string) ([]string, error) {
	items, err := s.repo.List(uid)
	if err != nil {
		log.Printf("service/List: get from repo err: %v", err)
		return nil, err
	}

	return items, nil
}

// GetUn - get unique link unonimously from storage
func (s *Service) GetUn(shortlink string) (model.DataEl, error) {
	value, err := s.repo.GetUn(shortlink)
	if err != nil {
		log.Printf("service/Get: get from repo err: %v", err)
		return model.DataEl{}, err
	}

	return value, nil
}
