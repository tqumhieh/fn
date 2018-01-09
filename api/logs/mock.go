package logs

import (
	"context"
	"io"

	"github.com/fnproject/fn/api/models"
)

type mock struct {
	Logs map[string]io.Reader
}

func NewMock() models.LogStore {
	return &mock{make(map[string]io.Reader)}
}

func (m *mock) InsertLog(ctx context.Context, appID, callID string, callLog io.Reader) error {
	m.Logs[callID] = callLog
	return nil
}

func (m *mock) GetLog(ctx context.Context, appID, callID string) (io.Reader, error) {
	logEntry := m.Logs[callID]
	if logEntry == nil {
		return nil, models.ErrCallLogNotFound
	}

	return logEntry, nil
}
