package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"text/template"
)

var (
	funcMap = template.FuncMap{
		"Base64Encode": base64.StdEncoding.EncodeToString,
		"Base64Decode": base64.StdEncoding.DecodeString,
		"Jsonify":      jsonify,
		"MD5":          md5String,
	}
)

func md5String(f string) string {
	h := md5.New()
	h.Write([]byte(f))
	return hex.EncodeToString(h.Sum([]byte{}))
}

func jsonify(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
