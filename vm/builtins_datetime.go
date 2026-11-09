package vm

import (
	"fmt"
	"strings"
	"time"

	"bramandscript/object"
)

// convertToLayout translates common formatting tokens into Go layout.
func convertToLayout(fmtStr string) string {
	r := strings.NewReplacer(
		"YYYY", "2006",
		"yyyy", "2006",
		"YY", "06",
		"yy", "06",
		"MM", "01",
		"DD", "02",
		"dd", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"SSS", "000",
		"a", "pm",
		"A", "PM",
		"Z", "Z07:00",
	)
	return r.Replace(fmtStr)
}

func timeToHash(t time.Time) *object.Hash {
	pairs := make(map[object.HashKey]object.HashPair)
	setField := func(k string, v object.Object) {
		keyObj := &object.String{Value: k}
		pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: v}
	}

	setField("year", &object.Integer{Value: int64(t.Year())})
	setField("month", &object.Integer{Value: int64(t.Month())})
	setField("day", &object.Integer{Value: int64(t.Day())})
	setField("hour", &object.Integer{Value: int64(t.Hour())})
	setField("minute", &object.Integer{Value: int64(t.Minute())})
	setField("second", &object.Integer{Value: int64(t.Second())})
	setField("millisecond", &object.Integer{Value: int64(t.Nanosecond() / 1e6)})
	setField("unix_ms", &object.Integer{Value: t.UnixMilli()})
	setField("iso", &object.String{Value: t.Format(time.RFC3339)})
	setField("weekday", &object.String{Value: t.Weekday().String()})
	setField("weekday_num", &object.Integer{Value: int64(t.Weekday())})

	_, offsetSec := t.Zone()
	setField("offset", &object.Integer{Value: int64(offsetSec)})

	return &object.Hash{Pairs: pairs}
}

func parseTimeToTime(s, layout string) (time.Time, error) {
	if layout != "" {
		goLayout := convertToLayout(layout)
		return time.Parse(goLayout, s)
	}

	// Try standard ISO layouts
	candidates := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"15:04:05",
		time.RFC1123Z,
		time.RFC1123,
	}

	for _, cand := range candidates {
		if t, err := time.Parse(cand, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse datetime string: %s", s)
}

func getUnixMsFromArg(arg object.Object) (int64, error) {
	switch v := arg.(type) {
	case *object.Integer:
		return v.Value, nil
	case *object.Float:
		return int64(v.Value), nil
	case *object.Hash:
		v.Mu.RLock()
		defer v.Mu.RUnlock()
		key := (&object.String{Value: "unix_ms"}).HashKey()
		if p, ok := v.Pairs[key]; ok {
			if intVal, ok := p.Value.(*object.Integer); ok {
				return intVal.Value, nil
			} else if fltVal, ok := p.Value.(*object.Float); ok {
				return int64(fltVal.Value), nil
			}
		}
		return 0, fmt.Errorf("hash missing valid 'unix_ms' integer field")
	default:
		return 0, fmt.Errorf("expected INTEGER or DateTime Hash, got %s", arg.Type())
	}
}

func isLeapYear(y int64) bool {
	return (y%4 == 0 && y%100 != 0) || (y%400 == 0)
}

func daysInMonth(y, m int64) int64 {
	switch m {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(y) {
			return 29
		}
		return 28
	default:
		return 0
	}
}

func init() {
	RegisterBuiltin("datetime_now", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: time.Now().UnixMilli()}
		},
	})

	RegisterBuiltin("datetime_now_iso", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: time.Now().UTC().Format(time.RFC3339)}
		},
	})

	RegisterBuiltin("datetime_parse", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("datetime_parse expects 1 or 2 arguments, got %d", len(args))}
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("first argument to datetime_parse must be STRING, got %s", args[0].Type())}
			}
			layout := ""
			if len(args) == 2 {
				if lObj, ok := args[1].(*object.String); ok {
					layout = lObj.Value
				}
			}
			t, err := parseTimeToTime(strObj.Value, layout)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			return timeToHash(t)
		},
	})

	RegisterBuiltin("datetime_format", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("datetime_format expects 1 or 2 arguments, got %d", len(args))}
			}
			ms, err := getUnixMsFromArg(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			t := time.UnixMilli(ms)
			layout := time.RFC3339
			if len(args) == 2 {
				if lObj, ok := args[1].(*object.String); ok {
					layout = convertToLayout(lObj.Value)
				}
			}
			return &object.String{Value: t.Format(layout)}
		},
	})

	RegisterBuiltin("datetime_parts", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("datetime_parts expects 1 or 2 arguments, got %d", len(args))}
			}
			ms, err := getUnixMsFromArg(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			t := time.UnixMilli(ms)
			if len(args) == 2 {
				if b, ok := args[1].(*object.Boolean); ok && b.Value {
					t = t.UTC()
				}
			}
			return timeToHash(t)
		},
	})

	RegisterBuiltin("datetime_is_leap_year", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("datetime_is_leap_year expects 1 argument, got %d", len(args))}
			}
			year := int64(0)
			switch v := args[0].(type) {
			case *object.Integer:
				year = v.Value
			case *object.Hash:
				ms, err := getUnixMsFromArg(v)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				year = int64(time.UnixMilli(ms).Year())
			default:
				return &object.Error{Message: fmt.Sprintf("expected INTEGER year or DateTime Hash, got %s", args[0].Type())}
			}
			return nativeBoolToBooleanObject(isLeapYear(year))
		},
	})

	RegisterBuiltin("datetime_days_in_month", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("datetime_days_in_month expects 2 arguments (year, month), got %d", len(args))}
			}
			yObj, ok1 := args[0].(*object.Integer)
			mObj, ok2 := args[1].(*object.Integer)
			if !ok1 || !ok2 {
				return &object.Error{Message: "arguments to datetime_days_in_month must be (INTEGER, INTEGER)"}
			}
			return &object.Integer{Value: daysInMonth(yObj.Value, mObj.Value)}
		},
	})

	RegisterBuiltin("datetime_add", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{Message: fmt.Sprintf("datetime_add expects 3 arguments (dt_or_ms, amount, unit), got %d", len(args))}
			}
			ms, err := getUnixMsFromArg(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			amtObj, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("amount must be INTEGER, got %s", args[1].Type())}
			}
			unitObj, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("unit must be STRING, got %s", args[2].Type())}
			}

			t := time.UnixMilli(ms)
			amt := amtObj.Value
			unit := strings.ToLower(unitObj.Value)

			switch unit {
			case "ms", "millisecond", "milliseconds":
				t = t.Add(time.Duration(amt) * time.Millisecond)
			case "s", "sec", "second", "seconds":
				t = t.Add(time.Duration(amt) * time.Second)
			case "m", "min", "minute", "minutes":
				t = t.Add(time.Duration(amt) * time.Minute)
			case "h", "hr", "hour", "hours":
				t = t.Add(time.Duration(amt) * time.Hour)
			case "d", "day", "days":
				t = t.AddDate(0, 0, int(amt))
			case "w", "week", "weeks":
				t = t.AddDate(0, 0, int(amt*7))
			case "month", "months":
				t = t.AddDate(0, int(amt), 0)
			case "y", "year", "years":
				t = t.AddDate(int(amt), 0, 0)
			default:
				return &object.Error{Message: fmt.Sprintf("unknown datetime unit '%s'", unitObj.Value)}
			}

			return &object.Integer{Value: t.UnixMilli()}
		},
	})

	RegisterBuiltin("datetime_diff", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Message: fmt.Sprintf("datetime_diff expects 2 or 3 arguments, got %d", len(args))}
			}
			ms1, err := getUnixMsFromArg(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			ms2, err := getUnixMsFromArg(args[1])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			diffMs := ms1 - ms2
			unit := "ms"
			if len(args) == 3 {
				if uObj, ok := args[2].(*object.String); ok {
					unit = strings.ToLower(uObj.Value)
				}
			}

			switch unit {
			case "ms", "millisecond", "milliseconds":
				return &object.Integer{Value: diffMs}
			case "s", "sec", "second", "seconds":
				return &object.Float{Value: float64(diffMs) / 1000.0}
			case "m", "min", "minute", "minutes":
				return &object.Float{Value: float64(diffMs) / (60 * 1000.0)}
			case "h", "hr", "hour", "hours":
				return &object.Float{Value: float64(diffMs) / (3600 * 1000.0)}
			case "d", "day", "days":
				return &object.Float{Value: float64(diffMs) / (86400 * 1000.0)}
			default:
				return &object.Error{Message: fmt.Sprintf("unknown datetime diff unit '%s'", unit)}
			}
		},
	})

	RegisterBuiltin("datetime_timezone_offset", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			_, offset := time.Now().Zone()
			return &object.Integer{Value: int64(offset)}
		},
	})
}
