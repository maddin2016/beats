// +build windows

package diskio

import (
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/windows/perfmon"
)

type RawValue struct {
	Name  string
	Value interface{}
}

func init() {
	if err := mb.Registry.AddMetricSet("system", "diskio", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system disk IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	statistics  *DiskIOStat
	oldRawValue map[string]*perfmon.PdhRawCounter
	executed    bool
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms := &MetricSet{
		BaseMetricSet: base,
		statistics:    NewDiskIOStat(),
	}
	return ms, nil
}

// Fetch fetches disk IO metrics from the OS.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	query, err := perfmon.NewQuery("")
	if err != nil {
		return nil, err
	}
	defer query.Close()

	err = query.AddCounter("\\LogicalDisk(*)\\Disk Write Bytes/sec", perfmon.FloatFlormat, "")
	if err != nil {
		return nil, err
	}

	if err = query.Execute(); err != nil {
		return nil, err
	}

	actualRawValues, err := perfmon.PdhGetRawCounterArray(query.Counters["\\LogicalDisk(C:)\\Disk Write Bytes/sec"].Handle)
	if err != nil {
		return nil, err
	}

	rtn := make(map[string][]RawValue, len(actualRawValues))

	events := make([]common.MapStr, 0, len(stats))

	for _, rawValue := range actualRawValues {
		value, err := perfmon.PdhCalculateCounterFromRawValue(query.Counters["\\LogicalDisk(C:)\\Disk Write Bytes/sec"].Handle, perfmon.PdhFmtDouble|perfmon.PdhFmtNoCap100, &rawValue.Value, m.oldRawValue[rawValue.Name])

		if err != nil {
			switch err {
			case perfmon.PDH_CALC_NEGATIVE_DENOMINATOR:
			case perfmon.PDH_INVALID_DATA:
				if m.executed {
					return nil, err
				}
			default:
				return nil, err
			}
		}

		event := common.MapStr{
			"name": counters.Name,
			"read": common.MapStr{
				"count": *(*float64)(unsafe.Pointer(&value.LongValue)),
				"time":  counters.ReadTime,
				"bytes": counters.ReadBytes,
			},
			"write": common.MapStr{
				"count": counters.WriteCount,
				"time":  counters.WriteTime,
				"bytes": counters.WriteBytes,
			},
			"io": common.MapStr{
				"time": counters.IoTime,
			},
		}

		events = append(events, event)
	}

	if !m.executed {
		m.executed = true
	}

	if err != nil {
		return nil, err
	}

	for _, counters := range stats {

		event := common.MapStr{
			"name": counters.Name,
			"read": common.MapStr{
				"count": *(*float64)(unsafe.Pointer(&value.LongValue)),
				"time":  counters.ReadTime,
				"bytes": counters.ReadBytes,
			},
			"write": common.MapStr{
				"count": counters.WriteCount,
				"time":  counters.WriteTime,
				"bytes": counters.WriteBytes,
			},
			"io": common.MapStr{
				"time": counters.IoTime,
			},
		}

		events = append(events, event)

		if counters.SerialNumber != "" {
			event["serial_number"] = counters.SerialNumber
		}
	}

	// open a sampling means store the last cpu counter
	m.statistics.CloseSampling()

	return events, nil
}
