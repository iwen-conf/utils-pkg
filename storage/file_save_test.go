package storage

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
)

// TestHandleFileUpload 测试单文件上传功能
func TestHandleFileUpload(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "file_upload_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(testDir) // 测试结束后清理

	// 测试数据
	testCases := []struct {
		name           string
		fileContent    string
		fileName       string
		formFieldName  string
		options        FileUploadOptions
		expectedError  bool
		expectedUpload bool
	}{
		{
			name:           "基本文件上传",
			fileContent:    "测试文件内容",
			fileName:       "test.txt",
			formFieldName:  "file",
			options:        DefaultFileUploadOptions(),
			expectedError:  false,
			expectedUpload: true,
		},
		{
			name:          "文件大小超限",
			fileContent:   strings.Repeat("大文件测试", 1000), // 创建较大的文件内容
			fileName:      "large.txt",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        10, // 设置一个很小的限制
				AllowedFileTypes:   []string{},
				GenerateUniqueName: false,
				PreserveExtension:  true,
			},
			expectedError:  true,
			expectedUpload: false,
		},
		{
			name:          "文件类型限制",
			fileContent:   "图片内容模拟",
			fileName:      "test.jpg",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{"image/jpeg"}, // 只允许jpeg
				GenerateUniqueName: false,
				PreserveExtension:  true,
			},
			expectedError:  true, // 由于我们无法真正设置Content-Type为image/jpeg，预期会失败
			expectedUpload: false,
		},
		{
			name:          "生成唯一文件名",
			fileContent:   "唯一文件名测试",
			fileName:      "unique.txt",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{},
				GenerateUniqueName: true,
				PreserveExtension:  true,
			},
			expectedError:  false,
			expectedUpload: true,
		},
		{
			name:          "子路径测试",
			fileContent:   "子路径测试内容",
			fileName:      "subpath.txt",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{},
				GenerateUniqueName: false,
				PreserveExtension:  true,
				SubPath:            "subdir/nested",
			},
			expectedError:  false,
			expectedUpload: true,
		},
		{
			name:          "绝对路径测试",
			fileContent:   "绝对路径测试内容",
			fileName:      "absolute.txt",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{},
				GenerateUniqueName: false,
				PreserveExtension:  true,
				UseAbsolutePath:    true,
			},
			expectedError:  false,
			expectedUpload: true,
		},
		{
			name:          "绝对路径带子路径测试",
			fileContent:   "绝对路径带子路径测试内容",
			fileName:      "absolute_subpath.txt",
			formFieldName: "file",
			options: FileUploadOptions{
				MaxFileSize:        1024 * 1024,
				AllowedFileTypes:   []string{},
				GenerateUniqueName: false,
				PreserveExtension:  true,
				SubPath:            "absolute/path",
				UseAbsolutePath:    true,
			},
			expectedError:  false,
			expectedUpload: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 直接验证文件上传结果，不使用HTTP请求
			uploadDir := filepath.Join(testDir, tc.name)
			ctx := createTestContext(t, tc.formFieldName, tc.fileName, tc.fileContent)

			result := HandleFileUploadWithOptions(ctx, tc.formFieldName, uploadDir, tc.options)

			// 检查上传结果
			assert.DeepEqual(t, tc.expectedUpload, result.Uploaded)

			if tc.expectedError {
				assert.NotNil(t, result.Error)
			} else {
				assert.Nil(t, result.Error)

				// 检查文件是否存在
				if tc.options.SubPath != "" {
					uploadDir = filepath.Join(uploadDir, tc.options.SubPath)
				}

				var filePath string
				if tc.options.GenerateUniqueName {
					// 如果生成唯一文件名，我们只能检查目录中是否有文件
					files, err := os.ReadDir(uploadDir)
					assert.Nil(t, err)
					assert.DeepEqual(t, true, len(files) > 0)
				} else {
					filePath = filepath.Join(uploadDir, tc.fileName)
					assert.DeepEqual(t, true, FileExists(filePath))

					// 检查文件内容
					content, err := os.ReadFile(filePath)
					assert.Nil(t, err)
					assert.DeepEqual(t, tc.fileContent, string(content))
				}

				// 检查返回的路径
				if tc.options.UseAbsolutePath {
					// 验证返回的是绝对路径
					assert.DeepEqual(t, true, filepath.IsAbs(result.FilePath))
					// 验证路径指向正确的文件
					assert.DeepEqual(t, true, FileExists(result.FilePath))
				} else {
					// 验证返回的是相对路径
					assert.DeepEqual(t, false, filepath.IsAbs(result.FilePath))
					// 验证相对路径指向正确的文件
					absPath := filepath.Join(testDir, tc.name, result.FilePath)
					assert.DeepEqual(t, true, FileExists(absPath))
				}
			}
		})
	}
}

// TestHandleMultiFileUpload 测试多文件上传功能
func TestHandleMultiFileUpload(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "multi_file_upload_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(testDir) // 测试结束后清理

	// 测试用例
	t.Run("多文件上传基本功能", func(t *testing.T) {
		// 创建测试上下文
		formFieldName := "files"
		uploadDir := filepath.Join(testDir, "multi")

		// 创建测试文件数据
		fileContents := []string{
			"第一个文件内容",
			"第二个文件内容",
			"第三个文件内容",
		}
		fileNames := []string{
			"file1.txt",
			"file2.txt",
			"file3.txt",
		}

		// 创建测试上下文
		ctx := createMultiTestContext(t, formFieldName, fileNames, fileContents)

		// 执行多文件上传
		options := DefaultFileUploadOptions()
		result := HandleMultiFileUpload(ctx, formFieldName, uploadDir, options)

		// 验证结果
		assert.DeepEqual(t, 3, result.SuccessCount)
		assert.DeepEqual(t, 0, result.FailCount)
		assert.DeepEqual(t, 3, len(result.Files))

		// 检查每个文件是否成功上传
		for i, fileResult := range result.Files {
			assert.DeepEqual(t, true, fileResult.Uploaded)
			assert.Nil(t, fileResult.Error)
			assert.DeepEqual(t, fileNames[i], fileResult.FileName)

			// 检查文件内容
			content, err := os.ReadFile(fileResult.FilePath)
			assert.Nil(t, err)
			assert.DeepEqual(t, fileContents[i], string(content))
		}
	})

	t.Run("文件总大小超限", func(t *testing.T) {
		// 创建测试上下文
		formFieldName := "files"
		uploadDir := filepath.Join(testDir, "size_limit")

		// 创建测试文件数据 - 较大的文件
		fileContents := []string{
			strings.Repeat("大文件1", 1000),
			strings.Repeat("大文件2", 1000),
			strings.Repeat("大文件3", 1000),
		}
		fileNames := []string{
			"large1.txt",
			"large2.txt",
			"large3.txt",
		}

		// 创建测试上下文
		ctx := createMultiTestContext(t, formFieldName, fileNames, fileContents)

		// 设置一个很小的总大小限制
		options := DefaultFileUploadOptions()
		options.MaxTotalSize = 100 // 100字节的总大小限制

		// 执行多文件上传
		result := HandleMultiFileUpload(ctx, formFieldName, uploadDir, options)

		// 验证结果 - 应该因为总大小超限而失败
		assert.DeepEqual(t, 0, result.SuccessCount)
		assert.DeepEqual(t, 1, result.FailCount) // 只有一个错误记录，表示总大小超限
		assert.NotNil(t, result.Files[0].Error)
	})

	t.Run("子路径测试", func(t *testing.T) {
		// 创建测试上下文
		formFieldName := "files"
		uploadDir := filepath.Join(testDir, "subpath_test")

		// 创建测试文件数据
		fileContents := []string{"子路径文件内容"}
		fileNames := []string{"subpath_file.txt"}

		// 创建测试上下文
		ctx := createMultiTestContext(t, formFieldName, fileNames, fileContents)

		// 设置子路径
		options := DefaultFileUploadOptions()
		options.SubPath = "custom/path"

		// 执行多文件上传
		result := HandleMultiFileUpload(ctx, formFieldName, uploadDir, options)

		// 验证结果
		assert.DeepEqual(t, 1, result.SuccessCount)
		assert.DeepEqual(t, 0, result.FailCount)

		// 检查文件是否在子路径中
		expectedPath := filepath.Join(uploadDir, "custom/path", fileNames[0])
		assert.DeepEqual(t, true, FileExists(expectedPath))
	})

	t.Run("绝对路径测试", func(t *testing.T) {
		// 创建测试上下文
		formFieldName := "files"
		uploadDir := filepath.Join(testDir, "absolute_path_test")

		// 创建测试文件数据
		fileContents := []string{"绝对路径测试内容"}
		fileNames := []string{"absolute_test.txt"}

		// 创建测试上下文
		ctx := createMultiTestContext(t, formFieldName, fileNames, fileContents)

		// 设置绝对路径选项
		options := DefaultFileUploadOptions()
		options.UseAbsolutePath = true

		// 执行多文件上传
		result := HandleMultiFileUpload(ctx, formFieldName, uploadDir, options)

		// 验证结果
		assert.DeepEqual(t, 1, result.SuccessCount)
		assert.DeepEqual(t, 0, result.FailCount)

		// 检查返回的路径是否为绝对路径
		assert.DeepEqual(t, true, filepath.IsAbs(result.Files[0].FilePath))
		// 验证路径指向正确的文件
		assert.DeepEqual(t, true, FileExists(result.Files[0].FilePath))
	})

	t.Run("绝对路径带子路径测试", func(t *testing.T) {
		// 创建测试上下文
		formFieldName := "files"
		uploadDir := filepath.Join(testDir, "absolute_subpath_test")

		// 创建测试文件数据
		fileContents := []string{"绝对路径带子路径测试内容"}
		fileNames := []string{"absolute_subpath_test.txt"}

		// 创建测试上下文
		ctx := createMultiTestContext(t, formFieldName, fileNames, fileContents)

		// 设置绝对路径和子路径选项
		options := DefaultFileUploadOptions()
		options.UseAbsolutePath = true
		options.SubPath = "custom/absolute/path"

		// 执行多文件上传
		result := HandleMultiFileUpload(ctx, formFieldName, uploadDir, options)

		// 验证结果
		assert.DeepEqual(t, 1, result.SuccessCount)
		assert.DeepEqual(t, 0, result.FailCount)

		// 检查返回的路径是否为绝对路径
		assert.DeepEqual(t, true, filepath.IsAbs(result.Files[0].FilePath))
		// 验证路径指向正确的文件
		assert.DeepEqual(t, true, FileExists(result.Files[0].FilePath))
		// 验证路径包含子路径
		assert.DeepEqual(t, true, strings.Contains(result.Files[0].FilePath, "custom/absolute/path"))
	})
}

// 辅助函数：创建测试上下文
func createTestContext(t *testing.T, formFieldName, fileName, fileContent string) *app.RequestContext {
	// 创建一个新的请求上下文
	ctx := app.NewContext(0)

	// 创建一个buffer来存储multipart表单数据
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建文件部分
	part, err := writer.CreateFormFile(formFieldName, fileName)
	if err != nil {
		t.Fatalf("创建表单文件失败: %v", err)
	}

	// 写入文件内容
	if _, err := part.Write([]byte(fileContent)); err != nil {
		t.Fatalf("写入文件内容失败: %v", err)
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭writer失败: %v", err)
	}

	// 设置请求头
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// 设置请求体
	ctx.Request.SetBody(body.Bytes())

	return ctx
}

// 辅助函数：创建多文件测试上下文
func createMultiTestContext(t *testing.T, formFieldName string, fileNames []string, fileContents []string) *app.RequestContext {
	// 创建一个新的请求上下文
	ctx := app.NewContext(0)

	// 创建一个buffer来存储multipart表单数据
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加每个文件
	for i, name := range fileNames {
		// 创建文件部分
		part, err := writer.CreateFormFile(formFieldName, name)
		if err != nil {
			t.Fatalf("创建表单文件失败: %v", err)
		}

		// 写入文件内容
		if _, err := part.Write([]byte(fileContents[i])); err != nil {
			t.Fatalf("写入文件内容失败: %v", err)
		}
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭writer失败: %v", err)
	}

	// 设置请求头
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// 设置请求体
	ctx.Request.SetBody(body.Bytes())

	return ctx
}

// 测试辅助工具函数
func TestGetSafeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file:with:colons.txt", "file_with_colons.txt"},
		{"file*with*stars.txt", "file_with_stars.txt"},
		{"file?with?questions.txt", "file_with_questions.txt"},
		{"file\"with\"quotes.txt", "file_with_quotes.txt"},
		{"file<with>brackets.txt", "file_with_brackets.txt"},
		{"file|with|pipes.txt", "file_with_pipes.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := GetSafeFilename(tc.input)
			assert.DeepEqual(t, tc.expected, result)
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"file.txt", ".txt"},
		{"file.tar.gz", ".gz"},
		{"file", ""},
		{".hidden", ".hidden"},
		{"path/to/file.jpg", ".jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := GetFileExtension(tc.input)
			assert.DeepEqual(t, tc.expected, result)
		})
	}
}

func TestIsImageFile(t *testing.T) {
	testCases := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"text/plain", false},
		{"application/pdf", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			result := IsImageFile(tc.contentType)
			assert.DeepEqual(t, tc.expected, result)
		})
	}
}

func TestGetFormattedFileSize(t *testing.T) {
	testCases := []struct {
		size     int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1500, "1.46 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d", tc.size), func(t *testing.T) {
			result := GetFormattedFileSize(tc.size)
			assert.DeepEqual(t, tc.expected, result)
		})
	}
}

func TestFileExists(t *testing.T) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "test_file_exists")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempFilePath)

	// 测试存在的文件
	assert.DeepEqual(t, true, FileExists(tempFilePath))

	// 测试不存在的文件
	assert.DeepEqual(t, false, FileExists(tempFilePath+".nonexistent"))
}

func TestGenerateUniqueFilename(t *testing.T) {
	// 测试生成的唯一文件名不重复
	filename1 := generateUniqueFilename("test.txt")
	filename2 := generateUniqueFilename("test.txt")

	assert.NotEqual(t, filename1, filename2)
	assert.DeepEqual(t, 32, len(filename1)) // MD5哈希长度为32
}
