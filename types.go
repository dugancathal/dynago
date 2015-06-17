package dynago

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"
)

type StringSet []string

type NumberSet []string

type BinarySet [][]byte

type List []interface{}

type Number string

func (n Number) IntVal() (int, error) {
	return strconv.Atoi(string(n))
}

func (n Number) Int64Val() (int64, error) {
	return strconv.ParseInt(string(n), 10, 64)
}

func (n Number) Uint64Val() (uint64, error) {
	return strconv.ParseUint(string(n), 10, 64)
}

func (n Number) FloatVal() (float64, error) {
	return strconv.ParseFloat(string(n), 64)
}

// Represents an entire document structure composed of keys and dynamo value
type Document map[string]interface{}

func (d Document) MarshalJSON() ([]byte, error) {
	output := make(map[string]interface{}, len(d))
	for key, val := range d {
		if v := reflect.ValueOf(val); !isEmptyValue(v) {
			output[key] = wireEncode(val)
		}
	}
	return json.Marshal(output)
}

func (d *Document) UnmarshalJSON(buf []byte) error {
	raw := make(map[string]interface{})
	err := json.Unmarshal(buf, &raw)
	if err != nil {
		return err
	}
	if *d == nil {
		*d = make(Document)
	}
	dd := *d

	for key, val := range raw {
		dd[key] = wireDecode(val)
	}
	return nil
}

// Helper to get a string from a document.
func (d Document) GetString(key string) string {
	if d[key] != nil {
		return d[key].(string)
	} else {
		return ""
	}
}

func (d Document) GetNumber(key string) Number {
	if d[key] != nil {
		return d[key].(Number)
	} else {
		return Number("")
	}
}

func (d Document) GetStringSet(key string) StringSet {
	if d[key] != nil {
		return d[key].(StringSet)
	} else {
		return StringSet{}
	}
}

func (d Document) GetList(key string) List {
	if d[key] != nil {
		return d[key].(List)
	} else {
		return List{}
	}
}

/*
Helper to get a Time from a document.

If the value is omitted from the DB, or an empty string, then the return
is nil. If the value fails to parse as iso8601, then this method panics.
*/
func (d Document) GetTime(key string) (t *time.Time) {
	val := d[key]
	if val != nil {
		s := val.(string)
		parsed, err := time.ParseInLocation(iso8601compact, s, time.UTC)
		if err != nil {
			panic(err)
		}
		t = &parsed
	}
	return t
}

// Allow a document to be used to specify params
func (d Document) AsParams() (params []Param) {
	for key, val := range d {
		params = append(params, Param{key, val})
	}
	return
}

func (d Document) GetBool(key string) bool {
	if d[key] != nil {
		number := d[key].(Number)

		if res, _ := number.IntVal(); res == 1 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// Helper to build a hash key.
func HashKey(name string, value interface{}) Document {
	return Document{name: value}
}

// Helper to build a hash-range key.
func HashRangeKey(hashName string, hashVal interface{}, rangeName string, rangeVal interface{}) Document {
	return Document{
		hashName:  hashVal,
		rangeName: rangeVal,
	}
}

type Param struct {
	Key   string
	Value interface{}
}

// Allows a solo Param to also satisfy the Params interface
func (p Param) AsParams() []Param {
	return []Param{p}
}

/*
Anything which implements Params can be used as expression parameters for
dynamodb expressions.

DynamoDB item queries using expressions can be provided parameters in a number
of handy ways:
	.Param(":k1", v1).Param(":k2", v2)
	-or-
	.Params(Param{":k1", v1}, Param{":k2", v2})
	-or-
	.FilterExpression("...", Param{":k1", v1}, Param{":k2", v2})
	-or-
	.FilterExpression("...", Document{":k1": v1, ":k2": v2})
Or any combination of Param, Document, or potentially other custom types which
provide the Params interface.
*/
type Params interface {
	AsParams() []Param
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Invalid:
		return true
	}
	return false
}
