package main

import (
	"github.com/Jeffail/gabs"
	"os"
	"fmt"
)


func main() {
	token := "TOKEN_FURY"
	url := "https://read-services-proxy.furycloud.io/applications/mpcs-movements/ds/services/ds-movements-v1/search"
	size:= 500
	sleep:=1000
	body := `
{
    "query": {
      "and":[
     
        {"date_range": { "field": "date_created", "gt": "2019-01-01", "lt": "2019-02-20", "format": "YYYY-MM-dd", "time_zone": "-04:00" }},
        {"eq": {
            "field": "status",
            "value": "unavailable"
        }},
        {"not":{"exists": {"field": "date_released"}}}
      ]
        
    },
	"projections": ["id"],
    "type": "scroll",
    "secondary_search":true,
    "size": 10
}
`
	fileName:="export.csv"

	file, err := os.Create(fileName)
	check(err)
	defer file.Close()

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	check(err)

	jsonParsed.Set("scroll","type")
	jsonParsed.Set(size,"size")

	process(url,token,jsonParsed, sleep, func(response []*gabs.Container) error{
		idList := ""
		for _, child := range response {
			msj := fmt.Sprintf(`%.0f,MLM`,
				child.Path("id").Data().(float64),
			)
			idList = idList + msj + "\n"
		}
		_,err := file.WriteString(idList)
		check(err)

		return nil
	})



}

func check(e error) {
	if e != nil {
		panic(e)
	}
}