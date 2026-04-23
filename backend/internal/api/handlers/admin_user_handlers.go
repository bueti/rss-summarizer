package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/domain/user"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AdminUserHandlers struct {
	userRepo repository.UserRepository
}

func NewAdminUserHandlers(userRepo repository.UserRepository) *AdminUserHandlers {
	return &AdminUserHandlers{
		userRepo: userRepo,
	}
}

type UserResponse struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	PictureURL *string `json:"picture_url,omitempty"`
	Role       string  `json:"role"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type ListUsersResponse struct {
	Body struct {
		Users []UserResponse `json:"users"`
	}
}

type UpdateUserRoleRequest struct {
	UserID string `path:"user_id" doc:"User ID"`
	Body   struct {
		Role string `json:"role" enum:"user,admin" doc:"User role"`
	}
}

type UpdateUserRoleResponse struct {
	Body UserResponse
}

func (h *AdminUserHandlers) Register(api huma.API) {
	// List all users - admin only
	huma.Register(api, huma.Operation{
		OperationID: "admin-list-users",
		Method:      http.MethodGet,
		Path:        "/v1/admin/users",
		Summary:     "List all users",
		Description: "Returns a list of all users in the system (admin only)",
		Tags:        []string{"Admin"},
	}, h.ListUsers)

	// Update user role - admin only
	huma.Register(api, huma.Operation{
		OperationID: "admin-update-user-role",
		Method:      http.MethodPut,
		Path:        "/v1/admin/users/{user_id}/role",
		Summary:     "Update user role",
		Description: "Update a user's role (admin only)",
		Tags:        []string{"Admin"},
	}, h.UpdateUserRole)
}

func (h *AdminUserHandlers) ListUsers(ctx context.Context, input *struct{}) (*ListUsersResponse, error) {
	// Admin check is enforced by AdminMiddleware on /v1/admin/*.
	users, err := h.userRepo.ListAll(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list users")
	}

	// Convert to response format
	userResponses := make([]UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = UserResponse{
			ID:         u.ID.String(),
			Email:      u.Email,
			Name:       u.Name,
			PictureURL: u.PictureURL,
			Role:       u.Role,
			CreatedAt:  u.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  u.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &ListUsersResponse{
		Body: struct {
			Users []UserResponse `json:"users"`
		}{
			Users: userResponses,
		},
	}, nil
}

func (h *AdminUserHandlers) UpdateUserRole(ctx context.Context, input *UpdateUserRoleRequest) (*UpdateUserRoleResponse, error) {
	// Admin check is enforced by AdminMiddleware on /v1/admin/*. We still need
	// the caller ID to prevent an admin removing their own role.
	callerID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	targetUserID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid user ID")
	}

	// Validate role
	if input.Body.Role != user.RoleUser && input.Body.Role != user.RoleAdmin {
		return nil, huma.Error400BadRequest("Role must be 'user' or 'admin'")
	}

	// Prevent user from removing their own admin role
	if targetUserID == callerID && input.Body.Role != user.RoleAdmin {
		return nil, huma.Error400BadRequest("Cannot remove your own admin role")
	}

	// Update role
	if err := h.userRepo.UpdateRole(ctx, targetUserID, input.Body.Role); err != nil {
		return nil, huma.Error500InternalServerError("Failed to update user role")
	}

	// Get updated user
	updatedUser, err := h.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch updated user")
	}

	return &UpdateUserRoleResponse{
		Body: UserResponse{
			ID:         updatedUser.ID.String(),
			Email:      updatedUser.Email,
			Name:       updatedUser.Name,
			PictureURL: updatedUser.PictureURL,
			Role:       updatedUser.Role,
			CreatedAt:  updatedUser.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  updatedUser.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
