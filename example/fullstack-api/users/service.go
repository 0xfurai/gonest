package users

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
)

// UsersService handles user business logic backed by a SQL database.
type UsersService struct {
	db *sql.DB
}

func NewUsersService(db *sql.DB) *UsersService {
	return &UsersService{db: db}
}

// Seed inserts initial data if the users table is empty.
func (s *UsersService) Seed() error {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count > 0 {
		return nil
	}
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO users (email, password, first_name, last_name, role, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"admin@example.com", HashPassword("admin123"),
		"Admin", "User", string(common.RoleAdmin), "active", now, now,
	)
	return err
}

func (s *UsersService) Create(dto CreateUserDto) (*User, error) {
	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER(?)", dto.Email).Scan(&exists)
	if exists > 0 {
		return nil, gonest.NewConflictException("email already registered")
	}

	role := dto.Role
	if role == "" {
		role = common.RoleUser
	}
	now := time.Now()

	result, err := s.db.Exec(
		`INSERT INTO users (email, password, first_name, last_name, role, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.ToLower(dto.Email), HashPassword(dto.Password),
		dto.FirstName, dto.LastName, string(role), "active", now, now,
	)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to create user: " + err.Error())
	}

	id, _ := result.LastInsertId()
	return s.FindByID(int(id)), nil
}

func (s *UsersService) FindAll(offset, limit int) ([]*User, int) {
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)

	rows, err := s.db.Query(
		`SELECT id, email, password, first_name, last_name, role, status, avatar_url, created_at, updated_at
		 FROM users ORDER BY id LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, total
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		if u := scanUserRows(rows); u != nil {
			users = append(users, u)
		}
	}
	return users, total
}

func (s *UsersService) FindByID(id int) *User {
	return scanUserRow(s.db.QueryRow(
		`SELECT id, email, password, first_name, last_name, role, status, avatar_url, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	))
}

func (s *UsersService) FindByEmail(email string) *User {
	return scanUserRow(s.db.QueryRow(
		`SELECT id, email, password, first_name, last_name, role, status, avatar_url, created_at, updated_at
		 FROM users WHERE LOWER(email) = LOWER(?)`, email,
	))
}

func (s *UsersService) Update(id int, dto UpdateUserDto) (*User, error) {
	var sets []string
	var args []any

	if dto.FirstName != "" {
		sets = append(sets, "first_name = ?")
		args = append(args, dto.FirstName)
	}
	if dto.LastName != "" {
		sets = append(sets, "last_name = ?")
		args = append(args, dto.LastName)
	}
	if dto.AvatarURL != "" {
		sets = append(sets, "avatar_url = ?")
		args = append(args, dto.AvatarURL)
	}
	if len(sets) == 0 {
		return s.FindByID(id), nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now(), id)

	_, err := s.db.Exec("UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		return nil, gonest.NewInternalServerError("update failed: " + err.Error())
	}
	user := s.FindByID(id)
	if user == nil {
		return nil, gonest.NewNotFoundException("user not found")
	}
	return user, nil
}

func (s *UsersService) UpdateAdmin(id int, dto UpdateUserAdminDto) (*User, error) {
	var sets []string
	var args []any

	if dto.FirstName != "" {
		sets = append(sets, "first_name = ?")
		args = append(args, dto.FirstName)
	}
	if dto.LastName != "" {
		sets = append(sets, "last_name = ?")
		args = append(args, dto.LastName)
	}
	if dto.Role != "" {
		sets = append(sets, "role = ?")
		args = append(args, string(dto.Role))
	}
	if dto.Status != "" {
		sets = append(sets, "status = ?")
		args = append(args, dto.Status)
	}
	if len(sets) == 0 {
		return s.FindByID(id), nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now(), id)

	_, err := s.db.Exec("UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		return nil, gonest.NewInternalServerError("update failed: " + err.Error())
	}
	user := s.FindByID(id)
	if user == nil {
		return nil, gonest.NewNotFoundException("user not found")
	}
	return user, nil
}

func (s *UsersService) Delete(id int) error {
	result, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return gonest.NewInternalServerError("delete failed: " + err.Error())
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return gonest.NewNotFoundException("user not found")
	}
	return nil
}

func (s *UsersService) VerifyPassword(user *User, password string) bool {
	return CheckPassword(user.Password, password)
}

// --- Password hashing (salted SHA-256, production should use bcrypt) ---

func HashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	h := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(h[:])
}

func CheckPassword(stored, password string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	h := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(h[:]) == parts[1]
}

// --- Row scanners ---

func scanUserRows(rows *sql.Rows) *User {
	u := &User{}
	var role, status string
	if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&role, &status, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	u.Role = common.Role(role)
	u.Status = status
	return u
}

func scanUserRow(row *sql.Row) *User {
	u := &User{}
	var role, status string
	if err := row.Scan(&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&role, &status, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	u.Role = common.Role(role)
	u.Status = status
	return u
}
