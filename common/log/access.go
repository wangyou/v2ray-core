package log

import (
	"github.com/v2ray/v2ray-core/common/serial"
)

// AccessStatus is the status of an access request from clients.
type AccessStatus string

const (
	AccessAccepted = AccessStatus("accepted")
	AccessRejected = AccessStatus("rejected")
)

var (
	accessLoggerInstance logWriter = &noOpLogWriter{}
)

type accessLog struct {
	From   serial.String
	To     serial.String
	Status AccessStatus
	Reason serial.String
}

func (this *accessLog) String() string {
	return this.From.String() + " " + string(this.Status) + " " + this.To.String() + " " + this.Reason.String()
}

// InitAccessLogger initializes the access logger to write into the give file.
func InitAccessLogger(file string) error {
	logger, err := newFileLogWriter(file)
	if err != nil {
		Error("Failed to create access logger on file (", file, "): ", file, err)
		return err
	}
	accessLoggerInstance = logger
	return nil
}

// Access writes an access log.
func Access(from, to serial.String, status AccessStatus, reason serial.String) {
	accessLoggerInstance.Log(&accessLog{
		From:   from,
		To:     to,
		Status: status,
		Reason: reason,
	})
}
