package requests

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
)

func APIQueryParams[T any](request *http.Request) (*T, error) {
	var reference T

	params := request.URL.Query()
	parsed := new(T)
	valueOf := reflect.ValueOf(reference)

	for fieldIdx := range valueOf.NumField() {
		fieldType := valueOf.Type().Field(fieldIdx)

		tagValue, tagExists := fieldType.Tag.Lookup("form")
		if !tagExists {
			return nil, response.ErrMissingFormTag
		}

		split := strings.Split(tagValue, ",")

		if len(split) == 0 {
			return nil, response.ErrMissingValueForTag
		}

		// the first is the param tag name
		if params.Has(split[0]) {
			inp := params.Get(split[0])

			if fieldType.Type.Kind() != reflect.Pointer {
				return parsed, response.ErrNotAPointer
			}

			name := valueOf.Field(fieldIdx).Type().Elem().Name()
			switch name {
			case "string":
				reflect.ValueOf(parsed).Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(&inp))
			case "bool":
				boolVal := strings.ToLower(inp) == "true"
				reflect.ValueOf(parsed).Elem().FieldByName(fieldType.Name).Set(reflect.ValueOf(&boolVal))
			default:
				return nil, response.ErrUnexpectedDataType
			}
		}
	}

	return parsed, nil
}
