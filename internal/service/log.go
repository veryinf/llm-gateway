package service

import (
	"context"
	"io"
	"llm-gateway/internal/core"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/eventbus"
)

var (
	DefaultLogService *LogService
)

// LogEvent 日志事件
type LogEvent struct {
	Level   slog.Level
	Message string
	Attrs   []slog.Attr
	Time    time.Time
	Raw     string // 原始格式化后的日志行
}

type LogService struct {
	mu      sync.RWMutex
	logDir  string
	mode    string // console | file | both
	today   string
	logFile *os.File
	writer  io.Writer
	level   *slog.LevelVar
	jsonHdl slog.Handler
	eb      *eventbus.EventBus[LogEvent]
}

// InitSlog 初始化全局 slog 日志。
//
//	mode:   "console" | "file" | "both"
//	level:  "debug" | "info" | "warn" | "error"
//	logDir: 日志文件目录，为空时默认 "./logs"
func InitSlog(mode, level, logDir string) *LogService {
	lvl := parseLogLevel(level)

	if logDir == "" {
		logDir = "./logs"
	}

	svc := &LogService{
		mode:   mode,
		logDir: logDir,
		level:  lvl,
		eb:     eventbus.NewEventBus[LogEvent](),
	}

	var textHandler slog.Handler
	opts := &slog.HandlerOptions{
		Level:       lvl,
		ReplaceAttr: replaceTimeAttr,
	}

	switch mode {
	case "console":
		textHandler = slog.NewTextHandler(os.Stdout, opts)

	case "file", "both":
		svc.mu.Lock()
		svc.initFileWriter()
		svc.mu.Unlock()
		if mode == "both" {
			svc.writer = io.MultiWriter(os.Stdout, svc.logFile)
		} else {
			svc.writer = svc.logFile
		}
		textHandler = slog.NewTextHandler(svc.writer, opts)
		go svc.cleanupLoop()
		go svc.checkDateChangeLoop()

	default:
		// 默认 console
		svc.mode = "console"
		textHandler = slog.NewTextHandler(os.Stdout, opts)
	}

	// 用 eventHandler 包装，发布日志事件
	textHandler = &eventHandler{base: textHandler, eb: svc.eb}

	logger := slog.New(textHandler)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(lvl.Level())

	// Web 请求日志用 JSON Handler，写入同一输出目标
	jsonOpts := &slog.HandlerOptions{Level: lvl, ReplaceAttr: replaceTimeAttr}
	var jsonBase slog.Handler
	switch mode {
	case "file", "both":
		jsonBase = slog.NewJSONHandler(svc.writer, jsonOpts)
	default:
		jsonBase = slog.NewJSONHandler(os.Stdout, jsonOpts)
	}
	svc.jsonHdl = &eventHandler{base: jsonBase, eb: svc.eb}

	DefaultLogService = svc
	return svc
}

// JSONHandler 返回供 Web 请求日志使用的 JSON Handler。
func (s *LogService) JSONHandler() slog.Handler {
	return s.jsonHdl
}

// GetLogPath 返回指定日期的日志文件路径。
func (s *LogService) GetLogPath(date string) string {
	if date == "" {
		date = time.Now().Format("20060102")
	}
	return filepath.Join(s.logDir, date+".log")
}

// initFileWriter 打开当天的日志文件（调用方需持有 s.mu 写锁）。
func (s *LogService) initFileWriter() {
	day := time.Now().Format("20060102")
	if s.today == day && s.logFile != nil {
		return
	}

	oldFile := s.logFile

	err := os.MkdirAll(s.logDir, 0755)
	if err != nil {
		return
	}
	logPath := filepath.Join(s.logDir, day+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file", "path", logPath, "error", err)
		return
	}

	s.today = day
	s.logFile = file

	if oldFile != nil {
		_ = oldFile.Close()
	}
}

// rotateFile 日期变化时轮转日志文件。
func (s *LogService) rotateFile() {
	s.mu.Lock()
	s.initFileWriter()
	f := s.logFile
	s.mu.Unlock()

	if f == nil {
		return
	}

	switch s.mode {
	case "file":
		s.writer = f
	case "both":
		s.writer = io.MultiWriter(os.Stdout, f)
	}
}

func (s *LogService) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

func (s *LogService) checkDateChangeLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.rotateFile()
	}
}

func (s *LogService) cleanup() {
	retention := int(GetConfig("system.log.retention").Int())
	if retention < 1 {
		retention = 7
	}
	cutoff := time.Now().AddDate(0, 0, -retention)

	entries, err := os.ReadDir(s.logDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			name := strings.TrimSuffix(entry.Name(), ".log")
			t, err := time.Parse("20060102", name)
			if err != nil {
				continue
			}
			if t.Before(cutoff) {
				path := filepath.Join(s.logDir, entry.Name())
				_ = os.Remove(path)
				slog.Info("removed expired log file", "path", path)
			}
		}
	}
}

// replaceTimeAttr 统一时间格式为 "06-01-02 15:04:05"。
func replaceTimeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.String(a.Key, a.Value.Time().Format("06-01-02 15:04:05"))
	}
	return a
}

// parseLogLevel 解析日志级别字符串。
func parseLogLevel(level string) *slog.LevelVar {
	var lvl slog.LevelVar
	switch strings.ToLower(level) {
	case "debug":
		lvl.Set(slog.LevelDebug)
	case "warn", "warning":
		lvl.Set(slog.LevelWarn)
	case "error":
		lvl.Set(slog.LevelError)
	default:
		lvl.Set(slog.LevelInfo)
	}
	return &lvl
}

// AddLogCallback 添加日志事件回调函数
func (s *LogService) AddLogCallback(callback core.EventHandler[LogEvent]) {
	s.eb.Subscribe("log", callback, true, 0, nil)
}

// RemoveLogCallback 移除日志事件回调函数
func (s *LogService) RemoveLogCallback(callback core.EventHandler[LogEvent]) {
	s.eb.Unsubscribe("log", callback)
}

// eventHandler 包装 slog.Handler，在写入日志时发布事件
type eventHandler struct {
	base  slog.Handler
	eb    *eventbus.EventBus[LogEvent]
	attrs []slog.Attr // 累积的 attrs（来自 WithAttrs）
}

func (h *eventHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *eventHandler) Handle(ctx context.Context, record slog.Record) error {
	// 先调用底层 handler 写入日志
	err := h.base.Handle(ctx, record)

	// 合并累积的 attrs 和 record 中的 attrs
	attrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	event := LogEvent{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
		Time:    record.Time,
	}

	// 格式化原始日志行
	var buf strings.Builder
	buf.WriteString(record.Time.Format("06-01-02 15:04:05"))
	buf.WriteString(" ")
	buf.WriteString(record.Level.String())
	buf.WriteString(" ")
	buf.WriteString(record.Message)
	for _, attr := range attrs {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		buf.WriteString("=")
		buf.WriteString(attr.Value.String())
	}
	event.Raw = buf.String()

	h.eb.Publish(eventbus.Event[LogEvent]{Topic: "log", Payload: event})
	return err
}

func (h *eventHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 合并已有 attrs
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)
	return &eventHandler{base: h.base.WithAttrs(attrs), eb: h.eb, attrs: newAttrs}
}

func (h *eventHandler) WithGroup(name string) slog.Handler {
	return &eventHandler{base: h.base.WithGroup(name), eb: h.eb, attrs: h.attrs}
}
