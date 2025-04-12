package metricsasattributesprocessor

import (
	"context"

	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/cache"
	"github.com/puckpuck/opentelemetry-collector-extras/metricsasattributesprocessor/internal/common"
	"github.com/vodkaslime/wildcard"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type metricsProcessor struct {
	config          *Config
	cache           cache.MetricCache
	wildcardMatcher *wildcard.Matcher
	logger          *zap.Logger
}

func newMetricsProcessor(set processor.Settings, cfg *Config) (*metricsProcessor, error) {
	p := &metricsProcessor{
		config:          cfg,
		cache:           cache.GetCache(set.ID.String(), cfg.CacheTtl, cfg.MetricGroups, set.Logger),
		wildcardMatcher: wildcard.NewMatcher(),
		logger:          set.Logger,
	}
	return p, nil
}

func (mp *metricsProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)

			// If this scope isn't specified in any metric group we can skip it
			if !mp.isCheckedScope(sm.Scope()) {
				mp.logger.Debug("Skipping scope", zap.String("scope", sm.Scope().Name()))
				continue
			}
			mp.logger.Debug("Processing scope", zap.String("scope", sm.Scope().Name()))

			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				// Only Gauge and Sum metric types are supported
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					for l := 0; l < m.Gauge().DataPoints().Len(); l++ {
						dp := m.Gauge().DataPoints().At(l)
						mp.checkAndCacheDataPoint(m.Name(), rm.Resource(), sm.Scope(), dp)
					}
				case pmetric.MetricTypeSum:
					for l := 0; l < m.Sum().DataPoints().Len(); l++ {
						dp := m.Sum().DataPoints().At(l)
						mp.checkAndCacheDataPoint(m.Name(), rm.Resource(), sm.Scope(), dp)
					}
				default:
				}
			}
		}
	}
	return md, nil
}

func (mp *metricsProcessor) isCheckedScope(is pcommon.InstrumentationScope) bool {
	for _, mg := range mp.config.MetricGroups {
		for _, mm := range mg.MetricsMatchers {
			if match, _ := mp.wildcardMatcher.Match(mm.InstrumentationScope, is.Name()); match {
				return true
			}
		}
	}
	return false
}

func (mp *metricsProcessor) checkAndCacheDataPoint(name string, r pcommon.Resource, is pcommon.InstrumentationScope, dp pmetric.NumberDataPoint) {
	for _, mg := range mp.config.MetricGroups {
		if id, metricName, ok := mp.isMatchedMetric(mg, name, r, is, dp.Attributes()); ok {
			mp.logger.Debug("Matched metric", zap.String("checkName", name), zap.String("metricName", metricName), zap.String("metric_group", mg.Name), zap.String("id", id))
			mp.cache.MetricGroups[mg.Name].SetMetric(id, metricName, &dp)
		} else {
			mp.logger.Debug("Did not match metric", zap.String("checkName", name), zap.String("metricName", metricName), zap.String("metric_group", mg.Name), zap.String("id", id))
		}
	}
}

func (mp *metricsProcessor) isMatchedMetric(mg common.MetricGroup, name string, r pcommon.Resource, is pcommon.InstrumentationScope, metricAttrs pcommon.Map) (id string, metricName string, ok bool) {
	// to match a metric within this metric group, we need to ensure it has the necessary attributes to be selected
	// if this metric does not have the necessary attributes, we can skip it
	// an id is created by concatenating the values of the attributes selected for the metric group which will
	// be used to identify the MatchedMetricsCache for this metric group
	id, _ = common.IsSelectable(mg.MetricsSelectors, r, is, metricAttrs)

	for _, mm := range mg.MetricsMatchers {
		if match, _ := mp.wildcardMatcher.Match(mm.InstrumentationScope, is.Name()); match {
			for _, m := range mm.MetricsMatchers {

				if match, _ = mp.wildcardMatcher.Match(m.Name, name); match {
					metricName = name
					if m.NewName != "" {
						metricName = m.NewName
					}

					if m.Attributes != nil {
						for k, v := range m.Attributes {
							if s, ok := metricAttrs.Get(k); ok {
								if s.AsString() != v {
									goto nextMetricToMatch
								}
							} else {
								goto nextMetricToMatch
							}
						}
					}
					return id, metricName, true
				}

			nextMetricToMatch:
			}
		}
	}
	return "", "", false
}
