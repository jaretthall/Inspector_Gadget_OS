// Package safefs provides secure file system operations with path validation,
// size limits, and permission controls for the Inspector Gadget OS O-LLaMA component.
package safefs

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "inspector-gadget-os/o-llama/internal/logging"
)

// SafeFS provides controlled file system access with security boundaries
type SafeFS struct {
	basePaths      []string          // Allowed base paths
	maxFileSize    int64             // Maximum file size in bytes
	allowedExts    map[string]bool   // Allowed file extensions
	deniedPaths    []string          // Explicitly denied paths
	auditLogger    AuditLogger       // Audit logging interface
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogFileOperation(operation, path, user string, success bool, details string)
}

// Config holds SafeFS configuration
type Config struct {
	BasePaths      []string
	MaxFileSize    int64
	AllowedExts    []string
	DeniedPaths    []string
	AuditLogger    AuditLogger
}

// Common errors
var (
	ErrPathOutsideBase = fmt.Errorf("path outside allowed base paths")
	ErrPathTraversal   = fmt.Errorf("path traversal attempt detected")
	ErrFileTooBig      = fmt.Errorf("file exceeds maximum size limit")
	ErrExtNotAllowed   = fmt.Errorf("file extension not allowed")
	ErrPathDenied      = fmt.Errorf("path explicitly denied")
	ErrInvalidPath     = fmt.Errorf("invalid or unsafe path")
)

// NewSafeFS creates a new SafeFS instance with the given configuration
func NewSafeFS(config Config) *SafeFS {
	allowedExts := make(map[string]bool)
	for _, ext := range config.AllowedExts {
		allowedExts[strings.ToLower(ext)] = true
	}

	return &SafeFS{
		basePaths:   config.BasePaths,
		maxFileSize: config.MaxFileSize,
		allowedExts: allowedExts,
		deniedPaths: config.DeniedPaths,
		auditLogger: config.AuditLogger,
	}
}

// ValidatePath validates a file path against security policies
func (fs *SafeFS) ValidatePath(path string) error {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return ErrPathTraversal
	}
	
	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	
	// Check if path is under any allowed base path
	allowed := false
	for _, basePath := range fs.basePaths {
		absBasePath, err := filepath.Abs(basePath)
		if err != nil {
			continue
		}
		
		rel, err := filepath.Rel(absBasePath, absPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return ErrPathOutsideBase
	}
	
	// Check against explicitly denied paths
	for _, deniedPath := range fs.deniedPaths {
		absDenied, err := filepath.Abs(deniedPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(absPath, absDenied) {
			return ErrPathDenied
		}
	}
	
	// Check file extension if allowlist is configured
	if len(fs.allowedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(absPath))
		if !fs.allowedExts[ext] {
			return ErrExtNotAllowed
		}
	}
	
	return nil
}

// ReadFile safely reads a file with all security checks
func (fs *SafeFS) ReadFile(path, user string) ([]byte, error) {
	if err := fs.ValidatePath(path); err != nil {
        logging.L().Warnw("fs.read.denied", "path", path, "user", user, "reason", err.Error())
        fs.auditLog("read", path, user, false, err.Error())
		return nil, err
	}
	
	// Check file size before reading
	stat, err := os.Stat(path)
	if err != nil {
		fs.auditLog("read", path, user, false, fmt.Sprintf("stat failed: %v", err))
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
    if fs.maxFileSize > 0 && stat.Size() > fs.maxFileSize {
        logging.L().Warnw("fs.read.denied", "path", path, "user", user, "reason", "file too big", "size", stat.Size())
        fs.auditLog("read", path, user, false, fmt.Sprintf("file too big: %d bytes", stat.Size()))
		return nil, ErrFileTooBig
	}
	
	// Read the file
    data, err := os.ReadFile(path)
	if err != nil {
        logging.L().Errorw("fs.read.error", "path", path, "user", user, "error", err.Error())
        fs.auditLog("read", path, user, false, fmt.Sprintf("read failed: %v", err))
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
    logging.L().Infow("fs.read.ok", "path", path, "user", user, "size", len(data))
    fs.auditLog("read", path, user, true, fmt.Sprintf("read %d bytes", len(data)))
	return data, nil
}

// WriteFile safely writes a file with all security checks
func (fs *SafeFS) WriteFile(path, user string, data []byte, perm os.FileMode) error {
	if err := fs.ValidatePath(path); err != nil {
        logging.L().Warnw("fs.write.denied", "path", path, "user", user, "reason", err.Error())
        fs.auditLog("write", path, user, false, err.Error())
		return err
	}
	
	// Check size limit
    if fs.maxFileSize > 0 && int64(len(data)) > fs.maxFileSize {
        logging.L().Warnw("fs.write.denied", "path", path, "user", user, "reason", "data too big", "size", len(data))
        fs.auditLog("write", path, user, false, fmt.Sprintf("data too big: %d bytes", len(data)))
		return ErrFileTooBig
	}
	
	// Ensure directory exists
	dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        logging.L().Errorw("fs.write.error", "path", path, "user", user, "error", err.Error())
        fs.auditLog("write", path, user, false, fmt.Sprintf("mkdir failed: %v", err))
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write the file
    if err := os.WriteFile(path, data, perm); err != nil {
        logging.L().Errorw("fs.write.error", "path", path, "user", user, "size", len(data), "error", err.Error())
        fs.auditLog("write", path, user, false, fmt.Sprintf("write failed: %v", err))
		return fmt.Errorf("failed to write file: %w", err)
	}
	
    logging.L().Infow("fs.write.ok", "path", path, "user", user, "size", len(data))
    fs.auditLog("write", path, user, true, fmt.Sprintf("wrote %d bytes", len(data)))
	return nil
}

// ListDir safely lists directory contents with security checks
func (fs *SafeFS) ListDir(path, user string) ([]os.FileInfo, error) {
	// For directories, we need to validate the path without extension checking
    if err := fs.validatePathForDirectory(path); err != nil {
        logging.L().Warnw("fs.list.denied", "path", path, "user", user, "reason", err.Error())
        fs.auditLog("list", path, user, false, err.Error())
		return nil, err
	}
	
    entries, err := os.ReadDir(path)
	if err != nil {
        logging.L().Errorw("fs.list.error", "path", path, "user", user, "error", err.Error())
        fs.auditLog("list", path, user, false, fmt.Sprintf("readdir failed: %v", err))
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	var fileInfos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't stat
		}
		fileInfos = append(fileInfos, info)
	}
	
    logging.L().Infow("fs.list.ok", "path", path, "user", user, "count", len(fileInfos))
    fs.auditLog("list", path, user, true, fmt.Sprintf("listed %d entries", len(fileInfos)))
	return fileInfos, nil
}

// CopyFile safely copies a file with all security checks
func (fs *SafeFS) CopyFile(srcPath, dstPath, user string) error {
	// Validate both source and destination paths
	if err := fs.ValidatePath(srcPath); err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("src validation failed: %v", err))
		return fmt.Errorf("source path validation failed: %w", err)
	}
	
	if err := fs.ValidatePath(dstPath); err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("dst validation failed: %v", err))
		return fmt.Errorf("destination path validation failed: %w", err)
	}
	
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("open src failed: %v", err))
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()
	
	// Check source file size
	srcStat, err := srcFile.Stat()
	if err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("stat src failed: %v", err))
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	
	if fs.maxFileSize > 0 && srcStat.Size() > fs.maxFileSize {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("src file too big: %d bytes", srcStat.Size()))
		return ErrFileTooBig
	}
	
	// Create destination directory if needed
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("mkdir dst failed: %v", err))
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("create dst failed: %v", err))
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()
	
	// Copy file contents
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, false, fmt.Sprintf("copy failed: %v", err))
		return fmt.Errorf("failed to copy file contents: %w", err)
	}
	
	fs.auditLog("copy", fmt.Sprintf("%s -> %s", srcPath, dstPath), user, true, fmt.Sprintf("copied %d bytes", written))
	return nil
}

// validatePathForDirectory validates a path for directory operations (no extension check)
func (fs *SafeFS) validatePathForDirectory(path string) error {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return ErrPathTraversal
	}
	
	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	
	// Check if path is under any allowed base path
	allowed := false
	for _, basePath := range fs.basePaths {
		absBasePath, err := filepath.Abs(basePath)
		if err != nil {
			continue
		}
		
		rel, err := filepath.Rel(absBasePath, absPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return ErrPathOutsideBase
	}
	
	// Check against explicitly denied paths
	for _, deniedPath := range fs.deniedPaths {
		absDenied, err := filepath.Abs(deniedPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(absPath, absDenied) {
			return ErrPathDenied
		}
	}
	
	return nil
}

// auditLog logs file operations if an audit logger is configured
func (fs *SafeFS) auditLog(operation, path, user string, success bool, details string) {
	if fs.auditLogger != nil {
		fs.auditLogger.LogFileOperation(operation, path, user, success, details)
	}
}