# Godog

Godog is a golang library that allows to collect and aggregate arbitrary metrics and send them to datadog

```diff
- If you are using Datadog in Fury, an environment variable called "DATACENTER" needs to be
- set with "AWS" value.
```

## Log metrics
With every metric you log, the name of the metric and a value are required. As a standard, every business metric names should start with "business.", and every lower-level, more technical metrics should start with "application.". Also you can send tags to break down the metric, entered using the "tag-key:tag-value" format. The amount of tags is limited by a combinatorial value of 10k. Since the client adds the hostname tag by default, the amount of servers will afect this limit, as well as any other variable introduced as a tag.

There are 3 types of metrics to use, and they only differ from each other when you see them in the metric explorer. It's very important to **always use the same metric type for each metric name**

### "Simple" metrics
These metrics only add the value you send as "<metric_name>.sum"
```go
//Increments the metric by 1, with the tag key "site_id" and the tag value "MLA"
godog.RecordSimpleMetric("business.items.visits", 1, "site_id:MLA")

//Increments the metric by 5, with the tag key "site_id" and the tag value "MLC"
godog.RecordSimpleMetric("business.items.visits", 5, "site_id:MLC")

//Increments the metric by 1, without any tag
godog.RecordSimpleMetric("application.requests", 1)

//Increments the metric by time function, with the tag key "site_id" and the tag value "MLA"
result, error := godog.RecordSimpleTimeMetric("business.test", func() (interface{}, error) {
    return "test", nil
}, "site_id:MLA")
```
### "Compound" metrics
These metrics add the value you send as "<metric_name>.sum" and the amount of metrics you logged as "<metric_name>.qty". This metric is useful when you need an average value (dividing both metrics in datadog's frontend)
```go
//Increments the metric by 250 and the qty by 1, without tags
godog.RecordCompoundMetric("business.questions.answers.time", 250)

//Increments the metric by 250 and the qty by 1, with the tags "site_id" and "browser"
godog.RecordCompoundMetric("business.questions.answers.time", 250, "site_id:MLA", "browser:mobile")

//Increments the metric by time function and the qty by 1, with the tag key "site_id" and the tag value "MLA"
result, error := godog.RecordCompoundTimeMetric("business.test", func() (interface{}, error) {
    return "test", nil
}, "site_id:MLA")
```
### "Full" metrics
These metrics add the value you send as "<metric_name>.sum", the amount of metrics you logged as "<metric_name>.qty", and also the minimum and maximum values as "<metric_name>.min" and "<metric_name>.max"
```go
//Increments the metric by 30 and the qty by 1, it also saves 30 as max and/or min value of the interval, if that condition is met
godog.RecordFullMetric("business.si.price", 30)

//Increments the metric by 30 and the qty by 1 with the tag "payment_type", it also saves 30 as max and/or min value of the interval, if that condition is met
godog.RecordFullMetric("business.si.price", 30, "payment_type:credit_card")

//Increments the metric by time function and the qty by 1 with the tag "site_id", it also saves 1 as max and/or min value of the interval, if that condition is met
result, error := godog.RecordFullTimeMetric("business.test", func() (interface{}, error) {
    return "test", nil
}, "site_id:MLA")
```

## Questions?

[fury@mercadolibre.com](fury@mercadolibre.com)