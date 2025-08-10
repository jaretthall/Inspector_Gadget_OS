package safefs

import (
	"path/filepath"
	"testing"
)

// MockAuditLogger for testing
type MockAuditLogger struct {
	logs []AuditEntry
}

type AuditEntry struct {
	Operation string
	Path      string
	User      string
	Success   bool
	Details   string
}

func (m *MockAuditLogger) LogFileOperation(operation, path, user string, success bool, details string) {
	m.logs = append(m.logs, AuditEntry{
		Operation: operation,
		Path:      path,
		User:      user,
		Success:   success,
		Details:   details,
	})
}

func TestPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	config := Config{
		BasePaths:   []string{tempDir},
		MaxFileSize: 1024 * 1024, // 1MB
		AllowedExts: []string{".txt", ".md", ".json"},
		DeniedPaths: []string{filepath.Join(tempDir, "restricted")},
	}
	
	fs := NewSafeFS(config)
	
	tests := []struct {
		name        string
		path        string
		expectError error
	}{
		{
			name:        "Valid path",
			path:        filepath.Join(tempDir, "test.txt"),
			expectError: nil,
		},
		{
			name:        "Path traversal attempt",
			path:        filepath.Join(tempDir, "../etc/passwd"),
			expectError: ErrPathOutsideBase, // This will resolve outside base path
		},
		{
			name:        "Outside base path",
			path:        "/etc/passwd",
			expectError: ErrPathOutsideBase,
		},
		{
			name:        "Disallowed extension",
			path:        filepath.Join(tempDir, "test.exe"),
			expectError: ErrExtNotAllowed,
		},
		{
			name:        "Denied path",
			path:        filepath.Join(tempDir, "restricted", "file.txt"),
			expectError: ErrPathDenied,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.ValidatePath(tt.path)
			if tt.expectError == nil && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if tt.expectError != nil && err != tt.expectError {
				t.Errorf("Expected error %v, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	auditLogger := &MockAuditLogger{}
	
	config := Config{
		BasePaths:   []string{tempDir},
		MaxFileSize: 1024,
		AllowedExts: []string{".txt"},
		AuditLogger: auditLogger,
	}
	
	fs := NewSafeFS(config)
	
	testFile := filepath.Join(tempDir, "test.txt")
	testData := []byte("Hello, SafeFS!")
	
	// Test write operation
	err := fs.WriteFile(testFile, "testuser", testData, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	
	// Test read operation
	readData, err := fs.ReadFile(testFile, "testuser")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	
	if string(readData) != string(testData) {
		t.Errorf("Read data doesn't match written data")
	}
	
	// Test list directory
	fileInfos, err := fs.ListDir(tempDir, "testuser")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	
	if len(fileInfos) != 1 {
		t.Errorf("Expected 1 file, got %d", len(fileInfos))
	}
	
	// Test copy operation
	copyFile := filepath.Join(tempDir, "copy.txt")
	err = fs.CopyFile(testFile, copyFile, "testuser")
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}
	
	// Verify audit logs
	if len(auditLogger.logs) < 4 {
		t.Errorf("Expected at least 4 audit log entries, got %d", len(auditLogger.logs))
	}
}

func TestSizeLimits(t *testing.T) {
	tempDir := t.TempDir()
	
	config := Config{
		BasePaths:   []string{tempDir},
		MaxFileSize: 10, // Very small limit for testing
		AllowedExts: []string{".txt"},
	}
	
	fs := NewSafeFS(config)
	
	testFile := filepath.Join(tempDir, "big.txt")
	bigData := make([]byte, 20) // Exceeds limit
	
	// Should fail due to size limit
	err := fs.WriteFile(testFile, "testuser", bigData, 0644)
	if err != ErrFileTooBig {
		t.Errorf("Expected ErrFileTooBig, got: %v", err)
	}
}

func TestHiddenPathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	
	config := Config{
		BasePaths:   []string{tempDir},
		MaxFileSize: 1024,
		AllowedExts: []string{".txt"},
	}
	
	fs := NewSafeFS(config)
	
	// Various path traversal attempts
	traversalPaths := []string{
		filepath.Join(tempDir, "..", "etc", "passwd"),
		filepath.Join(tempDir, "subdir", "..", "..", "etc", "passwd"),
		filepath.Join(tempDir, "....//....//etc//passwd"),
	}
	
	for _, path := range traversalPaths {
		err := fs.ValidatePath(path)
		if err == nil {
			t.Errorf("Path traversal should have been blocked: %s", path)
		}
	}
}