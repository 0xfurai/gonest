package users

import "github.com/0xfurai/gonest/example/fullstack-api/common"

// CreateUserDto is used for admin user creation.
type CreateUserDto struct {
	Email     string      `json:"email" validate:"required,email" swagger:"example=newuser@example.com"`
	Password  string      `json:"password" validate:"required,min=8" swagger:"example=securepass123"`
	FirstName string      `json:"firstName" validate:"required,min=1,max=100" swagger:"example=Jane"`
	LastName  string      `json:"lastName" validate:"required,min=1,max=100" swagger:"example=Smith"`
	Role      common.Role `json:"role" validate:"omitempty,oneof=admin user" swagger:"example=user"`
}

// UpdateUserDto is used for updating user profile.
type UpdateUserDto struct {
	FirstName string `json:"firstName,omitempty" validate:"omitempty,min=1,max=100" swagger:"example=Jane"`
	LastName  string `json:"lastName,omitempty" validate:"omitempty,min=1,max=100" swagger:"example=Smith"`
	AvatarURL string `json:"avatarUrl,omitempty" validate:"omitempty,max=2000" swagger:"example=https://example.com/avatar.jpg"`
}

// UpdateUserAdminDto allows admins to change roles and status.
type UpdateUserAdminDto struct {
	FirstName string      `json:"firstName,omitempty" validate:"omitempty,min=1,max=100" swagger:"example=Jane"`
	LastName  string      `json:"lastName,omitempty" validate:"omitempty,min=1,max=100" swagger:"example=Smith"`
	Role      common.Role `json:"role,omitempty" validate:"omitempty,oneof=admin user" swagger:"example=admin"`
	Status    string      `json:"status,omitempty" validate:"omitempty,oneof=active inactive" swagger:"example=active"`
}
