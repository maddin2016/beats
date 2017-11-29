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

type CounterType uint32

const (
	CounterTypeReadBytes CounterType = iota
	CounterTypeReadCount
	CounterTypeReadTime
	CounterTypeWriteBytes
	CounterTypeWriteCount
	CounterTypeWriteTime
)

var counterTypes = map[CounterType]string{
	CounterTypeReadBytes:  "read.bytes",
	CounterTypeReadCount:  "read.count",
	CounterTypeReadTime:   "read.time",
	CounterTypeWriteBytes: "write.bytes",
	CounterTypeWriteCount: "write.count",
	CounterTypeWriteTime:  "write.time",
}

func (cType CounterType) String() string {
	return counterTypes[cType]
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
	oldRawValue map[string]perfmon.PdhRawCounter
	executed    bool
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms := &MetricSet{
		BaseMetricSet: base,
		statistics:    NewDiskIOStat(),
		oldRawValue:   map[string]perfmon.PdhRawCounter{},
	}
	return ms, nil
}

// Fetch fetches disk IO metrics from the OS.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	events, err := m.Read()
	if err != nil {
		return nil, err
	}

	return events, nil

	// for _, counters := range stats {

	// 	event := common.MapStr{
	// 		"name": counters.Name,
	// 		"read": common.MapStr{
	// 			"count": *(*float64)(unsafe.Pointer(&value.LongValue)),
	// 			"time":  counters.ReadTime,
	// 			"bytes": counters.ReadBytes,
	// 		},
	// 		"write": common.MapStr{
	// 			"count": counters.WriteCount,
	// 			"time":  counters.WriteTime,
	// 			"bytes": counters.WriteBytes,
	// 		},
	// 		"io": common.MapStr{
	// 			"time": counters.IoTime,
	// 		},
	// 	}

	// 	events = append(events, event)

	// 	if counters.SerialNumber != "" {
	// 		event["serial_number"] = counters.SerialNumber
	// 	}
	// }

	// // open a sampling means store the last cpu counter
	// m.statistics.CloseSampling()
}

func (m *MetricSet) Read() ([]common.MapStr, error) {
	query, err := perfmon.NewQuery("")
	if err != nil {
		return nil, err
	}
	defer query.Close()

	counters := map[string]CounterType{
		"\\LogicalDisk(*)\\Disk Write Bytes/sec": CounterTypeWriteBytes,
		"\\LogicalDisk(*)\\Disk Read Bytes/sec":  CounterTypeReadBytes,
	}

	for k, _ := range counters {
		err = query.AddCounter(k, perfmon.FloatFlormat, "")
		if err != nil {
			return nil, err
		}
	}

	if err = query.Execute(); err != nil {
		return nil, err
	}

	events := make([]common.MapStr, 0, len(counters))
	event := common.MapStr{}

	for k, v := range counters {
		actualRawValues, err := perfmon.PdhGetRawCounterArray(query.Counters[k].Handle)
		if err != nil {
			return nil, err
		}

		//rtn := make(map[string][]RawValue, len(actualRawValues))

		for _, rawValue := range actualRawValues {
			// Filter _total and Harddisk
			if len(rawValue.Name) > 3 {
				continue
			}

			event.Put(rawValue.Name, common.MapStr{})

			oldName := k + "_" + rawValue.Name

			value, err := perfmon.PdhCalculateCounterFromRawValue(query.Counters[k].Handle, perfmon.PdhFmtDouble|perfmon.PdhFmtNoCap100, rawValue.Value, m.oldRawValue[oldName])
			m.oldRawValue[oldName] = rawValue.Value
			if err != nil {
				switch err {
				case perfmon.PDH_CALC_NEGATIVE_VALUE:
				case perfmon.PDH_CSTATUS_INVALID_DATA:
					if m.executed {
						return nil, err
					} else {
						continue
					}
				default:
					return nil, err
				}
			}

			event[rawValue.Name] = common.MapStr{
				v.String(): *(*float64)(unsafe.Pointer(&value.LongValue)),
			}
		}

	}
	events = append(events, event)

	if !m.executed {
		m.executed = true
	}

	return events, nil
}
