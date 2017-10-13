package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/gogap/config"
	"github.com/gogap/go-wkhtmltox/wkhtmltox"
	"github.com/gorilla/mux"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/rs/cors"
	"github.com/urfave/negroni"
)

import (
	_ "github.com/gogap/go-wkhtmltox/wkhtmltox/fetcher/http"
)

const (
	defaultTemplateText = `{"code":{{.Code}},"message":"{{.Message}}"{{if .Result}},"result":{{.Result|Jsonify}}{{end}}}`
)

var (
	htmlToX *wkhtmltox.WKHtmlToX

	renderTmpls = make(map[string]*template.Template)

	defaultTmpl *template.Template
)

type ConvertData struct {
	Data []byte `json:"data"`
}

type WKHtmlToXOptions struct {
	To        string                   `json:"to"`
	Fetcher   wkhtmltox.FetcherOptions `json:"fetcher"`
	Converter json.RawMessage          `json:"converter"`
	Template  string                   `json:"template"`
}

type ConvertArgs struct {
	Options WKHtmlToXOptions `json:"options"`
}

type ConvertResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func main() {

	var err error

	defer func() {
		if err != nil {
			fmt.Printf("[go-wkhtmltox]: %s\n", err.Error())
		}
	}()

	conf := config.NewConfig(
		config.ConfigFile("app.conf"),
	)

	serviceConf := conf.GetConfig("service")

	wkHtmlToXConf := conf.GetConfig("wkhtmltox")

	htmlToX, err = wkhtmltox.New(wkHtmlToXConf)

	if err != nil {
		return
	}

	// init templates

	defaultTmpl, err = template.New("default").Funcs(funcMap).Parse(defaultTemplateText)

	if err != nil {
		return
	}

	err = loadTemplates(
		serviceConf.GetConfig("templates"),
	)

	if err != nil {
		return
	}

	// init http server
	c := cors.New(
		cors.Options{
			AllowedOrigins:     serviceConf.GetStringList("cors.allowed-origins"),
			AllowedMethods:     serviceConf.GetStringList("cors.allowed-methods"),
			AllowedHeaders:     serviceConf.GetStringList("cors.allowed-headers"),
			ExposedHeaders:     serviceConf.GetStringList("cors.exposed-headers"),
			AllowCredentials:   serviceConf.GetBoolean("cors.allow-credentials"),
			MaxAge:             int(serviceConf.GetInt64("cors.max-age")),
			OptionsPassthrough: serviceConf.GetBoolean("cors.options-passthrough"),
			Debug:              serviceConf.GetBoolean("cors.debug"),
		},
	)

	r := mux.NewRouter()

	pathPrefix := serviceConf.GetString("path", "/")

	r.PathPrefix(pathPrefix).Path("/convert").
		Methods("POST").
		HandlerFunc(handleHtmlToX)

	n := negroni.Classic()

	n.Use(c) // use cors

	if serviceConf.GetBoolean("gzip-enabled", true) {
		n.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	n.UseHandler(r)

	enableHTTP := serviceConf.GetBoolean("http.enabled", true)
	enableHTTPS := serviceConf.GetBoolean("https.enabled", false)

	if enableHTTP {
		listenAddr := serviceConf.GetString("address", "127.0.0.1:8080")
		http.ListenAndServe(listenAddr, n)
	}

	if enableHTTPS {

		listenAddr := serviceConf.GetString("address", "127.0.0.1:443")
		certFile := serviceConf.GetString("https.cert")
		keyFile := serviceConf.GetString("https.key")

		http.ListenAndServeTLS(listenAddr, certFile, keyFile, n)
	}
}

func writeResp(rw http.ResponseWriter, converArgs ConvertArgs, resp ConvertResponse) {

	var tmpl *template.Template
	if len(converArgs.Options.Template) == 0 {
		tmpl = defaultTmpl
	} else {
		var exist bool

		tmpl, exist = renderTmpls[converArgs.Options.Template]
		if !exist {
			tmpl = defaultTmpl
		}
	}

	args := map[string]interface{}{
		"Code":    resp.Code,
		"Message": resp.Message,
		"Result":  resp.Result,
		"Header":  rw.Header(),
		"To":      converArgs.Options.To,
	}

	err := tmpl.Execute(rw, args)

	if err != nil {
		log.Println(err)
	}
}

func handleHtmlToX(rw http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)

	decoder.UseNumber()

	args := ConvertArgs{}

	err := decoder.Decode(&args)

	if err != nil {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, err.Error(), nil})
		return
	}

	if len(args.Options.Converter) == 0 {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, "options.converter is nil", nil})
		return
	}

	to := strings.ToUpper(args.Options.To)

	var opts wkhtmltox.ConvertOptions

	if to == "IMAGE" {
		opts = &wkhtmltox.ToImageOptions{}
	} else if to == "PDF" {
		opts = &wkhtmltox.ToPDFOptions{}
	} else {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, "argument of to is illegal (image|pdf)", nil})
		return
	}

	err = json.Unmarshal(args.Options.Converter, opts)

	if err != nil {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, err.Error(), nil})
		return
	}

	var convData []byte

	convData, err = htmlToX.Convert(args.Options.Fetcher, opts)

	if err != nil {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, err.Error(), nil})
		return
	}

	writeResp(rw, args, ConvertResponse{0, "", ConvertData{Data: convData}})

	return
}

func loadTemplates(tmplsConf config.Configuration) (err error) {
	if tmplsConf == nil {
		return
	}

	tmpls := tmplsConf.Keys()

	for _, name := range tmpls {

		file := tmplsConf.GetString(name + ".template")

		tmpl := template.New(name).Funcs(funcMap)

		var data []byte
		data, err = ioutil.ReadFile(file)

		if err != nil {
			return
		}

		tmpl, err = tmpl.Parse(string(data))

		if err != nil {
			return
		}

		renderTmpls[name] = tmpl
	}

	return
}
