/**
 * @author mlabarinas
 */

package godog

import (
	"math"
	"regexp"
	"sync"
)

const tagPattern = "^[\\w\\.\\-_/]*:[\\w\\.\\-_/%]*$"

var tagValidator *regexp.Regexp

type Metric struct {
	name   string
	tags   []string
	class  string
	amount int64
	sum    float64
	min    float64
	max    float64
	sync.RWMutex
}

func (m *Metric) Init(name string, tags []string, class string) {
	m.name = name
	m.amount = 0
	m.sum = 0
	m.min = math.MaxFloat64
	m.max = 0

	if tags != nil {
		m.tags = tags
	}

	if class != "" {
		m.class = class
	}
}

func (m *Metric) GetName() string {
	return m.name
}

func (m *Metric) SetName(name string) {
	m.name = name
}

func (m *Metric) GetTags() []string {
	return m.tags
}

func (m *Metric) SetTags(tags []string) {
	m.tags = tags
}

func (m *Metric) GetClass() string {
	return m.class
}

func (m *Metric) SetClass(class string) {
	m.class = class
}

func (m *Metric) GetAmount() int64 {
	return m.amount
}

func (m *Metric) SetAmount(amount int64) {
	m.amount = amount
}

func (m *Metric) GetSum() float64 {
	return m.sum
}

func (m *Metric) SetSum(sum float64) {
	m.sum = sum
}

func (m *Metric) GetMin() float64 {
	return m.min
}

func (m *Metric) SetMin(min float64) {
	m.min = min
}

func (m *Metric) GetMax() float64 {
	return m.max
}

func (m *Metric) SetMax(max float64) {
	m.max = max
}

func (m *Metric) GetClassName(class string) string {
	switch class {
	case "S":
		return "simple"
	case "C":
		return "compound"
	case "F":
		return "full"
	default:
		return "unknown: " + class
	}
}

func (m *Metric) AddValue(value float64) {
	m.amount++
	m.sum += value
	m.max = math.Max(m.max, value)
	m.min = math.Min(m.min, value)
}

func IsValidTag(tag string) bool {
	return tagValidator.MatchString(tag)
}

func init() {
	tagValidator = regexp.MustCompile(tagPattern)
}
