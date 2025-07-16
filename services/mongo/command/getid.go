package command

import (
	"fmt"
	"reflect"
)

func GetDocumentID[T any](document T) (interface{}, error) {
	v := reflect.ValueOf(document)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	field := v.FieldByName("Id")
	if !field.IsValid() {
		field = v.FieldByName("ID")
	}
	if !field.IsValid() {
		return nil, fmt.Errorf("field _id not found")
	}
	
	return field.Interface(), nil
}

func SetDocumentID[T any](document T, id interface{}) error {
	v := reflect.ValueOf(document)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("document must be a pointer")
	}
	v = v.Elem()
	
	field := v.FieldByName("Id")
	if !field.IsValid() {
		field = v.FieldByName("ID")
	}
	if !field.IsValid() {
		return fmt.Errorf("field _id not found")
	}
	if !field.CanSet() {
		return fmt.Errorf("field _id cannot be set")
	}
	
	field.Set(reflect.ValueOf(id))
	return nil
}