package users

import (
	"time"

	"github.com/0xfurai/gonest/example/fullstack-api/common"
)

// User is the user entity stored in the database.
type User struct {
	ID        int         `json:"id"`
	Email     string      `json:"email"`
	Password  string      `json:"-" serialize:"exclude"`
	FirstName string      `json:"firstName"`
	LastName  string      `json:"lastName"`
	Role      common.Role `json:"role"`
	Status    string      `json:"status"`
	AvatarURL string      `json:"avatarUrl,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

// UserPublic is the public-facing user response (no password).
type UserPublic struct {
	ID        int         `json:"id" swagger:"example=1"`
	Email     string      `json:"email" swagger:"example=user@example.com"`
	FirstName string      `json:"firstName" swagger:"example=John"`
	LastName  string      `json:"lastName" swagger:"example=Doe"`
	Role      common.Role `json:"role" swagger:"example=user"`
	Status    string      `json:"status" swagger:"example=active"`
	AvatarURL string      `json:"avatarUrl,omitempty" swagger:"example=https://example.com/avatar.jpg"`
	CreatedAt time.Time   `json:"createdAt" swagger:"format=date-time"`
}

func (u *User) ToPublic() UserPublic {
	return UserPublic{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		Status:    u.Status,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
}
