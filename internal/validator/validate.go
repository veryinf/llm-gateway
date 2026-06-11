package validator

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var translations = map[string]string{
	"required":    "此字段为必填项",
	"is_in_invalid": "值不在允许的范围内",
	"is_json":     "必须是有效的 JSON",
}

// ValidationTranslate 将 ozzo-validation 错误翻译为中文
func ValidationTranslate(err error) error {
	if err == nil {
		return nil
	}

	if validationErrors, ok := err.(validation.Errors); ok {
		var msg string
		for field, fieldErr := range validationErrors {
			errMsg := fieldErr.Error()
			if translated, exists := translations[errMsg]; exists {
				errMsg = translated
			}
			if msg != "" {
				msg += "; "
			}
			msg += fmt.Sprintf("%s: %s", field, errMsg)
		}
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}

	return nil
}
