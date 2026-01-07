# github.com/kakkk/gopkg

自己封装的一些常用的组件

- `go get github.com/kakkk/gopkg/dlock@latest`
  - 分布式锁的简单实现
  - 可基于Redis或数据库（兼容MySQL、PostgreSQL、SQLite）实现
- `go get github.com/kakkk/gopkg/requestid@latest`
  - 往context中添加requestID，用于日志&链路追踪
- `go get github.com/sirupsen/logrus@latest`
  - `logrus`的简单封装
  - 使用`lumberjack`实现日志分割
  - 自动获取`kakkk/gopkg/requestid`的requestID
- `go get github.com/kakkk/gopkg/gormlogger@latest`
  - 实现 gorm logger
  - 基于`github.com/kakkk/gopkg/logger`