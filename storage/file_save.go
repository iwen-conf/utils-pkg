package storage

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// UploadFileResult 包含文件上传操作的结果
// UploadFileResult contains the result of a file upload operation
type UploadFileResult struct {
	FilePath    string // 保存的文件路径
	Uploaded    bool   // 是否成功上传
	Error       error  // 错误信息
	FileName    string // 文件名
	FileSize    int64  // 文件大小（字节）
	ContentType string // 文件类型
}

// MultiUploadResult 包含多文件上传的结果
// MultiUploadResult contains results of multiple file uploads
type MultiUploadResult struct {
	Files        []UploadFileResult // 上传的文件结果列表
	TotalSize    int64              // 所有文件的总大小
	SuccessCount int                // 成功上传的文件数量
	FailCount    int                // 上传失败的文件数量
}

// FileUploadOptions 定义文件上传的选项
// FileUploadOptions defines options for file upload
type FileUploadOptions struct {
	MaxFileSize        int64    // 最大文件大小（字节），0表示不限制
	AllowedFileTypes   []string // 允许的文件类型，空表示不限制
	GenerateUniqueName bool     // 是否生成唯一文件名
	PreserveExtension  bool     // 生成唯一文件名时是否保留原文件扩展名
	SubPath            string   // 上传目录下的子路径，为空则直接使用上传目录
	MaxTotalSize       int64    // 多文件上传时的总大小限制，0表示不限制
	UseFileHash        bool     // 是否使用文件哈希作为文件名并进行去重
	HashAlgorithm      string   // 哈希算法，支持"md5"和"sha256"，默认为"sha256"
	UseAbsolutePath    bool     // 是否返回绝对路径，默认为false
}

// DefaultFileUploadOptions 返回默认的文件上传选项
// DefaultFileUploadOptions returns default file upload options
func DefaultFileUploadOptions() FileUploadOptions {
	return FileUploadOptions{
		MaxFileSize:        10 * 1024 * 1024, // 默认10MB
		AllowedFileTypes:   []string{},       // 默认不限制文件类型
		GenerateUniqueName: false,            // 默认不生成唯一文件名
		PreserveExtension:  true,             // 默认保留文件扩展名
		SubPath:            "",               // 默认不使用子路径
		MaxTotalSize:       50 * 1024 * 1024, // 默认50MB总大小限制
		UseFileHash:        false,            // 默认不使用文件哈希去重
		HashAlgorithm:      "sha256",         // 默认使用SHA-256哈希算法
		UseAbsolutePath:    false,            // 默认返回相对路径
	}
}

// HandleFileUpload 处理文件上传，包括目录创建
// HandleFileUpload handles file upload with proper directory creation
// 参数:
// - c: Hertz请求上下文
// - formFieldName: 包含文件的表单字段名
// - uploadDir: 文件保存目录
// 返回:
// - UploadFileResult 包含文件路径（如果成功）和错误（如果有）
func HandleFileUpload(c *app.RequestContext, formFieldName, uploadDir string) UploadFileResult {
	return HandleFileUploadWithOptions(c, formFieldName, uploadDir, DefaultFileUploadOptions())
}

// HandleFileUploadWithOptions 使用自定义选项处理文件上传
// HandleFileUploadWithOptions handles file upload with custom options
// 参数:
// - c: Hertz请求上下文
// - formFieldName: 包含文件的表单字段名
// - uploadDir: 文件保存目录
// - options: 自定义上传选项
// 返回:
// - UploadFileResult 包含文件路径和相关信息
func HandleFileUploadWithOptions(c *app.RequestContext, formFieldName, uploadDir string, options FileUploadOptions) UploadFileResult {
	result := UploadFileResult{
		Uploaded: false,
	}

	// 准备完整的上传路径（包括子路径）
	fullUploadDir := uploadDir
	if options.SubPath != "" {
		fullUploadDir = filepath.Join(uploadDir, options.SubPath)
	}

	// 确保目录存在
	if err := os.MkdirAll(fullUploadDir, 0755); err != nil {
		result.Error = fmt.Errorf("创建目录失败: %w", err)
		return result
	}

	// 尝试从表单获取文件
	fileHeader, err := c.FormFile(formFieldName)
	if err != nil {
		// 不将此视为错误，因为文件上传可能是可选的
		result.Error = fmt.Errorf("获取上传文件失败: %w", err)
		return result
	}

	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		result.Error = fmt.Errorf("打开上传文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 设置文件信息
	result.FileName = fileHeader.Filename
	result.FileSize = fileHeader.Size
	result.ContentType = fileHeader.Header.Get("Content-Type")

	// 检查文件大小
	if options.MaxFileSize > 0 && fileHeader.Size > options.MaxFileSize {
		result.Error = fmt.Errorf("文件过大: %d 字节, 最大允许: %d 字节", fileHeader.Size, options.MaxFileSize)
		return result
	}

	// 检查文件类型
	if len(options.AllowedFileTypes) > 0 {
		fileTypeAllowed := false
		for _, allowedType := range options.AllowedFileTypes {
			if strings.HasPrefix(result.ContentType, allowedType) {
				fileTypeAllowed = true
				break
			}
		}
		if !fileTypeAllowed {
			result.Error = fmt.Errorf("不支持的文件类型: %s", result.ContentType)
			return result
		}
	}

	// 准备文件名
	filename := fileHeader.Filename
	ext := filepath.Ext(filename)

	// 如果启用了文件哈希
	if options.UseFileHash {
		// 计算文件哈希
		hashValue, err := CalculateFileHash(file, options.HashAlgorithm)
		if err != nil {
			result.Error = fmt.Errorf("计算文件哈希失败: %w", err)
			return result
		}

		// 检查是否存在相同哈希的文件
		if exists, existingPath := CheckFileHashExists(hashValue, fullUploadDir, ext); exists {
			// 文件已存在，直接返回现有文件的信息
			result.FilePath = existingPath
			result.FileName = filepath.Base(existingPath)
			result.Uploaded = true
			return result
		}

		// 使用哈希值作为文件名
		if options.PreserveExtension {
			filename = hashValue + ext
		} else {
			filename = hashValue
		}
	} else if options.GenerateUniqueName {
		if options.PreserveExtension {
			baseFilename := strings.TrimSuffix(filename, ext)
			filename = generateUniqueFilename(baseFilename) + ext
		} else {
			filename = generateUniqueFilename(filename)
		}
	} else {
		// 确保文件名安全
		filename = GetSafeFilename(filename)
	}

	// 准备保存文件
	filePath := filepath.Join(fullUploadDir, filename)

	// 如果需要返回绝对路径，转换为绝对路径
	if options.UseAbsolutePath {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			result.Error = fmt.Errorf("获取绝对路径失败: %w", err)
			return result
		}
		filePath = absPath
	}

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		result.Error = fmt.Errorf("创建文件失败: %w", err)
		return result
	}
	defer dst.Close()

	// 复制文件内容
	if _, err = io.Copy(dst, file); err != nil {
		result.Error = fmt.Errorf("保存文件失败: %w", err)
		return result
	}

	result.FilePath = filePath
	result.FileName = filename
	result.Uploaded = true
	return result
}

// HandleMultiFileUpload 处理多文件上传
// HandleMultiFileUpload handles multiple file uploads
// 参数:
// - c: Hertz请求上下文
// - formFieldName: 包含文件的表单字段名
// - uploadDir: 文件保存目录
// - options: 自定义上传选项
// 返回:
// - MultiUploadResult 包含所有文件的上传结果
func HandleMultiFileUpload(c *app.RequestContext, formFieldName, uploadDir string, options FileUploadOptions) MultiUploadResult {
	result := MultiUploadResult{
		Files:        []UploadFileResult{},
		TotalSize:    0,
		SuccessCount: 0,
		FailCount:    0,
	}

	// 获取表单中的所有文件
	form, err := c.MultipartForm()
	if err != nil {
		// 添加一个空的结果，但带有错误信息
		result.Files = append(result.Files, UploadFileResult{
			Uploaded: false,
			Error:    fmt.Errorf("获取多文件表单失败: %w", err),
		})
		result.FailCount++
		return result
	}

	// 获取指定字段名的所有文件
	files := form.File[formFieldName]
	if len(files) == 0 {
		// 添加一个空的结果，但带有错误信息
		result.Files = append(result.Files, UploadFileResult{
			Uploaded: false,
			Error:    fmt.Errorf("表单中没有找到文件字段: %s", formFieldName),
		})
		result.FailCount++
		return result
	}

	// 计算所有文件的总大小
	var totalSize int64 = 0
	for _, fileHeader := range files {
		totalSize += fileHeader.Size
	}

	// 检查总大小限制
	if options.MaxTotalSize > 0 && totalSize > options.MaxTotalSize {
		result.Files = append(result.Files, UploadFileResult{
			Uploaded: false,
			Error:    fmt.Errorf("文件总大小过大: %d 字节, 最大允许: %d 字节", totalSize, options.MaxTotalSize),
		})
		result.FailCount++
		return result
	}

	// 准备完整的上传路径（包括子路径）
	fullUploadDir := uploadDir
	if options.SubPath != "" {
		fullUploadDir = filepath.Join(uploadDir, options.SubPath)
	}

	// 确保目录存在
	if err := os.MkdirAll(fullUploadDir, 0755); err != nil {
		result.Files = append(result.Files, UploadFileResult{
			Uploaded: false,
			Error:    fmt.Errorf("创建目录失败: %w", err),
		})
		result.FailCount++
		return result
	}

	// 处理每个文件
	for _, fileHeader := range files {
		fileResult := UploadFileResult{
			Uploaded:    false,
			FileName:    fileHeader.Filename,
			FileSize:    fileHeader.Size,
			ContentType: fileHeader.Header.Get("Content-Type"),
		}

		// 检查单个文件大小
		if options.MaxFileSize > 0 && fileHeader.Size > options.MaxFileSize {
			fileResult.Error = fmt.Errorf("文件过大: %d 字节, 最大允许: %d 字节", fileHeader.Size, options.MaxFileSize)
			result.Files = append(result.Files, fileResult)
			result.FailCount++
			continue
		}

		// 检查文件类型
		if len(options.AllowedFileTypes) > 0 {
			fileTypeAllowed := false
			for _, allowedType := range options.AllowedFileTypes {
				if strings.HasPrefix(fileResult.ContentType, allowedType) {
					fileTypeAllowed = true
					break
				}
			}
			if !fileTypeAllowed {
				fileResult.Error = fmt.Errorf("不支持的文件类型: %s", fileResult.ContentType)
				result.Files = append(result.Files, fileResult)
				result.FailCount++
				continue
			}
		}

		// 准备文件名
		filename := fileHeader.Filename
		ext := filepath.Ext(filename)

		// 如果启用了文件哈希
		if options.UseFileHash {
			// 打开文件以计算哈希
			file, err := fileHeader.Open()
			if err != nil {
				fileResult.Error = fmt.Errorf("打开上传文件失败: %w", err)
				result.Files = append(result.Files, fileResult)
				result.FailCount++
				continue
			}
			defer file.Close()

			// 计算文件哈希
			hashValue, err := CalculateFileHash(file, options.HashAlgorithm)
			if err != nil {
				fileResult.Error = fmt.Errorf("计算文件哈希失败: %w", err)
				result.Files = append(result.Files, fileResult)
				result.FailCount++
				continue
			}

			// 检查是否存在相同哈希的文件
			if exists, existingPath := CheckFileHashExists(hashValue, fullUploadDir, ext); exists {
				// 文件已存在，直接返回现有文件的信息
				fileResult.FilePath = existingPath
				fileResult.FileName = filepath.Base(existingPath)
				fileResult.Uploaded = true
				result.Files = append(result.Files, fileResult)
				result.SuccessCount++
				result.TotalSize += fileHeader.Size
				continue
			}

			// 使用哈希值作为文件名
			if options.PreserveExtension {
				filename = hashValue + ext
			} else {
				filename = hashValue
			}
		} else if options.GenerateUniqueName {
			if options.PreserveExtension {
				baseFilename := strings.TrimSuffix(filename, ext)
				filename = generateUniqueFilename(baseFilename) + ext
			} else {
				filename = generateUniqueFilename(filename)
			}
		} else {
			// 确保文件名安全
			filename = GetSafeFilename(filename)
		}

		// 准备保存文件
		filePath := filepath.Join(fullUploadDir, filename)

		// 如果需要返回绝对路径，转换为绝对路径
		if options.UseAbsolutePath {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				fileResult.Error = fmt.Errorf("获取绝对路径失败: %w", err)
				result.Files = append(result.Files, fileResult)
				result.FailCount++
				continue
			}
			filePath = absPath
		}

		// 保存文件
		if err := SaveMultipartFile(fileHeader, filePath); err != nil {
			fileResult.Error = fmt.Errorf("保存文件失败: %w", err)
			result.Files = append(result.Files, fileResult)
			result.FailCount++
			continue
		}

		// 更新文件结果
		fileResult.FilePath = filePath
		fileResult.FileName = filename
		fileResult.Uploaded = true

		// 添加到结果列表
		result.Files = append(result.Files, fileResult)
		result.SuccessCount++
		result.TotalSize += fileHeader.Size
	}

	return result
}

// SaveMultipartFile 保存multipart文件到指定路径
// SaveMultipartFile saves a multipart file to the specified path
// 参数:
// - file: multipart文件
// - dst: 目标路径
// 返回:
// - 错误（如果有）
func SaveMultipartFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	return nil
}

// generateUniqueFilename 生成唯一的文件名
// generateUniqueFilename generates a unique filename
// 参数:
// - originalName: 原始文件名
// 返回:
// - 唯一文件名
func generateUniqueFilename(originalName string) string {
	timestamp := time.Now().UnixNano()
	hash := md5.New()
	io.WriteString(hash, originalName)
	io.WriteString(hash, fmt.Sprintf("%d", timestamp))
	return hex.EncodeToString(hash.Sum(nil))
}

// CalculateFileHash 计算文件的哈希值
// CalculateFileHash calculates the hash of a file
// 参数:
// - file: 文件读取器
// - algorithm: 哈希算法 ("md5" 或 "sha256")
// 返回:
// - 文件哈希值的十六进制字符串
// - 错误（如果有）
func CalculateFileHash(file io.Reader, algorithm string) (string, error) {
	var hash string
	var err error

	// 重置文件指针到开始位置（如果支持）
	if seeker, ok := file.(io.Seeker); ok {
		if _, err = seeker.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("重置文件指针失败: %w", err)
		}
	}

	switch strings.ToLower(algorithm) {
	case "md5":
		hashMd5 := md5.New()
		if _, err = io.Copy(hashMd5, file); err != nil {
			return "", fmt.Errorf("计算MD5哈希失败: %w", err)
		}
		hash = hex.EncodeToString(hashMd5.Sum(nil))
	case "sha256", "":
		hashSha256 := sha256.New()
		if _, err = io.Copy(hashSha256, file); err != nil {
			return "", fmt.Errorf("计算SHA-256哈希失败: %w", err)
		}
		hash = hex.EncodeToString(hashSha256.Sum(nil))
	default:
		return "", fmt.Errorf("不支持的哈希算法: %s", algorithm)
	}

	// 再次重置文件指针（如果支持）
	if seeker, ok := file.(io.Seeker); ok {
		if _, err = seeker.Seek(0, io.SeekStart); err != nil {
			return hash, fmt.Errorf("重置文件指针失败（哈希值已计算）: %w", err)
		}
	}

	return hash, nil
}

// CheckFileHashExists 检查具有相同哈希值的文件是否已存在
// CheckFileHashExists checks if a file with the same hash already exists
// 参数:
// - hashValue: 文件哈希值
// - uploadDir: 上传目录
// - extension: 文件扩展名（可选）
// 返回:
// - 是否存在
// - 如果存在，返回现有文件路径
func CheckFileHashExists(hashValue, uploadDir, extension string) (bool, string) {
	filename := hashValue
	if extension != "" {
		filename = hashValue + extension
	}

	filePath := filepath.Join(uploadDir, filename)
	if FileExists(filePath) {
		return true, filePath
	}

	return false, ""
}

// GetFileExtension 获取文件扩展名
// GetFileExtension gets the file extension
// 参数:
// - filename: 文件名
// 返回:
// - 文件扩展名（包含点，如".jpg"）
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// IsImageFile 检查文件是否为图片
// IsImageFile checks if a file is an image
// 参数:
// - contentType: 文件的Content-Type
// 返回:
// - 是否为图片
func IsImageFile(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// GetSafeFilename 获取安全的文件名（移除不安全字符）
// GetSafeFilename gets a safe filename (removes unsafe characters)
// 参数:
// - filename: 原始文件名
// 返回:
// - 安全的文件名
func GetSafeFilename(filename string) string {
	// 替换不安全字符
	// Replace unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}

// GetFileInfo 获取文件信息
// GetFileInfo gets file information
// 参数:
// - filePath: 文件路径
// 返回:
// - 文件大小（字节）
// - 文件修改时间
// - 错误（如果有）
func GetFileInfo(filePath string) (int64, time.Time, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return fileInfo.Size(), fileInfo.ModTime(), nil
}

// FileExists 检查文件是否存在
// FileExists checks if a file exists
// 参数:
// - filePath: 文件路径
// 返回:
// - 文件是否存在
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetFormattedFileSize 获取格式化的文件大小（KB, MB, GB等）
// GetFormattedFileSize gets formatted file size (KB, MB, GB, etc.)
// 参数:
// - sizeInBytes: 文件大小（字节）
// 返回:
// - 格式化的文件大小字符串
func GetFormattedFileSize(sizeInBytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case sizeInBytes < KB:
		return fmt.Sprintf("%d B", sizeInBytes)
	case sizeInBytes < MB:
		return fmt.Sprintf("%.2f KB", float64(sizeInBytes)/float64(KB))
	case sizeInBytes < GB:
		return fmt.Sprintf("%.2f MB", float64(sizeInBytes)/float64(MB))
	case sizeInBytes < TB:
		return fmt.Sprintf("%.2f GB", float64(sizeInBytes)/float64(GB))
	default:
		return fmt.Sprintf("%.2f TB", float64(sizeInBytes)/float64(TB))
	}
}
