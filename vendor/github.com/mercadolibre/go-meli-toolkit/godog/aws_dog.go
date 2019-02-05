/**
 * @author mlabarinas
 */

package godog

import (
	"os"

	"github.com/mercadolibre/go-meli-toolkit/datadog-go/statsd"
)

const (
	ENDPOINT            string = "datadog:8125"
	DEFAULT_BUFFER_SIZE int    = 500
)

type AwsDogClient struct{}

var client *statsd.Client
var buffer *DDBuffer
var isProduction = os.Getenv("GO_ENVIRONMENT") == "production"

func (a *AwsDogClient) RecordSimpleMetric(metricName string, value float64, tags ...string) {
	if isProduction {
		buffer.Count(metricName, value, getTags(tags...), 1)
	}
}

func (a *AwsDogClient) RecordCompoundMetric(metricName string, value float64, tags ...string) {
	if isProduction {
		buffer.Gauge(metricName, value, getTags(tags...), 1)
	}
}

func (a *AwsDogClient) RecordFullMetric(metricName string, value float64, tags ...string) {
	if isProduction {
		client.TimeInMilliseconds(metricName, value, getTags(tags...), 1)
	}
}

func (a *AwsDogClient) RecordSimpleTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	time, result, error := takeTime(fn)

	if isProduction {
		buffer.Count(metricName, float64(time), getTags(tags...), 1)
	}

	return result, error
}

func (a *AwsDogClient) RecordCompoundTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	floatTime, result, error := takeTimeFloat(fn)

	if isProduction {
		buffer.Gauge(metricName, floatTime, getTags(tags...), 1)
	}

	return result, error
}

func (a *AwsDogClient) RecordFullTimeMetric(metricName string, fn action, tags ...string) (interface{}, error) {
	floatTime, result, error := takeTimeFloat(fn)

	if isProduction {
		client.TimeInMilliseconds(metricName, floatTime, getTags(tags...), 1)
	}

	return result, error
}

func getTags(tags ...string) []string {
	var defaultTags []string

	if os.Getenv("PLATFORM") != "" {
		defaultTags = append(defaultTags, GetRawTag("platform", os.Getenv("PLATFORM")))
	}
	if os.Getenv("APPLICATION") != "" {
		defaultTags = append(defaultTags, GetRawTag("application", os.Getenv("APPLICATION")))
	}
	if os.Getenv("DATACENTER") != "" {
		defaultTags = append(defaultTags, GetRawTag("datacenter", os.Getenv("DATACENTER")))
	}
	defaultTags = append(defaultTags, tags...)

	return defaultTags
}

func init() {
	if isProduction {
		c, error := statsd.NewBuffered(ENDPOINT, DEFAULT_BUFFER_SIZE)

		if error != nil {
			panic(error)
		}

		client = c
		buffer = CreateBuffer()
		return
	}
}
