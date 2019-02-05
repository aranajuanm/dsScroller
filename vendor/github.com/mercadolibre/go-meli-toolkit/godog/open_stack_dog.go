/**
 * @author mlabarinas
 */

package godog

import (
	"bytes"
	"errors"

	"log"

	"github.com/streamrail/concurrent-map"

	"os"
	"sort"
	"sync"
)

type OpenStackDogClient struct{}

type MetricMap struct {
	items cmap.ConcurrentMap
	sync.RWMutex
}

var metrics *MetricMap = &MetricMap{items: cmap.New()}

func (o *OpenStackDogClient) RecordSimpleMetric(metricName string, value float64, tags ...string) {
	recordMetric(metricName, value, "S", tags...)
}

func (o *OpenStackDogClient) RecordCompoundMetric(metricName string, value float64, tags ...string) {
	recordMetric(metricName, value, "C", tags...)
}

func (o *OpenStackDogClient) RecordFullMetric(metricName string, value float64, tags ...string) {
	recordMetric(metricName, value, "F", tags...)
}

func (o *OpenStackDogClient) RecordSimpleTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	floatTime, result, error := takeTimeFloat(fn)
	recordMetric(metricName, floatTime, "S", tags...)
	return result, error
}

func (o *OpenStackDogClient) RecordCompoundTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	floatTime, result, error := takeTimeFloat(fn)
	recordMetric(metricName, floatTime, "C", tags...)
	return result, error
}

func (o *OpenStackDogClient) RecordFullTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	floatTime, result, error := takeTimeFloat(fn)
	recordMetric(metricName, floatTime, "F", tags...)
	return result, error
}

func getMetric(metricName string, tags ...string) (*Metric, error) {
	if key, error := createKey(metricName, tags...); error != nil {
		return nil, error

	} else {
		value, _ := metrics.items.Get(key)

		return value.(*Metric), nil
	}
}

func getMetricsCombinatoryCount() int {
	return metrics.items.Count()
}

func cloneMetrics() cmap.ConcurrentMap {
	metricsItemsCloned := cmap.New()

	metrics.Lock()

	defer metrics.Unlock()

	for metric := range metrics.items.Iter() {
		metricsItemsCloned.Set(metric.Key, metric.Val)
	}

	metrics.items = cmap.New()

	return metricsItemsCloned
}

func cleanMetrics() {
	metrics.Lock()

	defer metrics.Unlock()

	metrics.items = cmap.New()
}

func recordMetric(name string, value float64, class string, tags ...string) {
	key, error := createKey(name, tags...)

	if error != nil {
		log.Println(error)

		return
	}

	if metric, exists := metrics.items.Get(key); !exists {
		metric := new(Metric)
		metric.Init(name, tags, class)
		metric.AddValue(value)

		metrics.items.Set(key, metric)

	} else {
		metric.(*Metric).Lock()

		metric.(*Metric).SetClass(class)
		metric.(*Metric).AddValue(value)

		metric.(*Metric).Unlock()
	}
}

func createKey(name string, tags ...string) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(name)

	sort.Strings(tags)

	for _, tag := range tags {
		if !IsValidTag(tag) {
			return "", errors.New("Invalid tag discard metric")
		}

		buffer.WriteString(tag)
	}

	return buffer.String(), nil
}

func init() {
	if os.Getenv("GO_ENVIRONMENT") == "production" && os.Getenv("DATACENTER") != "AWS" {
		go start()
	}
}
