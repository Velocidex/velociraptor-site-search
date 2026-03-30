package api

import (
	"log"
	"log/syslog"
	"os"
	"sync"
	"time"
)

var (
	syslog_logger = &syslogLogger{
		info_logger:  log.New(os.Stderr, "", 0),
		debug_logger: log.New(os.Stderr, "", 0),
		error_logger: log.New(os.Stderr, "", 0),
	}
)

type syslogLogger struct {
	mu sync.Mutex

	info_logger  *log.Logger
	debug_logger *log.Logger
	error_logger *log.Logger
}

func NewSyslogLogger() (res *syslogLogger, err error) {
	res = &syslogLogger{}
	res.info_logger, err = syslog.NewLogger(syslog.LOG_INFO, 0)
	if err != nil {
		return nil, err
	}

	res.debug_logger, err = syslog.NewLogger(syslog.LOG_DEBUG, 0)
	if err != nil {
		return nil, err
	}

	res.error_logger, err = syslog.NewLogger(syslog.LOG_ERR, 0)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type Logger struct{}

func (self *Logger) timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (self *Logger) Error(format string, args ...interface{}) {
	syslog_logger.error_logger.Printf(
		self.timestamp()+" ERROR:"+format+"\n", args...)
}

func (self *Logger) Debug(format string, args ...interface{}) {
	syslog_logger.debug_logger.Printf(
		self.timestamp()+" DEBUG:"+format+"\n", args...)
}

func (self *Logger) Info(format string, args ...interface{}) {
	syslog_logger.info_logger.Printf(
		self.timestamp()+" INFO:"+format+"\n", args...)
}

func (self *Logger) Write(b []byte) (int, error) {
	return syslog_logger.info_logger.Writer().Write(b)
}
