package gormlogger

import (
	"time"

	"gorm.io/gorm/logger"
)

// config 内部配置结构
type config struct {
	// slowThreshold 慢查询阈值
	// 执行时间超过此值的SQL会被记录为慢查询
	// 默认: 2秒
	slowThreshold time.Duration

	// ignoreRecordNotFoundError 是否忽略"记录未找到"错误
	// true: 不记录 ErrRecordNotFound 错误
	// false: 记录 ErrRecordNotFound 错误（作为Error级别）
	// 默认: true
	ignoreRecordNotFoundError bool

	// logLevel 日志级别
	// 决定了哪些 SQL 会被记录
	// 默认: logger.Warn
	logLevel logger.LogLevel
}

// defaultConfig 返回默认配置
func defaultConfig() *config {
	return &config{
		slowThreshold:             2 * time.Second,
		ignoreRecordNotFoundError: true,
		logLevel:                  logger.Warn,
	}
}

// Option 配置选项函数类型
type Option func(*config)

// WithSlowThreshold 设置慢查询阈值
//
// 参数:
//
//	slowThreshold - 判定为慢查询的时间阈值
//
// 作用:
//   - 执行时间超过此阈值的 SQL 语句将被记录为警告(Warn)日志
//   - 用于发现和优化性能较差的数据库操作
//
// 示例:
//
//	WithSlowThreshold(200 * time.Millisecond) // 设置阈值为200ms
//	WithSlowThreshold(time.Second)            // 设置阈值为1秒
func WithSlowThreshold(slowThreshold time.Duration) Option {
	return func(c *config) {
		c.slowThreshold = slowThreshold
	}
}

// WithIgnoreRecordNotFoundError 设置是否忽略"RecordNotFoundError"错误
//
// 参数:
//
//	ignoreRecordNotFoundError - true: 忽略; false: 不忽略
//
// 作用:
//   - GORM 在查询不到数据时会返回 ErrRecordNotFound
//   - 在许多业务逻辑中，查不到数据是正常情况（如判断用户是否存在）
//   - 设置为 true 可以避免日志中充斥着大量的"record not found"错误干扰排查其他问题
//   - 设置为 false 则会将此类错误记录为 Error 级别日志
//
// 示例:
//
//	WithIgnoreRecordNotFoundError(true)  // 忽略(默认)
//	WithIgnoreRecordNotFoundError(false) // 记录错误
func WithIgnoreRecordNotFoundError(ignoreRecordNotFoundError bool) Option {
	return func(c *config) {
		c.ignoreRecordNotFoundError = ignoreRecordNotFoundError
	}
}

// WithLogLevel 设置日志级别
//
// 参数:
//
//	level - GORM定义的日志级别
//
// 可选值:
//   - logger.Silent: 静默模式，不记录任何日志
//   - logger.Error:  仅记录错误日志（包括不被忽略的RecordNotFound）
//   - logger.Warn:   记录错误日志 + 慢查询日志（默认）
//   - logger.Info:   记录所有 SQL 语句（相当于 Debug 模式）
//
// 作用:
//   - 控制 SQL 日志的详细程度
//   - 可以通过 LogMode 方法在运行时动态修改
//
// 示例:
//
//	WithLogLevel(logger.Info)  // 记录所有SQL，适合开发调试
//	WithLogLevel(logger.Error) // 仅记录错误，适合生产环境减少日志量
func WithLogLevel(level logger.LogLevel) Option {
	return func(c *config) {
		c.logLevel = level
	}
}
