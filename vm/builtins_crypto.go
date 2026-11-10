package vm

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha1"
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
					if s, ok := col.(*object.String); ok {
						rec = append(rec, s.Value)
					} else {
						rec = append(rec, col.Inspect())
					}
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
			} else if bytesObj, ok := args[0].(*object.Bytes); ok {
				bytesObj.Mu.RLock()
				data = make([]byte, len(bytesObj.Value))
				copy(data, bytesObj.Value)
				bytesObj.Mu.RUnlock()
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
			cleanStr := strings.TrimSpace(strObj.Value)
			data, err := base64.StdEncoding.DecodeString(cleanStr)
			if err != nil {
				data, err = base64.RawStdEncoding.DecodeString(cleanStr)
			}
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
			cleanStr := strings.TrimSpace(strObj.Value)
			data, err := base64.URLEncoding.DecodeString(cleanStr)
			if err != nil {
				data, err = base64.RawURLEncoding.DecodeString(cleanStr)
			}
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

	sha1Builtin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1 or 2", len(args))}
			}
			data := []byte(args[0].Inspect())
			if strObj, ok := args[0].(*object.String); ok {
				data = []byte(strObj.Value)
			} else if bytesObj, ok := args[0].(*object.Bytes); ok {
				bytesObj.Mu.RLock()
				data = make([]byte, len(bytesObj.Value))
				copy(data, bytesObj.Value)
				bytesObj.Mu.RUnlock()
			}
			h := sha1.Sum(data)
			if len(args) == 2 && isTruthy(args[1]) {
				return &object.String{Value: string(h[:])}
			}
			return &object.String{Value: hex.EncodeToString(h[:])}
		},
	}
	RegisterBuiltinAliases(sha1Builtin, "sha1", "crypto_sha1")

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

	// Helper to normalize AES 256-bit key
	normalizeKey := func(obj object.Object) ([]byte, error) {
		k, err := getBytesFromObj(obj)
		if err != nil {
			return nil, err
		}
		if len(k) == 64 {
			if decoded, err := hex.DecodeString(string(k)); err == nil && len(decoded) == 32 {
				return decoded, nil
			}
		}
		if len(k) == 32 || len(k) == 16 || len(k) == 24 {
			return k, nil
		}
		// Hash to 32 bytes using SHA-256 for consistent 256-bit key
		h := sha256.Sum256(k)
		return h[:], nil
	}

	RegisterBuiltin("aes_gcm_encrypt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 4 {
				return &object.Error{Message: "aes_gcm_encrypt requires 2 to 4 arguments: (key, plaintext, [nonce], [additional_data])"}
			}
			keyBytes, err := normalizeKey(args[0])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid key: %v", err)}
			}
			plainBytes, err := getBytesFromObj(args[1])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid plaintext: %v", err)}
			}

			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("aes cipher error: %v", err)}
			}
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("gcm error: %v", err)}
			}

			var nonce []byte
			if len(args) >= 3 && args[2] != Null {
				nonce, err = getBytesFromObj(args[2])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid nonce: %v", err)}
				}
			} else {
				nonce = make([]byte, gcm.NonceSize())
				_, err = crand.Read(nonce)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("nonce generation error: %v", err)}
				}
			}

			var aad []byte
			if len(args) >= 4 && args[3] != Null {
				aad, _ = getBytesFromObj(args[3])
			}

			ciphertext := gcm.Seal(nil, nonce, plainBytes, aad)
			combined := append(nonce, ciphertext...)

			pairs := make(map[object.HashKey]object.HashPair)
			setPair := func(k string, v object.Object) {
				sk := &object.String{Value: k}
				pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: v}
			}
			setPair("ciphertext", object.NewBytes(ciphertext))
			setPair("nonce", object.NewBytes(nonce))
			setPair("combined", object.NewBytes(combined))
			setPair("hex", &object.String{Value: hex.EncodeToString(combined)})
			setPair("base64", &object.String{Value: base64.StdEncoding.EncodeToString(combined)})

			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("aes_gcm_decrypt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 4 {
				return &object.Error{Message: "aes_gcm_decrypt requires 2 to 4 arguments: (key, ciphertext, [nonce], [additional_data])"}
			}
			keyBytes, err := normalizeKey(args[0])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid key: %v", err)}
			}
			cipherBytes, err := getBytesFromObj(args[1])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid ciphertext: %v", err)}
			}

			block, err := aes.NewCipher(keyBytes)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("aes cipher error: %v", err)}
			}
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("gcm error: %v", err)}
			}

			var nonce []byte
			actualCiphertext := cipherBytes
			if len(args) >= 3 && args[2] != Null {
				nonce, err = getBytesFromObj(args[2])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid nonce: %v", err)}
				}
			} else {
				// Nonce prepended in combined ciphertext
				if len(cipherBytes) < gcm.NonceSize() {
					return &object.Error{Message: "ciphertext too short to contain nonce"}
				}
				nonce = cipherBytes[:gcm.NonceSize()]
				actualCiphertext = cipherBytes[gcm.NonceSize():]
			}

			var aad []byte
			if len(args) >= 4 && args[3] != Null {
				aad, _ = getBytesFromObj(args[3])
			}

			plaintext, err := gcm.Open(nil, nonce, actualCiphertext, aad)
			if err != nil {
				return &object.Error{Message: "aes_gcm_decrypt authentication failed: invalid key or corrupted ciphertext"}
			}

			return object.NewBytes(plaintext)
		},
	})

	RegisterBuiltin("ed25519_keypair", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			pub, priv, err := ed25519.GenerateKey(crand.Reader)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("ed25519 key generation error: %v", err)}
			}
			pairs := make(map[object.HashKey]object.HashPair)
			setPair := func(k string, v object.Object) {
				sk := &object.String{Value: k}
				pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: v}
			}
			setPair("public_key", object.NewBytes(pub))
			setPair("private_key", object.NewBytes(priv))
			setPair("public_hex", &object.String{Value: hex.EncodeToString(pub)})
			setPair("private_hex", &object.String{Value: hex.EncodeToString(priv)})
			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("ed25519_sign", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: "ed25519_sign requires 2 arguments: (private_key, message)"}
			}
			privBytes, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid private_key: %v", err)}
			}
			if len(privBytes) == 128 {
				if dec, err := hex.DecodeString(string(privBytes)); err == nil && len(dec) == 64 {
					privBytes = dec
				}
			}
			if len(privBytes) != ed25519.PrivateKeySize {
				return &object.Error{Message: fmt.Sprintf("ed25519 private key must be %d bytes, got %d", ed25519.PrivateKeySize, len(privBytes))}
			}
			msgBytes, err := getBytesFromObj(args[1])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid message: %v", err)}
			}
			sig := ed25519.Sign(privBytes, msgBytes)
			return object.NewBytes(sig)
		},
	})

	RegisterBuiltin("ed25519_verify", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Message: "ed25519_verify requires 3 arguments: (public_key, message, signature)"}
			}
			pubBytes, err := getBytesFromObj(args[0])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid public_key: %v", err)}
			}
			if len(pubBytes) == 64 {
				if dec, err := hex.DecodeString(string(pubBytes)); err == nil && len(dec) == 32 {
					pubBytes = dec
				}
			}
			if len(pubBytes) != ed25519.PublicKeySize {
				return &object.Error{Message: fmt.Sprintf("ed25519 public key must be %d bytes, got %d", ed25519.PublicKeySize, len(pubBytes))}
			}
			msgBytes, err := getBytesFromObj(args[1])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid message: %v", err)}
			}
			sigBytes, err := getBytesFromObj(args[2])
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("invalid signature: %v", err)}
			}
			if len(sigBytes) == 128 {
				if dec, err := hex.DecodeString(string(sigBytes)); err == nil && len(dec) == 64 {
					sigBytes = dec
				}
			}
			if len(sigBytes) != ed25519.SignatureSize {
				return False
			}
			if ed25519.Verify(pubBytes, msgBytes, sigBytes) {
				return True
			}
			return False
		},
	})

	RegisterBuiltin("jwt_sign", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Message: "jwt_sign requires 2 or 3 arguments: (payload, secret_or_key, [alg])"}
			}
			alg := "HS256"
			if len(args) == 3 && args[2] != Null {
				if s, ok := args[2].(*object.String); ok {
					alg = strings.ToUpper(s.Value)
				}
			}

			header := map[string]interface{}{
				"alg": alg,
				"typ": "JWT",
			}
			hBytes, _ := json.Marshal(header)
			hB64 := base64.RawURLEncoding.EncodeToString(hBytes)

			nativePayload := bramandObjectToNative(args[0])
			pBytes, err := json.Marshal(nativePayload)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("jwt payload marshal error: %v", err)}
			}
			pB64 := base64.RawURLEncoding.EncodeToString(pBytes)

			signingInput := hB64 + "." + pB64

			var sigB64 string
			switch alg {
			case "HS256":
				secBytes, err := getBytesFromObj(args[1])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid secret: %v", err)}
				}
				mac := hmac.New(sha256.New, secBytes)
				mac.Write([]byte(signingInput))
				sigB64 = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			case "EDDSA":
				privBytes, err := getBytesFromObj(args[1])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid private key: %v", err)}
				}
				if len(privBytes) == 128 {
					if dec, err := hex.DecodeString(string(privBytes)); err == nil && len(dec) == 64 {
						privBytes = dec
					}
				}
				if len(privBytes) != ed25519.PrivateKeySize {
					return &object.Error{Message: "EdDSA requires 64-byte Ed25519 private key"}
				}
				sig := ed25519.Sign(privBytes, []byte(signingInput))
				sigB64 = base64.RawURLEncoding.EncodeToString(sig)
			default:
				return &object.Error{Message: fmt.Sprintf("unsupported jwt algorithm '%s'", alg)}
			}

			return &object.String{Value: signingInput + "." + sigB64}
		},
	})

	RegisterBuiltin("jwt_verify", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Message: "jwt_verify requires 2 or 3 arguments: (token, secret_or_key, [alg])"}
			}
			tokenStr, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: "first argument to jwt_verify must be STRING token"}
			}
			parts := strings.Split(tokenStr.Value, ".")
			if len(parts) != 3 {
				pairs := make(map[object.HashKey]object.HashPair)
				setPair := func(k string, v object.Object) {
					sk := &object.String{Value: k}
					pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: v}
				}
				setPair("valid", False)
				setPair("error", &object.String{Value: "invalid jwt format"})
				return &object.Hash{Pairs: pairs}
			}

			// Decode header
			hBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err != nil {
				pairs := make(map[object.HashKey]object.HashPair)
				sk := &object.String{Value: "valid"}
				pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: False}
				return &object.Hash{Pairs: pairs}
			}
			var headerMap map[string]interface{}
			_ = json.Unmarshal(hBytes, &headerMap)
			alg := "HS256"
			if a, ok := headerMap["alg"].(string); ok {
				alg = strings.ToUpper(a)
			}
			if len(args) == 3 && args[2] != Null {
				if s, ok := args[2].(*object.String); ok {
					alg = strings.ToUpper(s.Value)
				}
			}

			signingInput := parts[0] + "." + parts[1]
			sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
			if err != nil {
				pairs := make(map[object.HashKey]object.HashPair)
				sk := &object.String{Value: "valid"}
				pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: False}
				return &object.Hash{Pairs: pairs}
			}

			valid := false
			switch alg {
			case "HS256":
				secBytes, err := getBytesFromObj(args[1])
				if err == nil {
					mac := hmac.New(sha256.New, secBytes)
					mac.Write([]byte(signingInput))
					expected := mac.Sum(nil)
					valid = hmac.Equal(sigBytes, expected)
				}
			case "EDDSA":
				pubBytes, err := getBytesFromObj(args[1])
				if err == nil {
					if len(pubBytes) == 64 {
						if dec, err := hex.DecodeString(string(pubBytes)); err == nil && len(dec) == 32 {
							pubBytes = dec
						}
					}
					if len(pubBytes) == ed25519.PublicKeySize {
						valid = ed25519.Verify(pubBytes, []byte(signingInput), sigBytes)
					}
				}
			}

			// Decode payload
			pairs := make(map[object.HashKey]object.HashPair)
			setPair := func(k string, v object.Object) {
				sk := &object.String{Value: k}
				pairs[sk.HashKey()] = object.HashPair{Key: sk, Value: v}
			}
			if valid {
				setPair("valid", True)
			} else {
				setPair("valid", False)
			}
			setPair("header", nativeToBramandObject(headerMap))

			pBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err == nil {
				var payloadMap interface{}
				if err := json.Unmarshal(pBytes, &payloadMap); err == nil {
					setPair("payload", nativeToBramandObject(payloadMap))
				} else {
					setPair("payload", Null)
				}
			} else {
				setPair("payload", Null)
			}

			return &object.Hash{Pairs: pairs}
		},
	})
}
