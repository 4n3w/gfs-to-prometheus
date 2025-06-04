package tsdb

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb"
)

type Writer struct {
	db       *tsdb.DB
	appender storage.Appender
}

func NewWriter(dataPath string) (*Writer, error) {
	absPath, err := filepath.Abs(dataPath)
	if err != nil {
		return nil, fmt.Errorf("invalid data path: %w", err)
	}

	opts := tsdb.DefaultOptions()
	opts.RetentionDuration = int64(365 * 24 * time.Hour / time.Millisecond) // 1 year

	db, err := tsdb.Open(absPath, nil, nil, opts, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open TSDB: %w", err)
	}

	return &Writer{
		db:       db,
		appender: db.Appender(context.Background()),
	}, nil
}

func (w *Writer) Close() error {
	if err := w.Commit(); err != nil {
		return err
	}
	return w.db.Close()
}

func (w *Writer) WriteMetric(name string, labelPairs map[string]string, value float64, ts time.Time) error {
	lbls := labels.NewBuilder(labels.EmptyLabels())
	lbls.Set(labels.MetricName, name)
	
	for k, v := range labelPairs {
		lbls.Set(k, v)
	}

	_, err := w.appender.Append(0, lbls.Labels(), timestamp.FromTime(ts), value)
	return err
}

func (w *Writer) Commit() error {
	if w.appender == nil {
		return nil
	}
	
	if err := w.appender.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	
	w.appender = w.db.Appender(context.Background())
	return nil
}

func (w *Writer) Rollback() error {
	if w.appender == nil {
		return nil
	}
	
	if err := w.appender.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}
	
	w.appender = w.db.Appender(context.Background())
	return nil
}