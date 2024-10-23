package metricsasattributesprocessor

import (
	"context"
	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/cache"
	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"

	"go.uber.org/zap"
)

type tracesProcessor struct {
	config *Config
	cache  cache.MetricCache
	logger *zap.Logger
}

func newTracesProcessor(set processor.Settings, cfg *Config) (*tracesProcessor, error) {
	p := &tracesProcessor{
		config: cfg,
		cache:  cache.GetCache(set.ID.String(), cfg.CacheTtl, cfg.MetricGroups, set.Logger),
		logger: set.Logger,
	}
	return p, nil
}
func (tp *tracesProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)

			for k := 0; k < ss.Spans().Len(); k++ {
				tp.addMetricsToSpan(rs.Resource(), ss.Scope(), ss.Spans().At(k))
			}
		}
	}

	return td, nil
}

func (tp *tracesProcessor) addMetricsToSpan(r pcommon.Resource, is pcommon.InstrumentationScope, s ptrace.Span) {
	for _, configMG := range tp.config.MetricGroups {

		if id, ok := tp.isSelectableSpan(configMG, r, is, s.Attributes()); ok {
			added := 0
			cacheMG := tp.cache.MetricGroups[configMG.Name]
			cacheMG.Mutex.RLock()
			if cacheMG.HasMatchedMetrics(id) {
				mmc := cacheMG.GetMatchedMetricsCache(id)
				mmc.Mutex.RLock()
				for k, v := range mmc.Metrics {
					switch v.DataPoint.ValueType() {
					case pmetric.NumberDataPointValueTypeDouble:
						s.Attributes().PutDouble(k, v.DataPoint.DoubleValue())
						added++
					case pmetric.NumberDataPointValueTypeInt:
						s.Attributes().PutInt(k, v.DataPoint.IntValue())
						added++
					}
				}
				mmc.Mutex.RUnlock()
				tp.logger.Debug("Added metrics to span", zap.String("id", id), zap.Int("added", added))
			} else {
				tp.logger.Debug("Selected span does not have metrics", zap.String("id", id))
			}
			cacheMG.Mutex.RUnlock()
		}
	}
}

func (tp *tracesProcessor) isSelectableSpan(mg common.MetricGroup, r pcommon.Resource, is pcommon.InstrumentationScope, spanAttrs pcommon.Map) (id string, ok bool) {
	id = ""
	for _, ms := range mg.TargetSelectors.SpansSelectors {
		switch ms.AttributeType {
		case common.AttributeTypeResource:
			if s, ok := r.Attributes().Get(ms.Name); ok {
				id += s.AsString() + MatcherDelim
			} else {
				return "", false
			}
		case common.AttributeTypeScope:
			if s, ok := is.Attributes().Get(ms.Name); ok {
				id += s.AsString() + MatcherDelim
			} else {
				return "", false
			}
		case common.AttributeTypeSpan:
			if s, ok := spanAttrs.Get(ms.Name); ok {
				id += s.AsString() + MatcherDelim
			} else {
				return "", false
			}
		}
	}

	return id, true
}
