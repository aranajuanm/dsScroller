package godog

import (
	"strings"
	"sync"
	"sync/atomic"
)

type ddMetric struct {
	name, ddType string
	value, count float64
	tags         []string
	mutex        *sync.Mutex
}

func (m *ddMetric) increment(value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.value += value
	m.count++
}

func createDDMetric(name, ddType string, value float64, tags []string) *ddMetric {
	return &ddMetric{
		name:   name,
		ddType: ddType,
		value:  value,
		count:  1,
		tags:   tags,
		mutex:  &sync.Mutex{},
	}
}

func getKey(metric string, tags []string) string {
	return metric + CHAR_SEP + strings.Join(tags, CHAR_SEP)
}

func atomicInc(value *int64) {
	atomic.AddInt64(value, 1)
}

func atomicGet(value *int64) int64 {
	return atomic.SwapInt64(value, 0)
}
