package vm

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("hex_encode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("hex_encode expects 1 argument, got %d", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			return &object.String{Value: hex.EncodeToString(data)}
		},
	})

	RegisterBuiltin("hex_decode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("hex_decode expects 1 argument, got %d", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("hex_decode argument must be STRING, got %s", args[0].Type())}
			}
			raw, err := hex.DecodeString(strings.TrimSpace(strObj.Value))
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("hex_decode failed: %s", err)}
			}
			return &object.String{Value: string(raw)}
		},
	})

	RegisterBuiltin("url_encode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("url_encode expects 1 argument, got %d", len(args))}
			}
			val := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				val = s.Value
			}
			return &object.String{Value: url.QueryEscape(val)}
		},
	})

	RegisterBuiltin("url_decode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("url_decode expects 1 argument, got %d", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("url_decode argument must be STRING, got %s", args[0].Type())}
			}
			unescaped, err := url.QueryUnescape(strObj.Value)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("url_decode failed: %s", err)}
			}
			return &object.String{Value: unescaped}
		},
	})

	RegisterBuiltin("url_parse_query", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("url_parse_query expects 1 argument, got %d", len(args))}
			}
			queryStr := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				queryStr = s.Value
			}
			if idx := strings.Index(queryStr, "?"); idx != -1 {
				queryStr = queryStr[idx+1:]
			}
			values, err := url.ParseQuery(queryStr)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("url_parse_query failed: %s", err)}
			}

			pairs := make(map[object.HashKey]object.HashPair)
			for k, vList := range values {
				keyObj := &object.String{Value: k}
				if len(vList) == 1 {
					pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: &object.String{Value: vList[0]}}
				} else {
					arr := make([]object.Object, len(vList))
					for i, item := range vList {
						arr[i] = &object.String{Value: item}
					}
					pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: &object.Array{Elements: arr}}
				}
			}
			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("url_build_query", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("url_build_query expects 1 argument, got %d", len(args))}
			}
			h, ok := args[0].(*object.Hash)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("url_build_query argument must be HASH, got %s", args[0].Type())}
			}
			h.Mu.RLock()
			defer h.Mu.RUnlock()

			vals := url.Values{}
			for _, pair := range h.Pairs {
				k := pair.Key.Inspect()
				if ks, ok := pair.Key.(*object.String); ok {
					k = ks.Value
				}
				switch v := pair.Value.(type) {
				case *object.Array:
					v.Mu.RLock()
					for _, elem := range v.Elements {
						valStr := elem.Inspect()
						if es, ok := elem.(*object.String); ok {
							valStr = es.Value
						}
						vals.Add(k, valStr)
					}
					v.Mu.RUnlock()
				default:
					valStr := v.Inspect()
					if vs, ok := v.(*object.String); ok {
						valStr = vs.Value
					}
					vals.Set(k, valStr)
				}
			}
			return &object.String{Value: vals.Encode()}
		},
	})

	RegisterBuiltin("csv_parse_records", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return &object.Error{Message: fmt.Sprintf("csv_parse_records expects 1 to 3 arguments, got %d", len(args))}
			}
			csvStr := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				csvStr = s.Value
			}
			delimiter := ','
			if len(args) >= 2 {
				if d, ok := args[1].(*object.String); ok && len(d.Value) > 0 {
					delimiter = rune(d.Value[0])
				}
			}
			hasHeaders := false
			if len(args) >= 3 {
				if b, ok := args[2].(*object.Boolean); ok {
					hasHeaders = b.Value
				}
			}

			r := csv.NewReader(strings.NewReader(csvStr))
			r.Comma = delimiter
			records, err := r.ReadAll()
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("csv_parse_records failed: %s", err)}
			}

			if !hasHeaders {
				rows := make([]object.Object, len(records))
				for i, rec := range records {
					cols := make([]object.Object, len(rec))
					for j, col := range rec {
						cols[j] = &object.String{Value: col}
					}
					rows[i] = &object.Array{Elements: cols}
				}
				return &object.Array{Elements: rows}
			}

			if len(records) == 0 {
				return &object.Array{Elements: []object.Object{}}
			}

			headers := records[0]
			var list []object.Object
			for _, rec := range records[1:] {
				pairs := make(map[object.HashKey]object.HashPair)
				for j, col := range rec {
					headerName := fmt.Sprintf("col_%d", j)
					if j < len(headers) {
						headerName = headers[j]
					}
					kObj := &object.String{Value: headerName}
					pairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: col}}
				}
				list = append(list, &object.Hash{Pairs: pairs})
			}
			return &object.Array{Elements: list}
		},
	})

	RegisterBuiltin("csv_format_records", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return &object.Error{Message: fmt.Sprintf("csv_format_records expects 1 to 3 arguments, got %d", len(args))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("first argument must be ARRAY, got %s", args[0].Type())}
			}
			arr.Mu.RLock()
			defer arr.Mu.RUnlock()

			delimiter := ','
			if len(args) >= 2 {
				if d, ok := args[1].(*object.String); ok && len(d.Value) > 0 {
					delimiter = rune(d.Value[0])
				}
			}

			var customHeaders []string
			if len(args) >= 3 {
				if hArr, ok := args[2].(*object.Array); ok {
					hArr.Mu.RLock()
					for _, el := range hArr.Elements {
						customHeaders = append(customHeaders, el.Inspect())
					}
					hArr.Mu.RUnlock()
				}
			}

			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			w.Comma = delimiter

			if len(arr.Elements) == 0 {
				if len(customHeaders) > 0 {
					_ = w.Write(customHeaders)
					w.Flush()
				}
				return &object.String{Value: buf.String()}
			}

			// Check first element to see if array of arrays or array of hashes
			first := arr.Elements[0]
			if h, ok := first.(*object.Hash); ok {
				// Array of hashes
				headers := customHeaders
				if len(headers) == 0 {
					h.Mu.RLock()
					for _, p := range h.Pairs {
						if ks, ok := p.Key.(*object.String); ok {
							headers = append(headers, ks.Value)
						} else {
							headers = append(headers, p.Key.Inspect())
						}
					}
					h.Mu.RUnlock()
					sort.Strings(headers)
				}
				_ = w.Write(headers)

				for _, elem := range arr.Elements {
					rowHash, ok := elem.(*object.Hash)
					if !ok {
						continue
					}
					rowHash.Mu.RLock()
					record := make([]string, len(headers))
					for j, header := range headers {
						k := (&object.String{Value: header}).HashKey()
						if p, ok := rowHash.Pairs[k]; ok {
							if s, ok := p.Value.(*object.String); ok {
								record[j] = s.Value
							} else {
								record[j] = p.Value.Inspect()
							}
						} else {
							record[j] = ""
						}
					}
					rowHash.Mu.RUnlock()
					_ = w.Write(record)
				}
			} else {
				// Array of arrays
				if len(customHeaders) > 0 {
					_ = w.Write(customHeaders)
				}
				for _, elem := range arr.Elements {
					if subArr, ok := elem.(*object.Array); ok {
						subArr.Mu.RLock()
						rec := make([]string, len(subArr.Elements))
						for j, cell := range subArr.Elements {
							if s, ok := cell.(*object.String); ok {
								rec[j] = s.Value
							} else {
								rec[j] = cell.Inspect()
							}
						}
						subArr.Mu.RUnlock()
						_ = w.Write(rec)
					}
				}
			}

			w.Flush()
			if err := w.Error(); err != nil {
				return &object.Error{Message: fmt.Sprintf("csv_format_records failed: %s", err)}
			}
			return &object.String{Value: buf.String()}
		},
	})
}
