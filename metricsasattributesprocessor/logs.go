package metricsasattributesprocessor

import (
	"context"

	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/cache"
	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"go.uber.org/zap"
)

type logsProcessor struct {
	config *Config
	cache  cache.MetricCache
	logger *zap.Logger
}

func newLogsProcessor(set processor.Settings, cfg *Config) (*logsProcessor, error) {
	p := &logsProcessor{
		config: cfg,
		cache:  cache.GetCache(set.ID.String(), cfg.CacheTtl, cfg.MetricGroups, set.Logger),
		logger: set.Logger,
	}
	return p, nil
}
func (lp *logsProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)

		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)

			for k := 0; k < sl.LogRecords().Len(); k++ {
				lp.addMetricsToLog(rl.Resource(), sl.Scope(), sl.LogRecords().At(k))
			}
		}
	}

	return ld, nil
}

func (lp *logsProcessor) addMetricsToLog(r pcommon.Resource, is pcommon.InstrumentationScope, l plog.LogRecord) {
	for _, configMG := range lp.config.MetricGroups {

		if id, ok := common.IsSelectable(configMG.TargetSelectors.LogsSelectors, r, is, l.Attributes()); ok {
			added := 0
			cacheMG := lp.cache.MetricGroups[configMG.Name]
			cacheMG.Mutex.RLock()
			if cacheMG.HasMatchedMetrics(id) {
				mmc := cacheMG.GetMatchedMetricsCache(id)
				mmc.Mutex.RLock()
				for k, v := range mmc.Metrics {
					switch v.DataPoint.ValueType() {
					case pmetric.NumberDataPointValueTypeDouble:
						l.Attributes().PutDouble(k, v.DataPoint.DoubleValue())
						added++
					case pmetric.NumberDataPointValueTypeInt:
						l.Attributes().PutInt(k, v.DataPoint.IntValue())
						added++
					default:
						// unsupported type do nothing
					}
				}
				mmc.Mutex.RUnlock()
				lp.logger.Debug("Added metrics to log", zap.String("id", id), zap.Int("added", added))
			} else {
				lp.logger.Debug("Selected log does not have metrics", zap.String("id", id))
			}
			cacheMG.Mutex.RUnlock()
		}
	}
}
