package safefs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathTraversal    = errors.New("path traversal detected")
	ErrPathOutsideBase  = errors.New("path outside allowed base directory")
	ErrFileSizeExceeded = errors.New("file size exceeds limit")
	ErrInvalidExtension = errors.New("file extension not allowed")
)

type SafeFS struct {
	basePaths          []string
	maxFileSize        int64
	allowedExtensions  []string
	deniedPaths        []string
	auditLogger        *AuditLogger
}

type AuditLogger struct {
	writer io.Writer
}

func NewSafeFS(basePaths []string, maxSize int64, extensions []string) *SafeFS {
	return &SafeFS{
		basePaths:         basePaths,
		maxFileSize:       maxSize,
		allowedExtensions: extensions,
		deniedPaths:       []string{"/etc", "/root", "/var/log"},
		auditLogger:       &AuditLogger{writer: os.Stdout},
	}
}

func (fs *SafeFS) ValidatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return ErrPathTraversal
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return err
	}

	// Check if path is in denied list
	for _, denied := range fs.deniedPaths {
		if strings.HasPrefix(absPath, denied) {
			return fmt.Errorf("access denied to path: %s", denied)
		}
	}

	// Check if path is within allowed base paths
	allowed := false
	for _, base := range fs.basePaths {
		absBase, _ := filepath.Abs(base)
		if strings.HasPrefix(absPath, absBase) {
			allowed = true
			break
		}
	}

	if !allowed && len(fs.basePaths) > 0 {
		return ErrPathOutsideBase
	}

	return nil
}

func (fs *SafeFS) ReadFile(path string) ([]byte, error) {
	if err := fs.ValidatePath(path); err != nil {
		fs.auditLogger.LogOperation("read", path, "denied", err.Error())
		return nil, err
	}

	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() > fs.maxFileSize {
		fs.auditLogger.LogOperation("read", path, "denied", "file too large")
		return nil, ErrFileSizeExceeded
	}

	// Check extension if restrictions are set
	if len(fs.allowedExtensions) > 0 {
		ext := filepath.Ext(path)
		allowed := false
		for _, allowedExt := range fs.allowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			fs.auditLogger.LogOperation("read", path, "denied", "invalid extension")
			return nil, ErrInvalidExtension
		}
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		fs.auditLogger.LogOperation("read", path, "failed", err.Error())
		return nil, err
	}

	fs.auditLogger.LogOperation("read", path, "success", fmt.Sprintf("%d bytes", len(content)))
	return content, nil
}

func (fs *SafeFS) ListDirectory(path string) ([]os.DirEntry, error) {
	if err := fs.ValidatePath(path); err != nil {
		fs.auditLogger.LogOperation("list", path, "denied", err.Error())
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		fs.auditLogger.LogOperation("list", path, "failed", err.Error())
		return nil, err
	}

	fs.auditLogger.LogOperation("list", path, "success", fmt.Sprintf("%d entries", len(entries)))
	return entries, nil
}

func (al *AuditLogger) LogOperation(operation, path, result, details string) {
	fmt.Fprintf(al.writer, "[AUDIT] op=%s path=%s result=%s details=%s\n", 
		operation, path, result, details)
}