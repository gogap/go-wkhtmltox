package wkhtmltox

import (
	"testing"
	"time"
)

func TestExecuteCommand(t *testing.T) {
	result, err := execCommand(time.Second*30, []byte(`http://www.qq.com`), "wkhtmltopdf", []string{"--quiet", "-", "-"}...)

	if err != nil {
		t.Error(err)
		return
	}

	if len(result) < 1000 {
		t.Error("covert page failure")
		return
	}
}
