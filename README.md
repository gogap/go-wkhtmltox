go-wkhtmltox
============

# Run as a service

## Run at local

```bash
> go get github.com/gogap/go-wkhtmltox
> cd $GOPATH/src/github.com/gogap/go-wkhtmltox
> go build
> ./go-wkhtmltox -c app.conf
```

## Run at docker

 ```bash
 docker pull gogap/go-wkhtmltox:latest
 docker run -it -d -p 8080:8080 xujinzheng/go-wkhtmltox:latest
 ```

> then you could access the 8080 port
> in osx, you could get the docker ip by command `docker-machine ip`, 
> and the access service by IP:8080

## Config

`app.conf`

```
{

	service {
		path = "/v1" // path prefix
		
		cors {
			allowed-origins = ["*"]
		}

		gzip-enabled = true


		// can start both http and https
		http {
			address = "0.0.0.0:8080"
			enabled = true
		}

		https {
			address = "0.0.0.0:443"
			enabled = false
			cert    = ""
			key     = ""
		}

		// it's the template for response
		templates  {
			render-data {
				template = "templates/render_data.tmpl"
			}
		}
	}

	wkhtmltox {
		// the way for fetch data, default is pass the url directly to wkhtmltox
		fetchers {
			f1 {
				driver = http
				options {}
			}
		}
	}
}
```


## API

```json
{
	"options": {
		"to" : "image",
		"fetcher": {
			"name": "http",
			"options": {
			}
		},
		"converter":{
			"uri": "https://www.bing.com"
		},
		"template": "render-data"
	}
}
```


### options

Field|Values|Usage
:--|:--|:--
to|image,pdf|convert to
fetcher ||if is nil, converter.uri could not be empty, it will pass to wkhtmltox
fetcher.name||fetcher name in `app.conf`
fetcher.options ||different fetcher driver has different options
converter||the options for converter


### converter

the options.converter is the following json struct


```json
{
  "uri":"https://www.bing.com",
   ...
}
```

#### ToImageOptions

```go
type ToImageOptions struct {
	URI     string       `json:"uri"`
	Crop    CropOptions  `json:"crop"`    // Cropping options
	Format  string       `json:"format"`  // Image format, default is png
	Quality int          `json:"quality"` // Output image quality (between 0 and 100) (default 94)
	Width   int          `json:"width"`   // Default is 1024
	Height  int          `json:"height"`  // Set screen height (default is calculated from page content) (default 0)
	Extend  ExtendParams `json:"extend"`  // Other params
}

type CropOptions struct {
	X int `json:"x"` // Set x coordinate for cropping
	Y int `json:"y"` // Set y coordinate for cropping
	H int `json:"h"` // Set height for cropping
	W int `json:"w"` // Set width for cropping
}
```

#### ToPDFOptions

```go
type ToPDFOptions struct {
	URI            string       `json:"uri"`
	NoCollate      bool         `json:"no_collate"`       // Collate when printing multiple copies, default is true. --collate or --no-collate
	Copies         int          `json:"copies"`           // Number of copies to print into the pdf default is 1
	GrayScale      bool         `json:"gray_scale"`       // PDF will be generated in grayscale
	LowQuality     bool         `json:"low_quality"`      // Generates lower quality pdf/ps. Useful to shrink the result document space
	Orientation    Orientation  `json:"orientation"`      // Set orientation to Landscape or Portrait (default Portrait)
	PageSize       string       `json:"page_size"`        // Set paper size to: A4, Letter, etc. (default A4)
	PrintMediaType bool         `json:"print_media_type"` // Use print media-type instead of screen. --print-media-type or --no-print-media-type
	Extend         ExtendParams `json:"extend"`           // Other params
}
```

> type ExtendParams map[string]string

### Use curl

#### To image

```bash
curl -X POST \
  http://IP:8080/v1/convert \
  -H 'accept-encoding: gzip' \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/json' \
  -d '{
	"options": {
		"to" : "image",
		"converter":{
			"uri": "https://www.bing.com"
		}
	}
}'
```



#### To pdf

```bash
curl -X POST \
  http://IP:8080/v1/convert \
  -H 'accept-encoding: gzip' \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/json' \
  -d '{
	"options": {
		"to" : "pdf",
		"converter":{
			"uri": "https://www.bing.com"
		}
	}
}'
```

### Template

The defualt template is 
`{"code":{{.Code}},"message":"{{.Message}}"{{if .Result}},"result":{{.Result|Jsonify}}{{end}}}`

response example:

```json
{"code":0,"message":"","result":{"data":"bGl.............}}
```


we could add `template` to render as different response, we have another example template named `render-data`


```json
{
	"options": {
		"to" : "image",
		"converter":{
			"uri": "https://www.bing.com"
		},
		"template": "render-data"
	}
}
```

the response is 

```html
<html>
	<body>
	     	<img src="data:image/jpeg;base64,bGl............"/> 
 	</body>
</html>
```

So, the template will render at brower directly. you could add more your templates



# Use this package as libary

TODO