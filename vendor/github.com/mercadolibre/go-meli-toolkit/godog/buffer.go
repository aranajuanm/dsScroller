package godog

import (
	"strconv"
	"sync"
	"time"
)

const (
	CHAR_SEP = " "
	MAX_SIZE = 1000
)

var flushTime = 1000 * time.Millisecond

/*
DDBuffer aggregates metrics and periodically send them to datadog statsd.
*/
type DDBuffer struct {
	metrics, snapshot map[string]*ddMetric
	mutex             *sync.RWMutex
	ticker            *time.Ticker
	direct            int64
}

func CreateBuffer() *DDBuffer {
	buffer := &DDBuffer{
		metrics: make(map[string]*ddMetric),
		mutex:   &sync.RWMutex{},
		ticker:  time.NewTicker(flushTime),
	}
	go buffer.run()

	return buffer
}

func (b *DDBuffer) Count(name string, value float64, tags []string, rate float64) {
	if b.GetSize() <= MAX_SIZE {
		b.recordMetric(name, value, tags, rate, "C")
	} else {
		client.Count(name, int64(value), tags, rate)
		atomicInc(&b.direct)
	}
}

func (b *DDBuffer) Gauge(name string, value float64, tags []string, rate float64) {
	if b.GetSize() <= MAX_SIZE {
		b.recordMetric(name, value, tags, rate, "G")
	} else {
		client.Gauge(name, value, tags, rate)
		atomicInc(&b.direct)
	}

}

/*
GetSize return the number of map keys. It's a dirty read to avoid unnecesary locks
*/
func (b *DDBuffer) GetSize() int {
	return len(b.metrics)
}

func (b *DDBuffer) recordMetric(name string, value float64, tags []string, rate float64, ddType string) {
	key := getKey(name, tags)
	b.mutex.RLock()
	if _, ok := b.metrics[key]; ok {
		b.metrics[key].increment(value)
		b.mutex.RUnlock()
	} else {
		b.mutex.RUnlock()
		b.mutex.Lock()
		if _, ok := b.metrics[key]; ok {
			b.metrics[key].increment(value)
		} else {
			b.metrics[key] = createDDMetric(name, ddType, value, tags)
		}
		b.mutex.Unlock()
	}
}

func (b *DDBuffer) run() {
	for {
		<-b.ticker.C
		b.send()
	}
}

func (b *DDBuffer) send() {
	b.makeSnapshot()
	var effective, count, direct int64 = int64(len(b.snapshot)), 0, atomicGet(&b.direct)
	for _, m := range b.snapshot {
		if m.ddType == "C" {
			client.Count(m.name, int64(m.value), getTags(m.tags...), m.count)
		} else if m.ddType == "G" {
			client.Gauge(m.name, m.value/m.count, getTags(m.tags...), m.count)
		}
		count += int64(m.count)
	}
	if len(b.snapshot) > 0 || direct > 0 {
		tags := []string{GetRawTag("flush", flushTime.String()), GetRawTag("maxsize", strconv.Itoa(MAX_SIZE))}
		client.Count("godog.buffer.metrics.count", count, append(getTags(tags...), GetRawTag("buffered", "true")), 1)
		client.Count("godog.buffer.metrics.count", direct, append(getTags(tags...), GetRawTag("buffered", "false")), 1)
		client.Count("godog.buffer.metrics.effective", effective, append(getTags(tags...), GetRawTag("buffered", "true")), 1)
		client.Count("godog.buffer.metrics.effective", direct, append(getTags(tags...), GetRawTag("buffered", "false")), 1)
	}
}

func (b *DDBuffer) makeSnapshot() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.snapshot = b.metrics
	b.metrics = make(map[string]*ddMetric)
}
