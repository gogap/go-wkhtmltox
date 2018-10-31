package wkhtmltox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gogap/config"
	"github.com/pborman/uuid"

	"github.com/gogap/go-wkhtmltox/wkhtmltox/fetcher"
)

type ToFormat string

type Orientation string

const (
	Landscape Orientation = "Landscape"
	Portrait  Orientation = "Portrait"
)

type CropOptions struct {
	X int `json:"x"` // Set x coordinate for cropping
	Y int `json:"y"` // Set y coordinate for cropping
	H int `json:"h"` // Set height for cropping
	W int `json:"w"` // Set width for cropping
}

type ExtendParams map[string]string

func (p ExtendParams) toCommandArgs() []string {
	var args []string

	for k, v := range p {

		k = strings.TrimPrefix(k, "-")
		k = strings.TrimPrefix(k, "-")

		if len(k) > 0 {

			k = strings.Replace(k, "_", "-", -1)

			switch k {
			case "-", "q", "quiet", "V", "version", "H", "extended-help", "h", "help", "license":
				continue
			}

			args = append(args, "--"+k)

			if len(v) > 0 {
				args = append(args, v)
			}
		}
	}

	return args
}

type ConvertOptions interface {
	convertOptions()
	toCommandArgs() []string
	uri() string
}

type ToImageOptions struct {
	URI     string       `json:"uri"`
	Crop    CropOptions  `json:"crop"`    // Cropping options
	Format  string       `json:"format"`  // Image format, default is png
	Quality int          `json:"quality"` // Output image quality (between 0 and 100) (default 94)
	Width   int          `json:"width"`   // Default is 1024
	Height  int          `json:"height"`  // Set screen height (default is calculated from page content) (default 0)
	Extend  ExtendParams `json:"extend"`  // Other params
}

func (p *ToImageOptions) uri() string {
	return p.URI
}

func (*ToImageOptions) convertOptions() {}

func (p *ToImageOptions) toCommandArgs() []string {

	var args []string

	if p.Crop.H != 0 {
		args = append(args, []string{"--crop-h", strconv.Itoa(p.Crop.H)}...)
	}

	if p.Crop.W != 0 {
		args = append(args, []string{"--crop-w", strconv.Itoa(p.Crop.W)}...)
	}

	if p.Crop.X != 0 {
		args = append(args, []string{"--crop-x", strconv.Itoa(p.Crop.X)}...)
	}

	if p.Crop.Y != 0 {
		args = append(args, []string{"--crop-y", strconv.Itoa(p.Crop.Y)}...)
	}

	if len(p.Format) > 0 {
		args = append(args, []string{"--format", p.Format}...)
	}

	if p.Height != 0 {
		args = append(args, []string{"--height", strconv.Itoa(p.Height)}...)
	}

	if p.Width != 0 {
		args = append(args, []string{"--width", strconv.Itoa(p.Width)}...)
	}

	extArgs := p.Extend.toCommandArgs()

	args = append(args, extArgs...)

	return args

}

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

func (p *ToPDFOptions) uri() string {
	return p.URI
}

func (*ToPDFOptions) convertOptions() {}

func (p *ToPDFOptions) toCommandArgs() []string {
	var args []string

	if p.NoCollate {
		args = append(args, "--no-collate")
	}

	if p.Copies > 1 {
		args = append(args, []string{"--copies", strconv.Itoa(p.Copies)}...)
	}

	if p.GrayScale {
		args = append(args, "--gray-scale")
	}

	if p.LowQuality {
		args = append(args, "--low-quality")
	}

	if len(p.Orientation) > 0 {
		args = append(args, []string{"--orientation", string(p.Orientation)}...)
	}

	if len(p.PageSize) > 0 {
		args = append(args, []string{"--page-size", p.PageSize}...)
	}

	if p.PrintMediaType {
		args = append(args, "--print-media-type")
	}

	extArgs := p.Extend.toCommandArgs()

	args = append(args, extArgs...)

	return args
}

type FetcherOptions struct {
	Name   string          `json:"name"`   // http, oss, data
	Params json.RawMessage `json:"params"` // Optional
}

type WKHtmlToX struct {
	verbose  bool
	timeout  time.Duration
	fetchers map[string]fetcher.Fetcher
}

func New(conf config.Configuration) (wkHtmlToX *WKHtmlToX, err error) {

	wk := &WKHtmlToX{
		fetchers: make(map[string]fetcher.Fetcher),
	}

	commandTimeout := conf.GetTimeDuration("timeout", time.Second*300)

	wk.timeout = commandTimeout

	verbose := conf.GetBoolean("verbose", false)

	wk.verbose = verbose

	fetchersConf := conf.GetConfig("fetchers")

	if fetchersConf == nil || len(fetchersConf.Keys()) == 0 {
		wkHtmlToX = wk
		return
	}

	fetcherList := fetchersConf.Keys()

	for _, fName := range fetcherList {

		if len(fName) == 0 || fName == "default" {
			err = fmt.Errorf("fetcher name could not be '' or 'default'")
			return
		}

		_, exist := wk.fetchers[fName]

		if exist {
			err = fmt.Errorf("fetcher of %s already exist", fName)
			return
		}

		fetcherConf := fetchersConf.GetConfig(fName)
		fDriver := fetcherConf.GetString("driver")

		if len(fDriver) == 0 {
			err = fmt.Errorf("the fetcher of %s's driver is empty", fName)
			return
		}

		fOptions := fetcherConf.GetConfig("options")

		var f fetcher.Fetcher
		f, err = fetcher.New(fDriver, fOptions)

		if err != nil {
			return
		}

		wk.fetchers[fName] = f
	}

	wkHtmlToX = wk

	return
}

func (p *WKHtmlToX) Convert(fetcherOpts FetcherOptions, convertOpts ConvertOptions) (ret []byte, err error) {

	cmd := ""
	ext := ""

	switch o := convertOpts.(type) {
	case *ToImageOptions:
		{
			cmd = "wkhtmltoimage"
			ext = ".jpg"
			if len(o.Format) > 0 {
				ext = "." + o.Format
			}
		}
	case *ToPDFOptions:
		{
			cmd = "wkhtmltopdf"
			ext = ".pdf"
		}
	default:
		err = fmt.Errorf("unkown ConvertOptions type")
		return
	}

	inputMethod := convertOpts.uri()

	var data []byte

	if len(fetcherOpts.Name) > 0 && fetcherOpts.Name != "default" {

		data, err = p.fetch(fetcherOpts)
		if err != nil {
			return
		}

		inputMethod = "-"
	}

	if len(inputMethod) == 0 {
		err = fmt.Errorf("non input method could be use, please check your fetcher options or uri param")
		return
	}

	tmpDir, err := ioutil.TempDir("", "go-wkhtmltox")
	if err != nil {
		return
	}

	tmpfileName := filepath.Join(tmpDir, uuid.New()) + ext

	args := convertOpts.toCommandArgs()

	if p.verbose {
		args = append(args, []string{inputMethod, tmpfileName}...)
	} else {
		args = append(args, []string{"--quiet", inputMethod, tmpfileName}...)
	}

	var output []byte
	output, err = execCommand(p.timeout, data, cmd, args...)

	if p.verbose {
		if len(output) > 0 {
			fmt.Println("[wkhtmltox][DBG]", string(output))
		}

		if err != nil {
			fmt.Println("[wkhtmltox][ERR]", err)
		}
	}

	if err != nil {
		return
	}

	defer os.Remove(tmpfileName)

	var result []byte
	result, err = ioutil.ReadFile(tmpfileName)

	ret = result

	return
}

func (p *WKHtmlToX) fetch(fetcherOpts FetcherOptions) (data []byte, err error) {
	fetcher, exist := p.fetchers[fetcherOpts.Name]
	if !exist {
		err = fmt.Errorf("fetcher %s not exist", fetcherOpts.Name)
		return
	}

	data, err = fetcher.Fetch([]byte(fetcherOpts.Params))

	return
}
