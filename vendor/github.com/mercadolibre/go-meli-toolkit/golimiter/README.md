# Rate Limiter

Throughput rate limiter for Go

# Import

```go
import "github.com/mercadolibre/go-meli-toolkit/golimiter"
```

# Building

Currently the only available type of rate limiter provided is based on the [Leaky Bucket Algorithm](https://en.wikipedia.org/wiki/Leaky_bucket).

An instance of `Limiter` will handle rate limiting for a single resource. 
It provides a bucket that will have a fixed number of tokens which will be consumed on every request, and will be refilled on a specified basis. 

```go
limiter := golimiter.New(3000, 100 * time.Millisecond)
```

In this particular example, we created a rate limiter with 100ms width buckets and a 3000 rpm limit. 

That means every 100ms we can consume 5 tokens, that will be reset accordingly.

# Operation

Once we've built a rate limiter instance we must send through it every request for its managed resources.

```go
func originalAction() (interface{}, error) {
	//...
}

result, err := limiter.Action(10, originalAction)
```

In this example we asked the limiter to substract 10 tokens of this time slice (bucket) for a specific resource. 
If it's unable to provide this amount, it'll return an error with message 'over quota'.

As every request might weight differently, we can substract different amounts of tokens accordingly.

## Questions?

[fury@mercadolibre.com](fury@mercadolibre.com)