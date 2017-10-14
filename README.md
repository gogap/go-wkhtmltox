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
		path = "/v1"
		
		cors {
			allowed-origins = ["*"]
		}

		gzip-enabled = true

		http {
			address = ":8080"
			enabled = true
		}

		https {
			address = ":443"
			enabled = false
			cert    = ""
			key     = ""
		}

		templates  {
			render-html {
				template = "templates/render_html.tmpl"
			}

			binary {
				template = "templates/binary.tmpl"
			}
		}
	}

	wkhtmltox {
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
```


### Request Args

Field|Values|Usage
:--|:--|:--
to|image,pdf|convert to
fetcher ||if is nil, converter.uri could not be empty, it will pass to wkhtmltox
fetcher.name||fetcher name in `app.conf`
fetcher.options ||different fetcher driver has different options
converter||the options for converter


### converter

the converter is the following json struct


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
		"to" : "image",
		"converter":{
			"uri": "https://www.bing.com"
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
		"to" : "pdf",
		"converter":{
			"uri": "https://www.bing.com"
		}
    }'
```

> if you enabled gzip, you should add arg `--compressed` to curl

#### Screenshot

![bing.com](https://github.com/gogap/repo-assets/raw/master/go-wkhtmltox/go-wkhtmltox-render-screenshot.png)

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
	"to" : "image",
	"converter":{
		"uri": "https://www.bing.com"
	},
	"template": "render-html"
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

#### Template funcs

Func|usage
:--|:--
base64Encode|encode value to base64 string
base64Decode|decode base64 string to string
jsonify|marshal object
md5|string md5 hash
toBytes|convert value to []byte
htmlEscape|for html safe
htmlUnescape|unescape html

#### Template Args

```go
type TemplateArgs struct {
	To string
	ConvertResponse
	Response *RespHelper
}

type ConvertResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}
```

#### Internal templates

> at templates dir


Name|Usage
:--|:--
 |default template, retrun `code`,`message`, `result`
render-html|render data to html
binary|you cloud use curl to download directly

##### use render-html

```bash
curl -X POST \
  http://IP:8080/v1/convert \
  -H 'accept-encoding: gzip' \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/json' \
  -d '{
	"to" : "image",
	"converter":{
		"uri": "https://www.bing.com"
	},
	"template": "render-html"
}' --compressed -o bing.html
```


##### use binary

```bash
curl -X POST \
  http://IP:8080/v1/convert \
  -H 'accept-encoding: gzip' \
  -H 'cache-control: no-cache' \
  -H 'content-type: application/json' \
  -d '{
	"to" : "image",
	"converter":{
		"uri": "https://www.bing.com"
	},
	"template": "binary"
}' --compressed -o bing.jpg
```


# Use this package as libary

TODO