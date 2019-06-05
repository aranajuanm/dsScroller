# DSScroller

## Introduction

Scroll DS and dump to file

## Installation

``` bash
$ git clone
$ dep ensure
```

## How to use it

1. Edit main.go
2. Set your fury token (token variable)
3. Set your read proxy (url variable) (see https://meli.facebook.com/groups/537713793068124/permalink/1104330106406487/)
4. Set your query (body variable)
5. Set your output file (fileName variable)
6. Update csv column generator (check your query projections!!!)
7. Run it: `go run *.go`

### Examples of how to edit the column generator

ie:

``` go
msj := fmt.Sprintf(`%.0f,%f`, //character representation
child.Path("id").Data().(float64), //how the data is going to be parsed
child.Path("amount").Data().(float64),
)
```

Or:

``` go
msj := fmt.Sprintf(`%.0f,%s`,
child.Path("id").Data().(float64),
child.Path("other_string_field").Data().(string),
)
```
