package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"sms/internal/models"
)

// handles database operations for students
type StudentRepository struct {
	db *sql.DB
}

// creates a new StudentRepository instance
func NewStudentRepository(db *sql.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

// creates a new student and their user account inside a transaction
// First inserts into users table then inserts into students table
// Automatically rolls back if any error occurs
func (r *StudentRepository) CreateStudent(ctx context.Context, user *models.User, student *models.Student) (*models.Student, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if user == nil || student == nil {
		return nil, errors.New("user and student must not be nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Hash password if not already hashed with bcrypt
	passwordHash := user.PasswordHash
	if passwordHash != "" && !strings.HasPrefix(passwordHash, "$2") {
		hashed, err := bcrypt.GenerateFromPassword([]byte(passwordHash), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = string(hashed)
	}

	// Insert user with role 'student'
	userQuery := `INSERT INTO users (email, password_hash, role) VALUES (?, ?, 'student')`
	userRes, err := tx.ExecContext(ctx, userQuery, user.Email, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	userID, err := userRes.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert user id: %w", err)
	}

	// Insert student profile
	studentQuery := `
		INSERT INTO students (user_id, roll_number, full_name, class_id, date_of_birth, guardian_phone)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	studentRes, err := tx.ExecContext(
		ctx,
		studentQuery,
		userID,
		student.RollNumber,
		student.FullName,
		student.ClassID,
		student.DateOfBirth,
		student.GuardianPhone,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert student: %w", err)
	}

	studentID, err := studentRes.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert student id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	student.ID = int(studentID)
	student.UserID = int(userID)
	student.Email = user.Email

	// insert class name if class is assigned
	if student.ClassID != nil {
		var className string
		_ = r.db.QueryRowContext(ctx, "SELECT name FROM classes WHERE id = ?", *student.ClassID).Scan(&className)
		student.ClassName = className
	}

	return student, nil
}

// returns all students joined with email and class name
func (r *StudentRepository) GetAllStudents(classID string, search string) ([]models.Student, error) {
	return r.GetAllStudentsContext(context.Background(), classID, search)
}

// returns all students with context, joined with email and class name
func (r *StudentRepository) GetAllStudentsContext(ctx context.Context, classID string, search string) ([]models.Student, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	query := `
		SELECT 
			s.id, 
			s.user_id, 
			s.roll_number, 
			s.full_name, 
			s.class_id, 
			s.date_of_birth, 
			s.guardian_phone,
			u.email,
			COALESCE(c.name, '') AS class_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN classes c ON s.class_id = c.id
	`

	var conditions []string
	var args []any

	trimmedClassID := strings.TrimSpace(classID)
	if trimmedClassID != "" && trimmedClassID != "all" {
		conditions = append(conditions, "s.class_id = ?")
		args = append(args, trimmedClassID)
	}

	trimmedSearch := strings.TrimSpace(search)
	if trimmedSearch != "" {
		conditions = append(conditions, "(s.full_name LIKE ? OR s.roll_number LIKE ?)")
		pattern := "%" + trimmedSearch + "%"
		args = append(args, pattern, pattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY s.id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query students: %w", err)
	}
	defer rows.Close()

	students := []models.Student{}
	for rows.Next() {
		var s models.Student
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.RollNumber,
			&s.FullName,
			&s.ClassID,
			&s.DateOfBirth,
			&s.GuardianPhone,
			&s.Email,
			&s.ClassName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student row: %w", err)
		}
		students = append(students, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return students, nil
}

// fetches a single student's full details by student ID
func (r *StudentRepository) GetStudentByID(id int64) (*models.Student, error) {
	return r.GetStudentByIDContext(context.Background(), id)
}

// fetches a single student's full details by student ID with context
func (r *StudentRepository) GetStudentByIDContext(ctx context.Context, id int64) (*models.Student, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	query := `
		SELECT 
			s.id, 
			s.user_id, 
			s.roll_number, 
			s.full_name, 
			s.class_id, 
			s.date_of_birth, 
			s.guardian_phone,
			u.email,
			COALESCE(c.name, '') AS class_name
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN classes c ON s.class_id = c.id
		WHERE s.id = ?
	`

	var s models.Student
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.RollNumber,
		&s.FullName,
		&s.ClassID,
		&s.DateOfBirth,
		&s.GuardianPhone,
		&s.Email,
		&s.ClassName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get student by id: %w", err)
	}

	return &s, nil
}

// updates the profile fields of a student
func (r *StudentRepository) UpdateStudent(id int64, student *models.Student) error {
	return r.UpdateStudentContext(context.Background(), id, student)
}

// updates the profile fields of a student with context
func (r *StudentRepository) UpdateStudentContext(ctx context.Context, id int64, student *models.Student) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if student == nil {
		return errors.New("student must not be nil")
	}

	query := `
		UPDATE students
		SET roll_number = ?, full_name = ?, class_id = ?, date_of_birth = ?, guardian_phone = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(
		ctx,
		query,
		student.RollNumber,
		student.FullName,
		student.ClassID,
		student.DateOfBirth,
		student.GuardianPhone,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update student: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Also update users.email if email is inserted
	if student.Email != "" {
		_, _ = r.db.ExecContext(
			ctx,
			"UPDATE users SET email = ? WHERE id = (SELECT user_id FROM students WHERE id = ?)",
			student.Email,
			id,
		)
	}

	student.ID = int(id)
	return nil
}

// deletes the user and student record inside a transaction
func (r *StudentRepository) DeleteStudent(userID int64) error {
	return r.DeleteStudentContext(context.Background(), userID)
}

// deletes the user and student record inside a transaction with context
func (r *StudentRepository) DeleteStudentContext(ctx context.Context, userID int64) error {
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete student record
	resStudents, err := tx.ExecContext(ctx, "DELETE FROM students WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	// Delete user record
	resUsers, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsStudents, _ := resStudents.RowsAffected()
	rowsUsers, _ := resUsers.RowsAffected()

	if rowsStudents == 0 && rowsUsers == 0 {
		// Fallback: in case userID was the students.id rather than users.id
		var actualUserID int64
		err := tx.QueryRowContext(ctx, "SELECT user_id FROM students WHERE id = ?", userID).Scan(&actualUserID)
		if err == nil {
			_, err = tx.ExecContext(ctx, "DELETE FROM students WHERE id = ?", userID)
			if err != nil {
				return fmt.Errorf("failed to delete student: %w", err)
			}
			_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", actualUserID)
			if err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}
		} else {
			return sql.ErrNoRows
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
