// +build windows

package diskio

import (
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/windows/perfmon"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "diskio", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system disk IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	statistics  *DiskIOStat
	oldRawValue *perfmon.PdhRawCounter
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

	err = query.AddCounter("\\LogicalDisk(C:)\\Disk Write Bytes/sec", perfmon.FloatFlormat, "")
	if err != nil && err != perfmon.PDH_NO_MORE_DATA {
		return nil, err
	}

	if err = query.Execute(); err != nil {
		return nil, err
	}

	_, actualRawValue, err := perfmon.PdhGetRawCounterValue(query.Counters["\\LogicalDisk(C:)\\Disk Write Bytes/sec"].Handle)
	if err != nil {
		return nil, err
	}

	value, err := perfmon.PdhCalculateCounterFromRawValue(query.Counters["\\LogicalDisk(C:)\\Disk Write Bytes/sec"].Handle, perfmon.PdhFmtDouble|perfmon.PdhFmtNoCap100, actualRawValue, m.oldRawValue)

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

	if !m.executed {
		m.executed = true
	}

	m.oldRawValue = actualRawValue

	if err != nil {
		return nil, err
	}

	//values = append(values, *(*float64)(unsafe.Pointer(&value.LongValue)))

	stats, err := disk.IOCounters()
	if err != nil {
		return nil, errors.Wrap(err, "disk io counters")
	}

	// open a sampling means sample the current cpu counter
	m.statistics.OpenSampling()

	events := make([]common.MapStr, 0, len(stats))
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

		extraMetrics, err := m.statistics.CalIOStatistics(counters)
		if err == nil {
			event["iostat"] = common.MapStr{
				"read": common.MapStr{
					"request": common.MapStr{
						"merges_per_sec": extraMetrics.ReadRequestMergeCountPerSec,
						"per_sec":        extraMetrics.ReadRequestCountPerSec,
					},
					"per_sec": common.MapStr{
						"bytes": extraMetrics.ReadBytesPerSec,
					},
				},
				"write": common.MapStr{
					"request": common.MapStr{
						"merges_per_sec": extraMetrics.WriteRequestMergeCountPerSec,
						"per_sec":        extraMetrics.WriteRequestCountPerSec,
					},
					"per_sec": common.MapStr{
						"bytes": extraMetrics.WriteBytesPerSec,
					},
				},
				"queue": common.MapStr{
					"avg_size": extraMetrics.AvgQueueSize,
				},
				"request": common.MapStr{
					"avg_size": extraMetrics.AvgRequestSize,
				},
				"await":        extraMetrics.AvgAwaitTime,
				"service_time": extraMetrics.AvgServiceTime,
				"busy":         extraMetrics.BusyPct,
			}
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
