package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	once         sync.Once
	globalLogger *logrus.Logger
)

// Init 初始化Logger
// 如果不传入任何选项，则只输出到控制台
// 初始化错误降级到默认配置
func Init(options ...Option) {
	once.Do(func() {
		logger, err := newLogger(options...)
		if err != nil {
			globalLogger.WithError(err).Errorf("[logger] init logger failed, fallback to default logger")
			return
		}
		globalLogger = logger
	})
}

// init 先初始化一个默认的logger，保证logger一定不为nil
func init() {
	logger, err := newLogger()
	if err != nil {
		// 记录错误，但仍创建可用的默认 logger
		fmt.Fprintf(os.Stderr, "[logger] init default logger failed: %v, using fallback logger\n", err)
		globalLogger = createFallbackLogger()
		return
	}
	globalLogger = logger
}

func createFallbackLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})
	return logger
}

// config 内部配置结构（不对外暴露）
type config struct {
	// fileName 日志文件路径
	// 如果为空字符串("")，则只输出到控制台，不写入文件
	// 示例: "logs/app.log"、"/var/log/myapp.log"
	fileName string

	// level 日志级别
	// 默认: logrus.InfoLevel
	// 可选值: logrus.PanicLevel、logrus.FatalLevel、logrus.ErrorLevel、
	//         logrus.WarnLevel、logrus.InfoLevel、logrus.DebugLevel、logrus.TraceLevel
	level logrus.Level

	// maxSize 单个日志文件的最大大小(单位：MB)
	// 当文件大小达到此值时，会进行日志分割
	// 默认: 32 (32MB)
	// 设置为0表示不限制文件大小
	maxSize int

	// maxBackups 保留的旧日志文件最大数量
	// 当日志文件被分割后，会保留指定数量的旧日志文件
	// 默认: 5
	// 设置为0表示不保留任何旧日志文件
	maxBackups int

	// maxAge 保留旧日志文件的最大天数(基于文件创建时间)
	// 超过指定天数的旧日志文件会被删除
	// 默认: 30 (30天)
	// 设置为0表示不根据天数删除旧文件
	maxAge int

	// compress 是否压缩归档的旧日志文件
	// 为true时，分割后的旧日志文件会被压缩为.gz格式
	// 默认: false
	// 注意: 只有归档的旧文件会被压缩，当前活跃日志文件不会被压缩
	compress bool

	// jsonFormat 是否使用JSON格式输出日志
	// 为true时，日志以JSON格式输出，适合机器解析和日志收集系统(如ELK)
	// 为false时，使用文本格式输出，更适合人类阅读和控制台显示
	// 默认: false
	jsonFormat bool

	// withConsole 是否同时输出到控制台
	// 当设置了fileName时，默认会将日志同时输出到文件和控制台
	// 将此字段设置为false可禁用控制台输出，只写入文件
	// 默认: true
	// 注意: 当fileName为空时，此字段会被忽略，始终输出到控制台
	withConsole bool
}

// Option 配置选项函数类型
type Option func(*config)

// DefaultConfig 返回默认配置
func defaultConfig() *config {
	return &config{
		fileName:    "", // 默认输出到控制台
		level:       logrus.DebugLevel,
		maxSize:     32,
		maxBackups:  5,
		maxAge:      30,
		compress:    false,
		jsonFormat:  false,
		withConsole: true,
	}
}

func validateConfig(cfg *config) error {
	if cfg.maxSize < 0 {
		return fmt.Errorf("maxSize cannot be negative")
	}
	if cfg.maxBackups < 0 && cfg.maxBackups != -1 {
		return fmt.Errorf("maxBackups must be >=0 or -1")
	}
	if cfg.maxAge < 0 {
		return fmt.Errorf("maxAge cannot be negative")
	}
	return nil
}

func newLogger(options ...Option) (*logrus.Logger, error) {
	// 应用默认配置
	cfg := defaultConfig()

	// 配置验证
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 应用所有选项
	for _, option := range options {
		option(cfg)
	}

	// 创建logrus实例
	logger := logrus.New()

	// 设置日志级别
	logger.SetLevel(cfg.level)

	// 设置格式
	if cfg.jsonFormat {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
			DisableColors:   false,
		})
	}

	// context hook
	logger.AddHook(&contextHook{})

	// 如果没有文件名，只输出到控制台
	if cfg.fileName == "" {
		logger.SetOutput(os.Stdout)
		return logger, nil
	}

	// 设置文件输出
	if err := setupFileOutput(logger, cfg); err != nil {
		return nil, err
	}

	return logger, nil
}

// setupFileOutput 设置文件输出
func setupFileOutput(logger *logrus.Logger, cfg *config) error {
	// 确保目录存在
	dir := filepath.Dir(cfg.fileName)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("[logger] logger create output dir fail: %w", err)
		}
	}

	// 配置Lumberjack
	logRotator := &lumberjack.Logger{
		Filename:   cfg.fileName,
		MaxSize:    cfg.maxSize,
		MaxBackups: cfg.maxBackups,
		MaxAge:     cfg.maxAge,
		Compress:   cfg.compress,
		LocalTime:  true,
	}

	// 设置输出
	if cfg.withConsole {
		// 同时输出到文件和控制台
		logger.SetOutput(logRotator)
		addConsoleHook(logger, cfg.jsonFormat)
	} else {
		// 只输出到文件
		logger.SetOutput(logRotator)
	}

	return nil
}

// addConsoleHook 添加控制台输出的Hook
func addConsoleHook(logger *logrus.Logger, jsonFormat bool) {
	// 创建一个控制台输出的hook
	logger.AddHook(&consoleHook{
		formatter: getConsoleFormatter(jsonFormat),
	})
}

// getConsoleFormatter 获取控制台格式化器
func getConsoleFormatter(jsonFormat bool) logrus.Formatter {
	if jsonFormat {
		return &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		}
	}

	return &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
		DisableColors:   false,
	}
}

// WithFileName 设置日志文件名
//
// 参数:
//
//	fileName - 日志文件的完整路径或相对路径
//	           - 如果为空字符串("")，则只输出到控制台，不写入文件
//	           - 如果包含目录路径，会自动创建不存在的目录
//	           - 示例: "logs/app.log"、"/var/log/myapp.log"、"error.log"
//
// 注意:
//   - 当设置文件名后，默认会同时输出到文件和控制台
//   - 可以通过 WithConsoleOutput(false) 关闭控制台输出，只写入文件
//   - 文件名中可以使用时间变量占位符（如果使用支持时间分割的库）
//
// 示例:
//
//	WithFileName("logs/app.log")
//	WithFileName("/var/log/myapp/application.log")
//	WithFileName("") // 空字符串表示只输出到控制台
func WithFileName(fileName string) Option {
	return func(c *config) {
		c.fileName = fileName
	}
}

// WithLevel 设置日志级别
//
// 参数:
//
//	level - 日志级别，决定哪些级别的日志会被记录和输出
//	       级别从低到高: Trace < Debug < Info < Warn < Error < Fatal < Panic
//	       设置某个级别后，会记录该级别及更高级别的日志
//	       例如: 设置Info级别，则Info、Warn、Error、Fatal、Panic级别的日志都会被记录
//
// 预定义级别:
//   - logrus.TraceLevel: 追踪级别，最详细的日志信息
//   - logrus.DebugLevel: 调试信息，用于开发阶段
//   - logrus.InfoLevel:  常规信息，记录程序正常运行状态（默认级别）
//   - logrus.WarnLevel:  警告信息，表明可能有问题但程序仍能运行
//   - logrus.ErrorLevel: 错误信息，表明程序功能出现错误
//   - logrus.FatalLevel: 严重错误，程序会调用os.Exit(1)终止
//   - logrus.PanicLevel: 紧急错误，程序会panic并终止
//
// 示例:
//
//	WithLevel(logrus.DebugLevel)  // 开发环境，记录所有日志
//	WithLevel(logrus.InfoLevel)   // 生产环境，记录Info及以上级别
//	WithLevel(logrus.WarnLevel)   // 只记录警告和错误
func WithLevel(level logrus.Level) Option {
	return func(c *config) {
		c.level = level
	}
}

// WithLevelString 通过字符串设置日志级别
//
// 参数:
//
//	level - 日志级别的字符串表示，不区分大小写
//	       有效值: "trace", "debug", "info", "warn", "warning", "error", "fatal", "panic"
//
// 特点:
//   - 比WithLevel更灵活，可以直接从配置文件或环境变量中读取
//   - 如果传入无效字符串，会静默失败并使用当前配置（不会修改日志级别）
//   - 推荐在需要动态配置日志级别时使用此函数
//
// 示例:
//
//	WithLevelString("debug")
//	WithLevelString("INFO")
//	WithLevelString("WARNING") // 注意: "warning"等同于"warn"
//
// 使用场景:
//   - 从环境变量读取: WithLevelString(os.Getenv("LOG_LEVEL"))
//   - 从配置文件读取: WithLevelString(config.LogLevel)
func WithLevelString(level string) Option {
	return func(c *config) {
		if l, err := logrus.ParseLevel(level); err == nil {
			c.level = l
		}
		// 如果解析失败，保持现有配置不变，不抛出错误
	}
}

// WithJSONFormat 设置是否使用JSON格式输出日志
//
// 参数:
//
//	json - true: 使用JSON格式输出，适合机器解析和日志收集系统
//	       false: 使用文本格式输出，适合人类阅读和控制台显示（默认）
//
// JSON格式特点:
//   - 每行日志是一个完整的JSON对象
//   - 易于被日志收集系统（如ELK、Splunk等）解析和处理
//   - 包含标准字段: time, level, msg, 以及额外添加的fields
//   - 不适合直接阅读，但便于结构化查询和分析
//
// 文本格式特点:
//   - 人类可读，包含颜色高亮（如果控制台支持）
//   - 格式: [时间] [级别] 消息 [字段]
//   - 适合开发环境和直接查看
//
// 推荐用法:
//   - 开发环境: WithJSONFormat(false) 或省略（默认为false）
//   - 生产环境: WithJSONFormat(true) 便于日志分析
//
// 示例:
//
//	WithJSONFormat(true)   // 生产环境，使用JSON格式
//	WithJSONFormat(false)  // 开发环境，使用文本格式
func WithJSONFormat(json bool) Option {
	return func(c *config) {
		c.jsonFormat = json
	}
}

// WithConsoleOutput 设置是否输出到控制台
//
// 参数:
//
//	console - true: 将日志输出到控制台（默认）
//	          false: 不输出到控制台
//
// 注意:
//   - 此选项仅在设置了文件名（fileName非空）时有效
//   - 如果fileName为空（只输出到控制台模式），此选项会被忽略，始终输出到控制台
//   - 默认情况下，当设置了文件名时，日志会同时输出到文件和控制台
//   - 将此选项设为false，可以只写入文件而不显示在控制台
//
// 典型场景:
//  1. 开发环境: WithConsoleOutput(true) - 同时查看控制台和文件
//  2. 生产环境: WithConsoleOutput(false) - 只写入文件，减少控制台I/O
//  3. 后台服务: WithConsoleOutput(false) - 守护进程通常不需要控制台输出
//
// 示例:
//
//	// 同时输出到文件和控制台（默认行为）
//	WithFileName("app.log"), WithConsoleOutput(true)
//
//	// 只输出到文件，不显示在控制台
//	WithFileName("app.log"), WithConsoleOutput(false)
func WithConsoleOutput(console bool) Option {
	return func(c *config) {
		c.withConsole = console
	}
}

// WithMaxSize 设置日志文件最大大小
//
// 参数:
//
//	maxSize - 单个日志文件的最大大小，单位：MB（兆字节）
//	          - 当文件大小达到此值时，会触发日志分割
//	          - 默认值: 32 (32MB)
//	          - 设置为0表示不限制文件大小（不推荐，可能导致文件过大）
//
// 工作原理:
//  1. 当日志文件大小达到maxSize时，会关闭当前文件
//  2. 将当前文件重命名为带时间戳的备份文件
//  3. 创建新的日志文件继续写入
//  4. 配合maxBackups和maxAge参数管理备份文件
//
// 推荐值:
//   - 开发环境: 10-50 MB
//   - 生产环境: 50-200 MB，根据日志量调整
//   - 微服务/容器: 10-20 MB（配合日志收集Agent）
//
// 示例:
//
//	WithMaxSize(10)   // 每个日志文件最大10MB
//	WithMaxSize(100)  // 每个日志文件最大100MB
//	WithMaxSize(0)    // 不限制文件大小（慎用）
func WithMaxSize(maxSize int) Option {
	return func(c *config) {
		if maxSize >= 0 {
			c.maxSize = maxSize
		}
		// 如果传入负数，保持现有配置不变
	}
}

// WithMaxBackups 设置保留的旧日志文件最大数量
//
// 参数:
//
//	maxBackups - 保留的旧日志文件（备份文件）的最大数量
//	             - 当日志文件分割后，会保留指定数量的旧日志文件
//	             - 默认值: 5
//	             - 设置为0表示不保留任何旧日志文件，立即删除
//	             - 设置为-1表示不限制保留数量（保留所有旧文件）
//
// 注意:
//   - 此参数与WithMaxAge参数共同作用，满足任一条件就会删除旧文件
//   - 备份文件命名格式: 原文件名-时间戳.log（或.gz压缩格式）
//   - 删除策略: 当备份文件数量超过maxBackups时，删除最旧的备份文件
//
// 典型用法:
//   - 按数量保留: WithMaxBackups(10) // 保留最近10个备份文件
//   - 不保留: WithMaxBackups(0)     // 不保留任何备份，分割后立即删除旧文件
//   - 全保留: WithMaxBackups(-1)    // 保留所有备份（慎用，可能占用大量磁盘）
//
// 示例:
//
//	WithMaxBackups(5)   // 默认，保留最近5个备份文件
//	WithMaxBackups(30)  // 保留最近30个备份文件
//	WithMaxBackups(0)   // 不保留备份文件
func WithMaxBackups(maxBackups int) Option {
	return func(c *config) {
		c.maxBackups = maxBackups
	}
}

// WithMaxAge 设置保留旧日志文件的最大天数
//
// 参数:
//
//	maxAge - 保留旧日志文件的最大天数，基于文件的创建时间
//	         - 超过指定天数的旧日志文件会被自动删除
//	         - 默认值: 30 (30天)
//	         - 设置为0表示不根据天数删除旧文件（但仍然可能因数量限制被删除）
//
// 注意:
//   - 此参数与WithMaxBackups参数共同作用，满足任一条件就会删除旧文件
//   - 天数计算基于文件的修改时间（mtime），不是日志内容的时间
//   - 通常配合WithCompress(true)使用，压缩可以节省存储空间
//
// 推荐值:
//   - 开发环境: 7-14天
//   - 生产环境: 30-90天，根据合规要求调整
//   - 审计日志: 180-365天或更长，根据法规要求
//
// 示例:
//
//	WithMaxAge(7)     // 保留最近7天的日志文件
//	WithMaxAge(30)    // 默认，保留最近30天的日志文件
//	WithMaxAge(365)   // 保留最近一年的日志文件
//	WithMaxAge(0)     // 不按天数删除旧文件
func WithMaxAge(maxAge int) Option {
	return func(c *config) {
		if maxAge >= 0 {
			c.maxAge = maxAge
		}
		// 如果传入负数，保持现有配置不变
	}
}

// WithCompress 设置是否压缩归档的旧日志文件
//
// 参数:
//
//	compress - true: 压缩归档的旧日志文件为.gz格式（默认）
//	           false: 不压缩旧日志文件
//
// 压缩优势:
//   - 显著减少磁盘空间占用（通常可减少70-90%）
//   - 便于长期存储和归档
//   - 网络传输更快
//
// 压缩劣势:
//   - 需要解压才能查看日志内容
//   - 增加CPU使用（压缩过程需要计算）
//
// 工作原理:
//   - 只有归档的旧日志文件会被压缩
//   - 当前正在写入的活跃日志文件不会被压缩
//   - 压缩后的文件扩展名为 .log.gz 或 .gz
//   - 压缩在日志分割时进行，不影响当前日志写入性能
//
// 推荐用法:
//   - 生产环境: 建议开启压缩，节省存储成本
//   - 开发环境: 可关闭压缩，便于直接查看历史日志
//   - 日志量大的系统: 必须开启压缩
//
// 示例:
//
//	WithCompress(true)   // 开启压缩
//	WithCompress(false)  // 关闭压缩（默认）
func WithCompress(compress bool) Option {
	return func(c *config) {
		c.compress = compress
	}
}
