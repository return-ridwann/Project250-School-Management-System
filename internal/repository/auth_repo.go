package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"sms/internal/models"
)

const (
	DefaultAdminEmail    = "admin@school.com"
	DefaultAdminPassword = "admin123"
	DefaultAdminRole     = "admin"
)

// this handles db ops for user auth and management
type Authrepo struct {
	db *sql.DB
}

// creates a new instance of Authrepo
func NewAuthrepo(db *sql.DB) *Authrepo {
	return &Authrepo{db: db}
}

// hashes plain pass using bcrypt and stores a new user in db
func (r *Authrepo) CreateUser(email, password, role string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `INSERT INTO users (email, password_hash, role) VALUES (?, ?, ?)`
	_, err = r.db.Exec(query, email, string(hashedPassword), role)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return r.GetUserByEmail(email)
}

// fetches a user record by email
func (r *Authrepo) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE email = ?`
	row := r.db.QueryRow(query, email)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}

	return &user, nil
}

// checks if any admin account exists on startup
// If none exists, it creates one default admin account
func (r *Authrepo) SeedDefaultAdmin() error {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE role = 'admin'`
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query admin count: %w", err)
	}

	if count > 0 {
		return nil
	}

	_, err = r.CreateUser(DefaultAdminEmail, DefaultAdminPassword, DefaultAdminRole)
	if err != nil {
		return fmt.Errorf("failed to seed default admin: %w", err)
	}

	log.Printf("Default admin seeded: %s\n", DefaultAdminEmail)
	return nil
}

// verifies a plain pass against a bcrypt hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}



// creates a new user using a db connection
func CreateUser(db *sql.DB, email, password, role string) (*models.User, error) {
	return NewAuthrepo(db).CreateUser(email, password, role)
}

// fetches a user by email using a db connection
func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	return NewAuthrepo(db).GetUserByEmail(email)
}

// seeds the default admin user if none exists
func SeedDefaultAdmin(db *sql.DB) error {
	return NewAuthrepo(db).SeedDefaultAdmin()
}
