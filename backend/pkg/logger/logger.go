package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// 调试级别
	debugLevel = iota
	// 信息级别
	infoLevel
	// 警告级别
	warnLevel
	// 错误级别
	errorLevel
)

// Logger 将应用日志按日期写入当前数据目录中的文件
type Logger struct {
	// 应用数据目录
	dataDir string
	// 级别
	level int
	// 处理操作发生时的时间，返回时间
	now func() time.Time
	// 文件模式
	fileMode os.FileMode
	// 目录模式
	dirMode os.FileMode
	// 保护共享状态的互斥锁
	mu sync.Mutex
}

// Options 配置 Logger 实例
type Options struct {
	// 数据目录
	DataDir string
	// 级别
	Level string
	// 读取当前时间，返回时间
	Now func() time.Time
	// 文件模式
	FileMode os.FileMode
	// 目录模式
	DirMode os.FileMode
}

// New 创建向 {dataDir}/logs/YYYY-MM-DD.log 写入日志的记录器
// 接收选项列表，返回日志记录器及可能发生的错误
func New(options Options) (*Logger, error) {
	level, err := parseLevel(options.Level) // 级别、解析级别时的错误
	if err != nil {
		return nil, err
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.FileMode == 0 {
		options.FileMode = 0o644
	}
	if options.DirMode == 0 {
		options.DirMode = 0o755
	}

	return &Logger{
		dataDir:  options.DataDir,
		level:    level,
		now:      options.Now,
		fileMode: options.FileMode,
		dirMode:  options.DirMode,
	}, nil
}

// Debug 在配置允许时写入一条调试日志
// 接收消息，无返回值
func (l *Logger) Debug(message string) {
	_ = l.write("DEBUG", debugLevel, message)
}

// Info 在配置允许时写入一条信息日志
// 接收消息，无返回值
func (l *Logger) Info(message string) {
	_ = l.write("INFO", infoLevel, message)
}

// Warn 在配置允许时写入一条警告日志
// 接收消息，无返回值
func (l *Logger) Warn(message string) {
	_ = l.write("WARN", warnLevel, message)
}

// Error 在配置允许时写入一条错误日志
// 接收消息，无返回值
func (l *Logger) Error(message string) {
	_ = l.write("ERROR", errorLevel, message)
}

// write 将通过级别过滤的单条日志追加到当天的日志文件
// 接收标签、级别、消息，返回操作错误
func (l *Logger) write(label string, level int, message string) error {
	if level < l.level {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	logDir := filepath.Join(l.dataDir, "logs") // 日志目录
	mkdirErr := os.MkdirAll(logDir, l.dirMode) // 创建所需目录时的错误
	if mkdirErr != nil {
		return mkdirErr
	}

	now := l.now()                                                                     // 操作发生时的时间
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", now.Format("2006-01-02")))  // 日志路径
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, l.fileMode) // 文件、以指定模式打开文件时的错误
	if err != nil {
		return err
	}
	defer file.Close()

	line := fmt.Sprintf("%s [%s] %s\n", now.Format(time.RFC3339), label, message) // 行
	_, err = file.WriteString(line)
	return err
}

// parseLevel 将日志级别字符串转换为内部使用的数值
// 接收级别，返回整数及可能发生的错误
func parseLevel(level string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "debug":
		return debugLevel, nil
	case "info":
		return infoLevel, nil
	case "warn":
		return warnLevel, nil
	case "error":
		return errorLevel, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", level)
	}
}
