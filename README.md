# DSScroller

## Introduction

> Scroll DS and dump to file

## Installation

> git clone<br/>
> dep ensure


## Code Samples

> Edit main.go<br/>
> Set your fury token (token variable)<br/>
> Set your read proxy (url variable) (see https://meli.facebook.com/groups/537713793068124/permalink/1104330106406487/)<br/>
> Set your query (body variable)<br/>
> Set your output file (fileName variable)<br/>
> Update csv column generator (check your query projections!!!)

Ej:
```
msj := fmt.Sprintf(`%.0f,MLM`,
				child.Path("id").Data().(float64),
			)
```

Or:

```
msj := fmt.Sprintf(`%.0f,%s`,
				child.Path("id").Data().(float64),
                child.Path("other_field").Data().(float64),
			)
```

> go run main.go