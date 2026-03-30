package json

import (
	"encoding/json"
	"fmt"
)

func Unmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

// Allows to format a JSON string safely similar to fmt.Sprintf
func Format(template string, args ...interface{}) string {
	arg_idx := 0
	result := make([]byte, 0, len(template)*2)

	for i := 0; i < len(template); i++ {
		if template[i] == '%' && i < len(template) {
			switch template[i+1] {

			// The %s format means to just copy it in.
			case 's':
				if arg_idx < len(args) {
					arg := ToString(args[arg_idx])
					memcat(&result, []byte(arg))
					arg_idx++
					i++
				}

			case 'q', 'i', 'd':
				if arg_idx < len(args) {
					arg, err := json.Marshal(args[arg_idx])
					if err != nil {
						arg = []byte("null")
					}
					memcat(&result, arg)
					arg_idx++
					i++
				}

			default:
				i++
			}
		} else {
			result = append(result, template[i])
		}
	}

	return string(result)
}

func ToString(x interface{}) string {
	switch t := x.(type) {
	case string:
		return t

	case []byte:
		return string(t)

	default:
		return fmt.Sprintf("%v", x)
	}
}

func memcat(dest *[]byte, src []byte) {
	for _, c := range src {
		*dest = append(*dest, c)
	}
}
