package vm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"

	"bramandscript/object"
)

func getBytesFromObj(obj object.Object) ([]byte, error) {
	switch v := obj.(type) {
	case *object.Bytes:
		v.Mu.RLock()
		defer v.Mu.RUnlock()
		cp := make([]byte, len(v.Value))
		copy(cp, v.Value)
		return cp, nil
	case *object.String:
		return []byte(v.Value), nil
	default:
		return nil, fmt.Errorf("expected BYTES or STRING, got %s", obj.Type())
	}
}

func init() {
	RegisterBuiltin("gzip_compress", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)
			_, err = w.Write(data)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("gzip compression error: %v", err)}
			}
			if err := w.Close(); err != nil {
				return &object.Error{Message: fmt.Sprintf("gzip close error: %v", err)}
			}
			return object.NewBytes(buf.Bytes())
		},
	})

	RegisterBuiltin("gzip_decompress", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			r, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("gzip read error: %v", err)}
			}
			defer r.Close()
			decomp, err := io.ReadAll(r)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("gzip decompress error: %v", err)}
			}
			return object.NewBytes(decomp)
		},
	})

	RegisterBuiltin("zlib_compress", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			var buf bytes.Buffer
			w := zlib.NewWriter(&buf)
			_, err = w.Write(data)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("zlib compression error: %v", err)}
			}
			if err := w.Close(); err != nil {
				return &object.Error{Message: fmt.Sprintf("zlib close error: %v", err)}
			}
			return object.NewBytes(buf.Bytes())
		},
	})

	RegisterBuiltin("zlib_decompress", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("zlib read error: %v", err)}
			}
			defer r.Close()
			decomp, err := io.ReadAll(r)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("zlib decompress error: %v", err)}
			}
			return object.NewBytes(decomp)
		},
	})

	RegisterBuiltin("tar_pack", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			var buf bytes.Buffer
			tw := tar.NewWriter(&buf)

			switch v := args[0].(type) {
			case *object.Hash:
				v.Mu.RLock()
				defer v.Mu.RUnlock()
				for _, pair := range v.Pairs {
					fileName := pair.Key.Inspect()
					fileData, err := getBytesFromObj(pair.Value)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("file %s data error: %v", fileName, err)}
					}
					hdr := &tar.Header{
						Name: fileName,
						Mode: 0644,
						Size: int64(len(fileData)),
					}
					if err := tw.WriteHeader(hdr); err != nil {
						return &object.Error{Message: fmt.Sprintf("tar write header error: %v", err)}
					}
					if _, err := tw.Write(fileData); err != nil {
						return &object.Error{Message: fmt.Sprintf("tar write content error: %v", err)}
					}
				}
			case *object.Array:
				v.Mu.RLock()
				defer v.Mu.RUnlock()
				for _, el := range v.Elements {
					entry, ok := el.(*object.Hash)
					if !ok {
						continue
					}
					entry.Mu.RLock()
					namePair, hasName := entry.Pairs[(&object.String{Value: "name"}).HashKey()]
					dataPair, hasData := entry.Pairs[(&object.String{Value: "data"}).HashKey()]
					entry.Mu.RUnlock()
					if !hasName || !hasData {
						continue
					}
					fileName := namePair.Value.Inspect()
					if strName, isStr := namePair.Value.(*object.String); isStr {
						fileName = strName.Value
					}
					fileData, err := getBytesFromObj(dataPair.Value)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					hdr := &tar.Header{
						Name: fileName,
						Mode: 0644,
						Size: int64(len(fileData)),
					}
					if err := tw.WriteHeader(hdr); err != nil {
						return &object.Error{Message: fmt.Sprintf("tar write header error: %v", err)}
					}
					if _, err := tw.Write(fileData); err != nil {
						return &object.Error{Message: fmt.Sprintf("tar write content error: %v", err)}
					}
				}
			default:
				return &object.Error{Message: "tar_pack requires HASH or ARRAY of file entries"}
			}

			if err := tw.Close(); err != nil {
				return &object.Error{Message: fmt.Sprintf("tar close error: %v", err)}
			}
			return object.NewBytes(buf.Bytes())
		},
	})

	RegisterBuiltin("tar_unpack", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			data, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			tr := tar.NewReader(bytes.NewReader(data))
			pairs := make(map[object.HashKey]object.HashPair)

			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("tar read entry error: %v", err)}
				}
				content, err := io.ReadAll(tr)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("tar read content error for %s: %v", hdr.Name, err)}
				}
				key := &object.String{Value: hdr.Name}
				val := object.NewBytes(content)
				pairs[key.HashKey()] = object.HashPair{Key: key, Value: val}
			}

			return &object.Hash{Pairs: pairs}
		},
	})
}
