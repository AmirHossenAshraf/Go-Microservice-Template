package service

import (
	"Go-Microservice-Template/internal/model"
	"Go-Microservice-Template/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// UserService defines the business operations for users.
type UserService interface {
	Register(ctx context.Context, req model.CreateUserRequest) (*model.User, error)
	Login(ctx context.Context, req model.LoginRequest, jwtSecret string, expHours int) (*model.LoginResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) (*model.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params model.ListParams) (*model.ListResponse[model.User], error)
}

type userService struct {
	repo  repository.UserRepository
	cache repository.UserCache
}

// NewUserService creates a new user service with repository and cache dependencies.
func NewUserService(repo repository.UserRepository, cache repository.UserCache) UserService {
	return &userService{repo: repo, cache: cache}
}

func (s *userService) Register(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
	// Check if email already exists
	existing, _ := s.repo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, repository.ErrDuplicate
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPassword),
		Role:     model.RoleUser,
		Active:   true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Warm cache
	if err := s.cache.Set(ctx, user); err != nil {
		log.Warn().Err(err).Msg("failed to cache new user")
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, req model.LoginRequest, jwtSecret string, expHours int) (*model.LoginResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate JWT
	expiresAt := time.Now().Add(time.Duration(expHours) * time.Hour)
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"role":  string(user.Role),
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &model.LoginResponse{
		Token:     tokenStr,
		ExpiresAt: expiresAt,
		User:      *user,
	}, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	// Try cache first
	user, err := s.cache.Get(ctx, id)
	if err == nil && user != nil {
		log.Debug().Str("user_id", id.String()).Msg("cache hit")
		return user, nil
	}

	// Cache miss — fetch from database
	user, err = s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	if err := s.cache.Set(ctx, user); err != nil {
		log.Warn().Err(err).Msg("failed to update cache")
	}

	return user, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply partial updates
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Name != nil {
		user.Name = *req.Name
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate cache
	if err := s.cache.Delete(ctx, id); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate cache")
	}

	return user, nil
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.Delete(ctx, id); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate cache after delete")
	}

	return nil
}

func (s *userService) List(ctx context.Context, params model.ListParams) (*model.ListResponse[model.User], error) {
	users, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[model.User]{
		Items:      users,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}
