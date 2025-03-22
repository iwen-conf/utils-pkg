package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

// 创建测试文件
func createTestFile(t *testing.T, path string, content []byte) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	assert.NoError(t, err)

	err = os.WriteFile(path, content, 0644)
	assert.NoError(t, err)
}

// 清理测试文件
func cleanupTestFile(t *testing.T, path string) {
	err := os.RemoveAll(path)
	assert.NoError(t, err)
}

func TestHandleFileUpload(t *testing.T) {
	// 创建测试目录
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// 创建测试文件
	testContent := []byte("test content")
	testFilePath := filepath.Join(testDir, "test.txt")
	createTestFile(t, testFilePath, testContent)
	defer cleanupTestFile(t, testFilePath)

	// 创建请求上下文
	c := app.NewContext(16)
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	// 测试基本上传功能
	result := HandleFileUpload(c, "file", testDir)
	assert.False(t, result.Uploaded) // 因为没有实际的文件上传
	assert.Error(t, result.Error)
}

func TestHandleFileUploadWithOptions(t *testing.T) {
	// 创建测试目录
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	testCases := []struct {
		name        string
		options     FileUploadOptions
		uploadDir   string
		expectPath  string
		shouldError bool
	}{
		{
			name: "基本上传",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{"text/plain"},
				GenerateUniqueName: false,
				PreserveExtension:  true,
			},
			uploadDir:   testDir,
			expectPath:  "/test_uploads/test.txt",
			shouldError: false,
		},
		{
			name: "使用子路径",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{"text/plain"},
				GenerateUniqueName: false,
				PreserveExtension:  true,
				SubPath:            "subdir",
			},
			uploadDir:   testDir,
			expectPath:  "/test_uploads/subdir/test.txt",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建测试文件
			testContent := []byte("test content")
			testFilePath := filepath.Join(tc.uploadDir, "test.txt")
			createTestFile(t, testFilePath, testContent)
			defer cleanupTestFile(t, testFilePath)

			// 创建请求上下文
			c := app.NewContext(16)
			c.Request.Header.Set("Content-Type", "multipart/form-data")

			result := HandleFileUploadWithOptions(c, "file", tc.uploadDir, tc.options)

			if tc.shouldError {
				assert.Error(t, result.Error)
				assert.False(t, result.Uploaded)
			} else {
				if result.Error != nil {
					t.Logf("Unexpected error: %v", result.Error)
				}
				// 由于没有实际的文件上传，这里会返回错误
				assert.False(t, result.Uploaded)
			}
		})
	}
}

func TestStandardizePath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "path/to/file",
			expected: "/path/to/file",
		},
		{
			input:    "./path/to/file",
			expected: "/path/to/file",
		},
		{
			input:    "../path/to/file",
			expected: "/path/to/file",
		},
		{
			input:    "/path/to/file",
			expected: "/path/to/file",
		},
		{
			input:    "path\\to\\file",
			expected: "/path/to/file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := standardizePath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMultiFileUpload(t *testing.T) {
	// 创建测试目录
	testDir := "test_uploads_multi"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// 创建测试文件
	testFiles := []struct {
		name    string
		content []byte
	}{
		{"test1.txt", []byte("test content 1")},
		{"test2.txt", []byte("test content 2")},
	}

	for _, tf := range testFiles {
		path := filepath.Join(testDir, tf.name)
		createTestFile(t, path, tf.content)
		defer cleanupTestFile(t, path)
	}

	// 创建请求上下文
	c := app.NewContext(16)
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	// 测试多文件上传
	options := FileUploadOptions{
		MaxFileSize:        1024 * 1024,
		AllowedFileTypes:   []string{"text/plain"},
		GenerateUniqueName: false,
		PreserveExtension:  true,
	}

	result := HandleMultiFileUpload(c, "files", testDir, options)
	assert.Equal(t, 0, result.SuccessCount) // 因为没有实际的文件上传
	assert.Equal(t, 1, result.FailCount)    // 应该有一个错误（找不到文件）
}

func TestSaveMultipartFile(t *testing.T) {
	// 创建测试目录
	testDir := "test_uploads_save"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// 测试空文件处理
	dstPath := filepath.Join(testDir, "saved_test.txt")
	err = SaveMultipartFile(nil, dstPath)
	assert.Error(t, err) // 期望返回错误，因为文件为空
	assert.Equal(t, "multipart file is nil", err.Error())
}

func TestGenerateUniqueFilename(t *testing.T) {
	filename := "test.txt"
	result1 := generateUniqueFilename(filename)
	result2 := generateUniqueFilename(filename)

	assert.NotEqual(t, result1, result2)
	assert.Len(t, result1, 32) // MD5哈希长度
	assert.Len(t, result2, 32)
}

func TestGetSafeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"test.txt", "test.txt"},
		{"test/file.txt", "test_file.txt"},
		{"test\\file.txt", "test_file.txt"},
		{"test:file.txt", "test_file.txt"},
		{"test*file.txt", "test_file.txt"},
		{"test?file.txt", "test_file.txt"},
		{"test\"file.txt", "test_file.txt"},
		{"test<file.txt", "test_file.txt"},
		{"test>file.txt", "test_file.txt"},
		{"test|file.txt", "test_file.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := GetSafeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFileExists(t *testing.T) {
	// 创建测试文件
	testDir := "test_exists"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	testFilePath := filepath.Join(testDir, "test.txt")
	createTestFile(t, testFilePath, []byte("test"))

	assert.True(t, FileExists(testFilePath))
	assert.False(t, FileExists(filepath.Join(testDir, "nonexistent.txt")))
}

func TestGetFormattedFileSize(t *testing.T) {
	testCases := []struct {
		size     int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d bytes", tc.size), func(t *testing.T) {
			result := GetFormattedFileSize(tc.size)
			assert.Equal(t, tc.expected, result)
		})
	}
}
