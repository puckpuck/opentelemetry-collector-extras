package common

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

const MatcherDelim = "_!@#_"

type MetricGroup struct {
	// Name of the metric group
	Name string `mapstructure:"name,required"`

	// Selectors for building the matchkey for the target signal resource
	TargetSelectors TargetSelectors `mapstructure:"target_selectors,required"`

	// Selectors for building the matchkey for the source metric
	MetricsSelectors []Selector `mapstructure:"metrics_selectors,required"`

	// the metrics to add to the target signal resource
	MetricsMatchers []MetricsToAdd `mapstructure:"metrics_to_add,required"`
}

type AttributeTypeID string

const (
	// AttributeTypeResource is the type for resource attributes
	AttributeTypeResource AttributeTypeID = "resource"

	// AttributeTypeScope is the type for scope attributes
	AttributeTypeScope AttributeTypeID = "scope"

	// AttributeTypeMetric is the type for metric attributes
	AttributeTypeMetric AttributeTypeID = "metric"

	// AttributeTypeSpan is the type for span attributes
	AttributeTypeSpan AttributeTypeID = "span"

	// AttributeTypeLog is the type for log attributes
	AttributeTypeLog AttributeTypeID = "log"
)

func (c *AttributeTypeID) UnmarshalText(text []byte) error {
	str := AttributeTypeID(strings.ToLower(string(text)))
	switch str {
	case AttributeTypeResource, AttributeTypeScope, AttributeTypeMetric, AttributeTypeSpan, AttributeTypeLog:
		*c = str
		return nil
	default:
		return fmt.Errorf("unknown attribute type %v", str)
	}
}

type Selector struct {
	// Can be one of "resource", "scope", "metric", "span", or "log"
	AttributeType AttributeTypeID `mapstructure:"attribute_type,required"`

	// Name of the attribute to use for matching
	Name string `mapstructure:"name,required"`
}

type TargetSelectors struct {
	// Selectors for spans
	SpansSelectors []Selector `mapstructure:"spans"`

	// Selectors for logs
	LogsSelectors []Selector `mapstructure:"logs"`
}

type MetricsToAdd struct {
	// InstrumentationScope to match for metric
	InstrumentationScope string `mapstructure:"instrumentation_scope,required"`

	// Metrics to add for this Matcher
	MetricsMatchers []MetricsMatcher `mapstructure:"metrics,required"`
}

type MetricsMatcher struct {
	// Name of metric to be added
	Name string `mapstructure:"name,required"`

	// Attribute key and values that must match for this metric to be added (optional)
	Attributes map[string]string `mapstructure:"include_only_attributes"`

	// New name for the metric (optional)
	NewName string `mapstructure:"new_name"`
}

func IsSelectable(selectors []Selector, r pcommon.Resource, is pcommon.InstrumentationScope, attrs pcommon.Map) (string, bool) {
	id := ""
	for _, s := range selectors {
		var val pcommon.Value
		var ok bool
		switch s.AttributeType {
		case AttributeTypeResource:
			val, ok = r.Attributes().Get(s.Name)
		case AttributeTypeScope:
			val, ok = is.Attributes().Get(s.Name)
		default:
			val, ok = attrs.Get(s.Name)
		}
		if !ok {
			return "", false
		}
		id += val.AsString() + MatcherDelim
	}
	return id, true
}
