package models

import "time"

type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"password_hash" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Teacher struct {
	ID          int     `json:"id" db:"id"`
	UserID      int     `json:"user_id" db:"user_id"`
	FullName    string  `json:"full_name" db:"full_name"`
	Designation string  `json:"designation" db:"designation"`
	Department  string  `json:"department" db:"department"`
	Phone       *string `json:"phone" db:"phone"`
}

type Student struct {
	ID            int     `json:"id" db:"id"`
	UserID        int     `json:"user_id" db:"user_id"`
	RollNumber    string  `json:"roll_number" db:"roll_number"`
	FullName      string  `json:"full_name" db:"full_name"`
	ClassID       *int    `json:"class_id" db:"class_id"`
	DateOfBirth   *string `json:"date_of_birth" db:"date_of_birth"`
	GuardianPhone *string `json:"guardian_phone" db:"guardian_phone"`
}

type Class struct {
	ID         int    `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	Section    string `json:"section" db:"section"`
	RoomNumber string `json:"room_number" db:"room_number"`
}

type Subject struct {
	ID        int    `json:"id" db:"id"`
	Code      string `json:"code" db:"code"`
	Title     string `json:"title" db:"title"`
	ClassID   int    `json:"class_id" db:"class_id"`
	TeacherID *int   `json:"teacher_id" db:"teacher_id"`
}

type Timetable struct {
	ID        int    `json:"id" db:"id"`
	ClassID   int    `json:"class_id" db:"class_id"`
	SubjectID int    `json:"subject_id" db:"subject_id"`
	TeacherID int    `json:"teacher_id" db:"teacher_id"`
	DayOfWeek string `json:"day_of_week" db:"day_of_week"`
	StartTime string `json:"start_time" db:"start_time"`
	EndTime   string `json:"end_time" db:"end_time"`
}

type Attendance struct {
	ID        int    `json:"id" db:"id"`
	StudentID int    `json:"student_id" db:"student_id"`
	ClassID   int    `json:"class_id" db:"class_id"`
	Date      string `json:"date" db:"date"`
	Status    string `json:"status" db:"status"`
}

type Grade struct {
	ID            int      `json:"id" db:"id"`
	StudentID     int      `json:"student_id" db:"student_id"`
	SubjectID     int      `json:"subject_id" db:"subject_id"`
	ExamType      string   `json:"exam_type" db:"exam_type"`
	MarksObtained float64  `json:"marks_obtained" db:"marks_obtained"`
	MaxMarks      float64  `json:"max_marks" db:"max_marks"`
	LetterGrade   *string  `json:"letter_grade" db:"letter_grade"`
	GradePoint    *float64 `json:"grade_point" db:"grade_point"`
}
