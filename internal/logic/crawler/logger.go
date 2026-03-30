package crawler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SimpleLogger 简单日志记录器
type SimpleLogger struct {
	file      *os.File
	mu        sync.Mutex
	logDir    string
	date      string
	maxDays   int
}

// globalLogger 全局日志实例
var globalLogger *SimpleLogger
var loggerOnce sync.Once

// GetLogger 获取全局日志实例
func GetLogger() *SimpleLogger {
	loggerOnce.Do(func() {
		globalLogger = &SimpleLogger{
			logDir:  "logs",
			date:    time.Now().Format("20060102"),
			maxDays: 7,
		}
		if err := globalLogger.init(); err != nil {
			fmt.Printf("❌ 初始化日志失败：%v\n", err)
		}
	})
	return globalLogger
}

// init 初始化日志文件
func (l *SimpleLogger) init() error {
	// 创建日志目录
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败：%w", err)
	}

	// 打开日志文件
	return l.rotate()
}

// rotate 检查是否需要轮转日志文件
func (l *SimpleLogger) rotate() error {
	today := time.Now().Format("20060102")
	
	// 如果是新的一天，关闭旧文件
	if l.date != today && l.file != nil {
		l.file.Close()
		l.file = nil
		l.date = today
	}

	// 如果文件未打开，打开新文件
	if l.file == nil {
		filename := filepath.Join(l.logDir, fmt.Sprintf("crawler-%s.log", today))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败：%w", err)
		}
		l.file = file
	}

	// 清理旧日志
	l.cleanOldLogs()

	return nil
}

// cleanOldLogs 清理超过 maxDays 天的日志
func (l *SimpleLogger) cleanOldLogs() {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	threshold := time.Now().AddDate(0, 0, -l.maxDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(threshold) {
			os.Remove(filepath.Join(l.logDir, entry.Name()))
		}
	}
}

// write 写入日志
func (l *SimpleLogger) write(level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否需要轮转
	if err := l.rotate(); err != nil {
		fmt.Printf("❌ 日志轮转失败：%v\n", err)
		return
	}

	// 格式化日志内容
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	content := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))

	// 写入文件
	if l.file != nil {
		l.file.WriteString(content)
	}

	// 输出到控制台
	fmt.Print(content)
}

// Info 记录 INFO 级别日志
func (l *SimpleLogger) Info(ctx context.Context, format string, args ...interface{}) {
	l.write("INFO", format, args...)
}

// Warn 记录 WARNING 级别日志
func (l *SimpleLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	l.write("WARNING", format, args...)
}

// Error 记录 ERROR 级别日志
func (l *SimpleLogger) Error(ctx context.Context, format string, args ...interface{}) {
	l.write("ERROR", format, args...)
}

// Debug 记录 DEBUG 级别日志
func (l *SimpleLogger) Debug(ctx context.Context, format string, args ...interface{}) {
	l.write("DEBUG", format, args...)
}

// Close 关闭日志文件
func (l *SimpleLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// 日志辅助函数

func logInfo(ctx context.Context, taskId, msg string) {
	logger := GetLogger()
	logger.Info(ctx, "📝 [%s] %s", taskId, msg)
}

func logWarn(ctx context.Context, taskId, msg string) {
	logger := GetLogger()
	logger.Warn(ctx, "⚠️ [%s] %s", taskId, msg)
}

func logError(ctx context.Context, taskId, msg string) {
	logger := GetLogger()
	logger.Error(ctx, "❌ [%s] %s", taskId, msg)
}

func logDebug(ctx context.Context, taskId, msg string) {
	logger := GetLogger()
	logger.Debug(ctx, "🔍 [%s] %s", taskId, msg)
}

func logStep(ctx context.Context, taskId string, step int, msg string) {
	logger := GetLogger()
	logger.Info(ctx, "📌 [%s] 步骤%d: %s", taskId, step, msg)
}

func logSuccess(ctx context.Context, taskId, msg string) {
	logger := GetLogger()
	logger.Info(ctx, "✅ [%s] %s", taskId, msg)
}

func logProgress(ctx context.Context, taskId string, current, total int, msg string) {
	logger := GetLogger()
	logger.Info(ctx, "📊 [%s] 进度：%d/%d - %s", taskId, current, total, msg)
}
