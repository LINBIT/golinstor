package linstor

import (
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

type ListOpts struct {
	Page    int `url:"offset"`
	PerPage int `url:"limit"`
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	vs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = vs.Encode()
	return u.String(), nil
}
