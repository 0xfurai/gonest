package files

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gonest"
)

// FileRecord tracks uploaded files.
type FileRecord struct {
	ID           int       `json:"id" swagger:"example=1"`
	Filename     string    `json:"filename" swagger:"example=a1b2c3d4.jpg"`
	OriginalName string    `json:"originalName" swagger:"example=photo.jpg"`
	MimeType     string    `json:"mimeType" swagger:"example=image/jpeg"`
	Size         int64     `json:"size" swagger:"example=102400"`
	URL          string    `json:"url" swagger:"example=/uploads/a1b2c3d4.jpg"`
	UploaderID   int       `json:"uploaderId" swagger:"example=1"`
	CreatedAt    time.Time `json:"createdAt" swagger:"format=date-time"`
}

// FilesService manages file upload and storage backed by SQL.
type FilesService struct {
	db        *sql.DB
	uploadDir string
}

func NewFilesService(db *sql.DB) *FilesService {
	dir := "./uploads"
	os.MkdirAll(dir, 0755)
	return &FilesService{db: db, uploadDir: dir}
}

func (s *FilesService) Upload(uploaderID int, file *gonest.UploadedFile) (*FileRecord, error) {
	if file == nil {
		return nil, gonest.NewBadRequestException("no file provided")
	}

	allowed := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true,
		"image/webp": true, "application/pdf": true,
	}
	if !allowed[file.MimeType] {
		return nil, gonest.NewBadRequestException("file type not allowed: " + file.MimeType)
	}
	if file.Size > 5<<20 {
		return nil, gonest.NewPayloadTooLargeException("file exceeds 5MB limit")
	}

	ext := filepath.Ext(file.Filename)
	uniqueName := generateFilename() + ext

	data, err := gonest.ReadFileBytes(file)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to read file: " + err.Error())
	}
	if err := os.WriteFile(filepath.Join(s.uploadDir, uniqueName), data, 0644); err != nil {
		return nil, gonest.NewInternalServerError("failed to save file: " + err.Error())
	}

	now := time.Now()
	url := "/uploads/" + uniqueName

	result, err := s.db.Exec(
		`INSERT INTO files (filename, original_name, mime_type, size, url, uploader_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uniqueName, file.Filename, file.MimeType, file.Size, url, uploaderID, now,
	)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to save file record: " + err.Error())
	}
	id, _ := result.LastInsertId()
	return s.FindByID(int(id)), nil
}

func (s *FilesService) FindByID(id int) *FileRecord {
	r := &FileRecord{}
	err := s.db.QueryRow(
		`SELECT id, filename, original_name, mime_type, size, url, uploader_id, created_at
		 FROM files WHERE id = ?`, id,
	).Scan(&r.ID, &r.Filename, &r.OriginalName, &r.MimeType, &r.Size, &r.URL, &r.UploaderID, &r.CreatedAt)
	if err != nil {
		return nil
	}
	return r
}

func (s *FilesService) FindAll() []*FileRecord {
	rows, err := s.db.Query(
		`SELECT id, filename, original_name, mime_type, size, url, uploader_id, created_at
		 FROM files ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var files []*FileRecord
	for rows.Next() {
		r := &FileRecord{}
		if err := rows.Scan(&r.ID, &r.Filename, &r.OriginalName, &r.MimeType, &r.Size, &r.URL, &r.UploaderID, &r.CreatedAt); err == nil {
			files = append(files, r)
		}
	}
	return files
}

func (s *FilesService) Delete(id int) error {
	f := s.FindByID(id)
	if f == nil {
		return gonest.NewNotFoundException("file not found")
	}
	os.Remove(filepath.Join(s.uploadDir, f.Filename))
	s.db.Exec("DELETE FROM files WHERE id = ?", id)
	return nil
}

func generateFilename() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func ValidateImageType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
