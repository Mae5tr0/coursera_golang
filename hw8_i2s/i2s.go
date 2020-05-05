package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	val := reflect.ValueOf(out)

	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("out is not a pointer")
	} else {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Int:
		dataVal, ok := data.(float64)
		if !ok {
			return fmt.Errorf("Can't convet to float64")
		}
		val.SetInt(int64(dataVal))
	case reflect.String:
		dataVal, ok := data.(string)
		if !ok {
			return fmt.Errorf("Can't convet to string")
		}
		val.SetString(dataVal)
	case reflect.Bool:
		dataVal, ok := data.(bool)
		if !ok {
			return fmt.Errorf("Can't convet to bool")
		}
		val.SetBool(dataVal)
	case reflect.Slice:
		dataSlice, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("Can't convet to slice")
		}
		for _, obj := range dataSlice {
			el := reflect.New(val.Type().Elem())
			err := i2s(obj, el.Interface())
			if err != nil {
				return fmt.Errorf("Can't parse slice")
			}
			val.Set(reflect.Append(val, el.Elem()))
		}
	case reflect.Struct:
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Can't convet to map")
		}

		for i := 0; i < val.NumField(); i++ {
			valueField := val.Field(i)
			typeField := val.Type().Field(i)

			err := i2s(dataMap[typeField.Name], valueField.Addr().Interface())
			if err != nil {
				return fmt.Errorf("Can't parse sub structure")
			}
		}
	default:
		return fmt.Errorf("Unknown type")
	}

	return nil
}
