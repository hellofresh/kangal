package proxy

import (
	"fmt"
	"net/url"
	"time"

	"github.com/thedevsaddam/govalidator"
)

func init() {
	govalidator.AddCustomRule("duration", func(field string, rule string, message string, value interface{}) error {
		_, err := time.ParseDuration(value.(string))
		if nil != err {
			return fmt.Errorf("The %s field must be a valid duration", field)
		}
		return nil
	})

	govalidator.AddCustomRule("http", func(field string, rule string, message string, value interface{}) error {
		_, err := url.Parse(value.(string))
		if nil != err {
			return fmt.Errorf("The %s field must be a valid URL", field)
		}
		return nil
	})
}
