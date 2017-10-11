package wkhtml

import (
	"encoding/json"
	"fmt"
	"github.com/gogap/config"
	"strconv"
	"strings"
	"time"

	"github.com/gogap/go-wkhtmltopdf/wkhtml/fetcher"
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

func (p ExtendParams) ToCommandArgs() []string {
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

func (*ToImageOptions) convertOptions() {}

func (p *ToImageOptions) ToCommandArgs() []string {

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

	extArgs := p.Extend.ToCommandArgs()

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

func (*ToPDFOptions) convertOptions() {}

func (p *ToPDFOptions) ToCommandArgs() []string {
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

	extArgs := p.Extend.ToCommandArgs()

	args = append(args, extArgs...)

	return args
}

type FetcherOptions struct {
	Name   string          `json:"name"`   // http, oss, data
	Params json.RawMessage `json:"params"` // Optional
}

type WKHtml struct {
	timeout  time.Duration
	fetchers map[string]fetcher.Fetcher
}

func New(conf config.Configuration) (wkHtml *WKHtml, err error) {

	wk := &WKHtml{
		fetchers: make(map[string]fetcher.Fetcher),
	}

	commandTimeout := conf.GetTimeDuration("timeout", time.Second*300)

	wk.timeout = commandTimeout

	fetchersConf := conf.GetConfig("fetchers")

	if fetchersConf == nil || len(fetchersConf.Keys()) == 0 {
		wkHtml = wk
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

	wkHtml = wk

	return
}

func (p *WKHtml) ToPDF(fetcherOpts FetcherOptions, convertOpts ToPDFOptions) (data []byte, err error) {
	return p.convert("wkhtmltopdf", fetcherOpts, convertOpts.URI, convertOpts.ToCommandArgs())
}

func (p *WKHtml) ToImage(fetcherOpts FetcherOptions, convertOpts ToImageOptions) (data []byte, err error) {
	return p.convert("wkhtmltoimage", fetcherOpts, convertOpts.URI, convertOpts.ToCommandArgs())
}

func (p *WKHtml) convert(cmd string, fetcherOpts FetcherOptions, uri string, args []string) (ret []byte, err error) {

	inputMethod := uri

	var data []byte

	if len(fetcherOpts.Name) > 0 && fetcherOpts.Name != "default" {

		data, err = p.fetch(fetcherOpts)
		if err != nil {
			return
		}

		inputMethod = "-"
	}

	if len(inputMethod) == 0 {
		err = fmt.Errorf("non input method could be use")
		return
	}

	args = append(args, []string{"--quiet", inputMethod, "-"}...)

	result, err := execCommand(p.timeout, data, cmd, args...)

	if err != nil {
		return
	}

	ret = result

	return
}

func (p *WKHtml) fetch(fetcherOpts FetcherOptions) (data []byte, err error) {
	fetcher, exist := p.fetchers[fetcherOpts.Name]
	if !exist {
		err = fmt.Errorf("fetcher %s not exist", fetcherOpts.Name)
		return
	}

	data, err = fetcher.Fetch([]byte(fetcherOpts.Params))

	return
}
