// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Marshalling and unmarshalling of
// JSON data into Go structs using reflection.

package json

import (
	"json";
	"reflect";
)

type _StructBuilder struct {
	val reflect.Value
}

var nobuilder *_StructBuilder

func setfloat(v reflect.Value, f float64) {
	switch v.Kind() {
	case reflect.FloatKind:
		v.(reflect.FloatValue).Set(float(f));
	case reflect.Float32Kind:
		v.(reflect.Float32Value).Set(float32(f));
	case reflect.Float64Kind:
		v.(reflect.Float64Value).Set(float64(f));
	}
}

func setint(v reflect.Value, i int64) {
	switch v.Kind() {
	case reflect.IntKind:
		v.(reflect.IntValue).Set(int(i));
	case reflect.Int8Kind:
		v.(reflect.Int8Value).Set(int8(i));
	case reflect.Int16Kind:
		v.(reflect.Int16Value).Set(int16(i));
	case reflect.Int32Kind:
		v.(reflect.Int32Value).Set(int32(i));
	case reflect.Int64Kind:
		v.(reflect.Int64Value).Set(int64(i));
	case reflect.UintKind:
		v.(reflect.UintValue).Set(uint(i));
	case reflect.Uint8Kind:
		v.(reflect.Uint8Value).Set(uint8(i));
	case reflect.Uint16Kind:
		v.(reflect.Uint16Value).Set(uint16(i));
	case reflect.Uint32Kind:
		v.(reflect.Uint32Value).Set(uint32(i));
	case reflect.Uint64Kind:
		v.(reflect.Uint64Value).Set(uint64(i));
	}
}

func (b *_StructBuilder) Int64(i int64) {
	if b == nil {
		return
	}
	v := b.val;
	switch v.Kind() {
	case reflect.FloatKind, reflect.Float32Kind, reflect.Float64Kind:
		setfloat(v, float64(i));
	default:
		setint(v, i);
	}
}

func (b *_StructBuilder) Uint64(i uint64) {
	if b == nil {
		return
	}
	v := b.val;
	switch v.Kind() {
	case reflect.FloatKind, reflect.Float32Kind, reflect.Float64Kind:
		setfloat(v, float64(i));
	default:
		setint(v, int64(i));
	}
}

func (b *_StructBuilder) Float64(f float64) {
	if b == nil {
		return
	}
	v := b.val;
	switch v.Kind() {
	case reflect.FloatKind, reflect.Float32Kind, reflect.Float64Kind:
		setfloat(v, f);
	default:
		setint(v, int64(f));
	}
}

func (b *_StructBuilder) Null() {
}

func (b *_StructBuilder) String(s string) {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.StringKind {
		v.(reflect.StringValue).Set(s);
	}
}

func (b *_StructBuilder) Bool(tf bool) {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.BoolKind {
		v.(reflect.BoolValue).Set(tf);
	}
}

func (b *_StructBuilder) Array() {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.PtrKind {
		pv := v.(reflect.PtrValue);
		psubtype := pv.Type().(reflect.PtrType).Sub();
		if pv.Get() == nil && psubtype.Kind() == reflect.ArrayKind {
			av := reflect.NewSliceValue(psubtype.(reflect.ArrayType), 0, 8);
			pv.SetSub(av);
		}
	}
}

func (b *_StructBuilder) Elem(i int) Builder {
	if b == nil || i < 0 {
		return nobuilder
	}
	v := b.val;
	if v.Kind() == reflect.PtrKind {
		// If we have a pointer to an array, allocate or grow
		// the array as necessary.  Then set v to the array itself.
		pv := v.(reflect.PtrValue);
		psub := pv.Sub();
		if psub.Kind() == reflect.ArrayKind {
			av := psub.(reflect.ArrayValue);
			if i > av.Cap() {
				n := av.Cap();
				if n < 8 {
					n = 8
				}
				for n <= i {
					n *= 2
				}
				av1 := reflect.NewSliceValue(av.Type().(reflect.ArrayType), av.Len(), n);
				av1.CopyFrom(av, av.Len());
				pv.SetSub(av1);
				av = av1;
			}
		}
		v = psub;
	}
	if v.Kind() == reflect.ArrayKind {
		// Array was grown above, or is fixed size.
		av := v.(reflect.ArrayValue);
		if av.Len() <= i && i < av.Cap() {
			av.SetLen(i+1);
		}
		if i < av.Len() {
			return &_StructBuilder{ av.Elem(i) }
		}
	}
	return nobuilder
}

func (b *_StructBuilder) Map() {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.PtrKind {
		pv := v.(reflect.PtrValue);
		if pv.Get() == nil {
			pv.SetSub(reflect.NewInitValue(pv.Type().(reflect.PtrType).Sub()))
		}
	}
}

func (b *_StructBuilder) Key(k string) Builder {
	if b == nil {
		return nobuilder
	}
	v := b.val;
	if v.Kind() == reflect.PtrKind {
		v = v.(reflect.PtrValue).Sub();
	}
	if v.Kind() == reflect.StructKind {
		sv := v.(reflect.StructValue);
		t := v.Type().(reflect.StructType);
		for i := 0; i < t.Len(); i++ {
			name, typ, tag, off := t.Field(i);
			if k == name {
				return &_StructBuilder{ sv.Field(i) }
			}
		}
	}
	return nobuilder
}

// Unmarshal parses the JSON syntax string s and fills in
// an arbitrary struct or array pointed at by val.
// It uses the reflection library to assign to fields
// and arrays embedded in val.  Well-formed data that does not fit
// into the struct is discarded.
//
// For example, given the following definitions:
//
//	type Email struct {
//		where string;
//		addr string;
//	}
//
//	type Result struct {
//		name string;
//		phone string;
//		emails []Email
//	}
//
//	var r = Result{ "name", "phone", nil }
//
// unmarshalling the JSON syntax string
//
//	{
//	  "email": [
//	    {
//	      "where": "home",
//	      "addr": "gre@example.com"
//	    },
//	    {
//	      "where": "work",
//	      "addr": "gre@work.com"
//	    }
//	  ],
//	  "name": "Grace R. Emlin",
//	  "address": "123 Main Street"
//	}
//
// via Unmarshal(s, &r) is equivalent to assigning
//
//	r = Result{
//		"Grace R. Emlin",	// name
//		"phone",	// no phone given
//		[]Email{
//			Email{ "home", "gre@example.com" },
//			Email{ "work", "gre@work.com" }
//		}
//	}
//
// Note that the field r.phone has not been modified and
// that the JSON field "address" was discarded.
//
// On success, Unmarshal returns with ok set to true.
// On a syntax error, it returns with ok set to false and errtok
// set to the offending token.
func Unmarshal(s string, val interface{}) (ok bool, errtok string) {
	var errindx int;
	var val1 interface{};
	b := &_StructBuilder{ reflect.NewValue(val) };
	ok, errindx, errtok = Parse(s, b);
	if !ok {
		return false, errtok
	}
	return true, ""
}
