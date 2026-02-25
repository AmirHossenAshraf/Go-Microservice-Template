package service

import (
	"Go-Microservice-Template/internal/repository"
)

// UserService defines the business operations for users.
type UserService interface {
}

type userService struct {
	repo  repository.UserRepository
	cache repository.UserCache
}

// NewUserService creates a new user service with repository and cache dependencies.
func NewUserService(repo repository.UserRepository, cache repository.UserCache) UserService {
	return &userService{repo: repo, cache: cache}
}
