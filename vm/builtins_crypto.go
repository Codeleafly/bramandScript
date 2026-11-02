package vm

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("json_parse", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `json_parse` must be STRING, got %s", args[0].Type())}
			}
			var val interface{}
			err := json.Unmarshal([]byte(strObj.Value), &val)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("json_parse error: %s", err)}
			}
			return nativeToBramandObject(val)
		},
	})

	RegisterBuiltin("json_stringify", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
			}
			native := bramandObjectToNative(args[0])
			var data []byte
			var err error
			if len(args) == 2 {
				indentSpaces := 2
				if intObj, ok := args[1].(*object.Integer); ok && intObj.Value >= 0 {
					indentSpaces = int(intObj.Value)
				}
				indent := strings.Repeat(" ", indentSpaces)
				data, err = json.MarshalIndent(native, "", indent)
			} else {
				data, err = json.Marshal(native)
			}
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("json_stringify error: %s", err)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("csv_parse", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("first argument to `csv_parse` must be STRING, got %s", args[0].Type())}
			}
			r := csv.NewReader(strings.NewReader(strObj.Value))
			if len(args) == 2 {
				if delimObj, ok := args[1].(*object.String); ok && len(delimObj.Value) > 0 {
					r.Comma = rune(delimObj.Value[0])
				}
			}
			records, err := r.ReadAll()
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("csv_parse error: %s", err)}
			}
			rows := make([]object.Object, len(records))
			for i, rec := range records {
				cols := make([]object.Object, len(rec))
				for j, col := range rec {
					cols[j] = &object.String{Value: col}
				}
				rows[i] = &object.Array{Elements: cols}
			}
			return &object.Array{Elements: rows}
		},
	})

	RegisterBuiltin("csv_stringify", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("first argument to `csv_stringify` must be ARRAY, got %s", args[0].Type())}
			}
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			if len(args) == 2 {
				if delimObj, ok := args[1].(*object.String); ok && len(delimObj.Value) > 0 {
					w.Comma = rune(delimObj.Value[0])
				}
			}
			for _, rowObj := range arr.Elements {
				rowArr, ok := rowObj.(*object.Array)
				if !ok {
					return &object.Error{Message: "csv rows must be arrays"}
				}
				var rec []string
				for _, col := range rowArr.Elements {
					rec = append(rec, col.Inspect())
				}
				if err := w.Write(rec); err != nil {
					return &object.Error{Message: fmt.Sprintf("csv_stringify error: %s", err)}
				}
			}
			w.Flush()
			return &object.String{Value: buf.String()}
		},
	})

	RegisterBuiltin("base64_encode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			return &object.String{Value: base64.StdEncoding.EncodeToString(data)}
		},
	})

	RegisterBuiltin("base64_decode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `base64_decode` must be STRING, got %s", args[0].Type())}
			}
			data, err := base64.StdEncoding.DecodeString(strObj.Value)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("base64_decode error: %s", err)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("base64_url_encode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			return &object.String{Value: base64.URLEncoding.EncodeToString(data)}
		},
	})

	RegisterBuiltin("base64_url_decode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `base64_url_decode` must be STRING, got %s", args[0].Type())}
			}
			data, err := base64.URLEncoding.DecodeString(strObj.Value)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("base64_url_decode error: %s", err)}
			}
			return &object.String{Value: string(data)}
		},
	})

	sha256Builtin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			h := sha256.Sum256(data)
			return &object.String{Value: hex.EncodeToString(h[:])}
		},
	}
	RegisterBuiltinAliases(sha256Builtin, "sha256", "crypto_sha256")

	RegisterBuiltin("sha512", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			h := sha512.Sum512(data)
			return &object.String{Value: hex.EncodeToString(h[:])}
		},
	})

	RegisterBuiltin("md5", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			h := md5.Sum(data)
			return &object.String{Value: hex.EncodeToString(h[:])}
		},
	})

	hmacBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
			}
			key := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				key = []byte(strObj.Value)
			}
			data := []byte(args[1].Inspect())
			if strObj, ok := args[1].(*object.String); ok {
				data = []byte(strObj.Value)
			}
			mac := hmac.New(sha256.New, key)
			mac.Write(data)
			return &object.String{Value: hex.EncodeToString(mac.Sum(nil))}
		},
	}
	RegisterBuiltinAliases(hmacBuiltin, "hmac_sha256", "crypto_hmac")

	RegisterBuiltin("random_bytes", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			count := 16
			if len(args) > 0 {
				if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
					count = int(intObj.Value)
				}
			}
			b := make([]byte, count)
			_, err := crand.Read(b)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("random_bytes error: %s", err)}
			}
			return &object.String{Value: hex.EncodeToString(b)}
		},
	})

	RegisterBuiltin("random_int", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var minVal, maxVal int64
			if len(args) == 1 {
				minVal = 0
				if intObj, ok := args[0].(*object.Integer); ok {
					maxVal = intObj.Value
				} else {
					return &object.Error{Message: "max argument must be INTEGER"}
				}
			} else if len(args) >= 2 {
				if intObj1, ok := args[0].(*object.Integer); ok {
					minVal = intObj1.Value
				}
				if intObj2, ok := args[1].(*object.Integer); ok {
					maxVal = intObj2.Value
				}
			} else {
				return &object.Error{Message: "wrong number of arguments to `random_int`"}
			}

			if maxVal < minVal {
				minVal, maxVal = maxVal, minVal
			}
			diff := maxVal - minVal + 1
			n, err := crand.Int(crand.Reader, big.NewInt(diff))
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("random_int error: %s", err)}
			}
			return &object.Integer{Value: minVal + n.Int64()}
		},
	})

	RegisterBuiltin("uuid", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var b [16]byte
			_, err := crand.Read(b[:])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("uuid error: %s", err)}
			}
			// RFC 4122 variant & version 4
			b[6] = (b[6] & 0x0f) | 0x40
			b[8] = (b[8] & 0x3f) | 0x80
			uuidStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
				b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
			return &object.String{Value: uuidStr}
		},
	})
}
