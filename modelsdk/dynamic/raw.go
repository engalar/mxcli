package dynamic

import (
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func RawString(elem element.Element, key string) (string, bool) {
	raw := elem.Raw()
	if raw == nil {
		return "", false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return "", false
	}
	return val.StringValueOK()
}

func RawBool(elem element.Element, key string) (bool, bool) {
	raw := elem.Raw()
	if raw == nil {
		return false, false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return false, false
	}
	return val.BooleanOK()
}

func RawInt32(elem element.Element, key string) (int32, bool) {
	raw := elem.Raw()
	if raw == nil {
		return 0, false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return 0, false
	}
	return val.Int32OK()
}

func rawDocument(elem element.Element, key string) (bson.Raw, bool) {
	raw := elem.Raw()
	if raw == nil {
		return nil, false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return nil, false
	}
	doc, ok := val.DocumentOK()
	if !ok {
		return nil, false
	}
	return bson.Raw(doc), true
}
