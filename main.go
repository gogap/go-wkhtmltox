package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/gogap/config"
	"github.com/gogap/go-wkhtmltox/wkhtmltox"
	"github.com/gorilla/mux"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/rs/cors"
	"github.com/spf13/cast"
	"github.com/urfave/cli"
	"github.com/urfave/negroni"
)

import (
	_ "github.com/gogap/go-wkhtmltox/wkhtmltox/fetcher/http"
)

const (
	defaultTemplateText = `{"code":{{.Code}},"message":"{{.Message}}"{{if .Result}},"result":{{.Result|jsonify}}{{end}}}`
)

var (
	htmlToX *wkhtmltox.WKHtmlToX

	renderTmpls = make(map[string]*template.Template)

	defaultTmpl *template.Template
)

type ConvertData struct {
	Data []byte `json:"data"`
}

type ConvertArgs struct {
	To        string                   `json:"to"`
	Fetcher   wkhtmltox.FetcherOptions `json:"fetcher"`
	Converter json.RawMessage          `json:"converter"`
	Template  string                   `json:"template"`
}

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

func main() {

	var err error

	defer func() {
		if err != nil {
			log.Printf("[go-wkhtmltox]: %s\n", err.Error())
		}
	}()

	app := cli.NewApp()

	app.Usage = "A server for wkhtmltopdf and wkhtmltoimage"

	app.Commands = cli.Commands{
		cli.Command{
			Name:   "run",
			Usage:  "run go-wkhtmltox service",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config,c",
					Usage: "config filename",
					Value: "app.conf",
				},
				cli.StringFlag{
					Name:  "cwd",
					Usage: "change work dir",
				},
			},
		},
	}

	err = app.Run(os.Args)
}

func run(ctx *cli.Context) (err error) {

	cwd := ctx.String("cwd")
	if len(cwd) != 0 {
		err = os.Chdir(cwd)
	}

	if err != nil {
		return
	}

	configFile := ctx.String("config")

	conf := config.NewConfig(
		config.ConfigFile(configFile),
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

	wg := sync.WaitGroup{}

	if enableHTTP {
		wg.Add(1)

		listenAddr := serviceConf.GetString("http.address", "127.0.0.1:8080")
		go func() {
			defer wg.Done()
			e := http.ListenAndServe(listenAddr, n)
			if e != nil {
				log.Printf("[go-wkhtmltox]: %s\n", e.Error())
			}
		}()
	}

	if enableHTTPS {
		wg.Add(1)

		listenAddr := serviceConf.GetString("http.address", "127.0.0.1:443")
		certFile := serviceConf.GetString("https.cert")
		keyFile := serviceConf.GetString("https.key")

		go func() {
			defer wg.Done()
			e := http.ListenAndServeTLS(listenAddr, certFile, keyFile, n)
			if e != nil {
				log.Printf("[go-wkhtmltox]: %s\n", e.Error())
			}
		}()
	}

	wg.Wait()

	return
}

func writeResp(rw http.ResponseWriter, convertArgs ConvertArgs, resp ConvertResponse) {

	var tmpl *template.Template
	if len(convertArgs.Template) == 0 {
		tmpl = defaultTmpl
	} else {
		var exist bool

		tmpl, exist = renderTmpls[convertArgs.Template]
		if !exist {
			tmpl = defaultTmpl
		}
	}

	respHelper := newRespHelper(rw)

	args := TemplateArgs{
		To:              convertArgs.To,
		ConvertResponse: resp,
		Response:        respHelper,
	}

	buf := bytes.NewBuffer(nil)

	err := tmpl.Execute(buf, args)

	if err != nil {
		log.Println(err)
	}

	if !respHelper.Holding() {
		rw.Write(buf.Bytes())
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

	if len(args.Converter) == 0 {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, "converter is nil", nil})
		return
	}

	to := strings.ToUpper(args.To)

	var opts wkhtmltox.ConvertOptions

	if to == "IMAGE" {
		opts = &wkhtmltox.ToImageOptions{}
	} else if to == "PDF" {
		opts = &wkhtmltox.ToPDFOptions{}
	} else {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, "argument of to is illegal (image|pdf)", nil})
		return
	}

	err = json.Unmarshal(args.Converter, opts)

	if err != nil {
		writeResp(rw, args, ConvertResponse{http.StatusBadRequest, err.Error(), nil})
		return
	}

	var convData []byte

	convData, err = htmlToX.Convert(args.Fetcher, opts)

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

type RespHelper struct {
	rw   http.ResponseWriter
	hold bool
}

func newRespHelper(rw http.ResponseWriter) *RespHelper {
	return &RespHelper{
		rw:   rw,
		hold: false,
	}
}

func (p *RespHelper) SetHeader(key, value interface{}) error {
	k := cast.ToString(key)
	v := cast.ToString(value)

	p.rw.Header().Set(k, v)

	return nil
}

func (p *RespHelper) Hold(v interface{}) error {
	h := cast.ToBool(v)
	p.hold = h

	return nil
}

func (p *RespHelper) Holding() bool {
	return p.hold
}

func (p *RespHelper) Write(data []byte) error {
	p.rw.Write(data)
	return nil
}

func (p *RespHelper) WriteHeader(code interface{}) error {
	c, err := cast.ToIntE(code)
	if err != nil {
		return err
	}

	p.rw.WriteHeader(c)

	return nil
}
