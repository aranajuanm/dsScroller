package main

import (
	"github.com/Jeffail/gabs"
	"os"
	"fmt"
)


func main() {
	token := "68b9c340c025bcad5d06ccd6a280d1ca901e3230ad82bfd00ee7b1907ef8b1d8"
	//http://mercadopago-read.dsapi.melifrontends.com/entities/ds_payments_v1__openplatform_payments_api/search
	url := "http://apicore-read.dsapi.melifrontends.com/entities/orders__orders_search_api/search"
	size:= 500
	sleep:=1000
	body := `
{
        "query": {
    "and": [
      {"eq": {"field": "tags", "value": "reservation"}},
      {"in": {"field": "currency_id", "value": ["MXN","USD"]}},
      {"not":{"match": {"field": "seller.last_name", "value": "test"}}},
      {"exists": {"field": "feedback.sale"}},
      {"not":{"exists": {"field": "feedback.purchase"}}},
      {"date_range": { "field": "date_created", "gt": "2018-12-01", "lt": "now", "format": "YYYY-MM-dd", "time_zone": "-04:00" }},
      {"not":{"in":{"field": "status","value":["cancelled"]}}} 
    ]
  },
  "projections": ["id","feedback.sale.id","seller.id","buyer.id"],
        "type":"scroll"
}
`

	file, err := os.Create("export.csv")
	check(err)
	defer file.Close()

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	check(err)

	jsonParsed.Set("scroll","type")
	jsonParsed.Set(size,"size")

	process(url,token,jsonParsed, sleep, func(response []*gabs.Container) error{
		idList := ""
		for _, child := range response {
			msj := fmt.Sprintf(`{"msg":{"feedback_id": %.0f, "order_id": %.0f,"from": %.0f,"to": %.0f,"role": "seller","site_id": "MLM","item_id": 1234,"headers": {"action":"insert"}}}`,
				child.Path("feedback.sale.id").Data().(float64),
				child.Path("id").Data().(float64),
				child.Path("seller.id").Data().(float64),
				child.Path("buyer.id").Data().(float64),
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