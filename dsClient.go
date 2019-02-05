package main

import (
	"github.com/Jeffail/gabs"
	"net/http"
	"github.com/mercadolibre/go-meli-toolkit/restful/rest"
	"time"
	"github.com/mercadolibre/go-meli-toolkit/restful/rest/retry"
	"fmt"
)

var restDsClient *rest.RequestBuilder

func process(url string, token string,request *gabs.Container, sleep int, callback func(response []*gabs.Container) error) {
	restDsClient.Headers = make(http.Header)
	restDsClient.Headers.Add("x-auth-token",token)
	restDsClient.Headers.Add("Content-Type","application/json")
	fmt.Printf("/")
	for {
		rBytes:=request.Bytes()
		response := restDsClient.Post(url,rBytes)

		if response.Err != nil{
			fmt.Println("LAST SCROLL:")
			fmt.Println(request.Path("scroll_id").Data().(string))
			panic(response.Err)
		}
		if response.StatusCode != http.StatusOK  {
			panic(response)
		}

		responseParsed, err := gabs.ParseJSON(response.Bytes())
		check(err)

		children, _ := responseParsed.S("documents").Children()

		if children == nil || len(children)==0{
			return
		}
		request.Set(responseParsed.Path("scroll_id").Data().(string),"scroll_id")
		request.Delete("size")

		callback(children)
		fmt.Printf("*")
		time.Sleep(time.Duration(sleep) * time.Millisecond)

	}
	fmt.Printf("/")

}


func init() {
	customPool := &rest.CustomPool{
		MaxIdleConnsPerHost: 4,
	}

	restDsClient = &rest.RequestBuilder{
		Timeout:        3*time.Second,
		ContentType:    rest.BYTES,
		DisableTimeout: true,
		EnableCache:    false,
		CustomPool:     customPool,
		RetryStrategy:  retry.NewSimpleRetryStrategy(3, 1000*time.Millisecond),
	}
}
