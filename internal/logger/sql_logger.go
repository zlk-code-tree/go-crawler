package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SQLLogger SQL 日志记录器
type SQLLogger struct {
	file   *os.File
	mu     sync.Mutex
	logDir string
	date   string
}

var sqlLogger *SQLLogger
var sqlLoggerOnce sync.Once

// GetSQLLogger 获取全局 SQL 日志实例
func GetSQLLogger() *SQLLogger {
	sqlLoggerOnce.Do(func() {
		sqlLogger = &SQLLogger{
			logDir: "logs/sql",
			date:   time.Now().Format("20060102"),
		}
		if err := sqlLogger.init(); err != nil {
			fmt.Printf("❌ 初始化 SQL 日志失败：%v\n", err)
		}
	})
	return sqlLogger
}

// init 初始化 SQL 日志文件
func (l *SQLLogger) init() error {
	// 创建日志目录
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("创建 SQL 日志目录失败：%w", err)
	}

	// 打开日志文件
	return l.rotate()
}

// rotate 检查是否需要轮转日志文件
func (l *SQLLogger) rotate() error {
	today := time.Now().Format("20060102")

	// 如果是新的一天，关闭旧文件
	if l.date != today && l.file != nil {
		l.file.Close()
		l.file = nil
		l.date = today
	}

	// 如果文件未打开，打开新文件
	if l.file == nil {
		filename := filepath.Join(l.logDir, fmt.Sprintf("sql-%s.log", today))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("打开 SQL 日志文件失败：%w", err)
		}
		l.file = file
	}

	return nil
}

// write 写入 SQL 日志
func (l *SQLLogger) write(sqlType, sql string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否需要轮转
	if err := l.rotate(); err != nil {
		fmt.Printf("❌ SQL 日志轮转失败：%v\n", err)
		return
	}

	// 格式化日志内容
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	content := fmt.Sprintf("[%s] [%s] %s\n", timestamp, sqlType, fmt.Sprintf(sql, args...))

	// 写入文件
	if l.file != nil {
		l.file.WriteString(content)
	}

	// 输出到控制台
	fmt.Print(content)
}

// Info 记录 SQL 执行信息
func (l *SQLLogger) Info(sql string, args ...interface{}) {
	l.write("INFO", sql, args...)
}

// Error 记录 SQL 错误
func (l *SQLLogger) Error(err error, sql string, args ...interface{}) {
	l.write("ERROR", "错误：%v | SQL: %s", append([]interface{}{err, sql}, args...)...)
}

// Close 关闭日志文件
func (l *SQLLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
