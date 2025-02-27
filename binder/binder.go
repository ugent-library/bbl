package binder

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/constraints"
)

func New(r *http.Request) *Binder {
	return &Binder{r: r}
}

type Binder struct {
	r           *http.Request
	err         error
	queryBinder *Values
	formBinder  *Values
}

type Values struct {
	binder *Binder
	values url.Values
}

func (b *Binder) Query() *Values {
	if b.queryBinder == nil {
		b.queryBinder = &Values{binder: b, values: b.r.URL.Query()}
	}
	return b.queryBinder
}

func (b *Binder) Form() *Values {
	if b.formBinder == nil {
		if b.r.Form == nil {
			b.err = b.r.ParseMultipartForm(32 << 20)
		}
		b.formBinder = &Values{binder: b, values: b.r.Form}
	}
	return b.formBinder
}

func (b *Binder) Err() error {
	return b.err
}

func (b *Values) Query() *Values {
	return b.binder.Query()
}

func (b *Values) Form() *Values {
	return b.binder.Form()
}

func (b *Values) Err() error {
	return b.binder.err
}

func (b *Values) Vacuum() *Values {
	if b.binder.err == nil {
		newValues := make(url.Values)
		for key, vals := range b.values {
			var newVals []string
			for _, val := range vals {
				val = strings.TrimSpace(val)
				if val != "" {
					newVals = append(newVals, val)
				}
			}
			if len(newVals) > 0 {
				newValues[key] = newVals
			}
		}
		b.values = newValues
	}
	return b
}

// TODO cap sparse array size
func (b *Values) Each(key string, yield func(*Values) bool) *Values {
	if b.binder.err != nil {
		return b
	}

	vv := []url.Values{}

	prefix := key + "["
	for key, vals := range b.values {
		if rest, ok := strings.CutPrefix(key, prefix); ok {
			if idx, newKey, ok := strings.Cut(rest, "]."); ok {
				intIdx, err := strconv.ParseInt(idx, 10, 0)
				if err != nil {
					b.binder.err = err
					return b
				}

				i := int(intIdx)

				if i >= len(vv) && i < cap(vv) {
					vv = vv[:i+1]
				} else if i >= cap(vv) {
					s := vv
					vv = make([]url.Values, i+1)
					copy(vv, s)
				}

				if v := vv[i]; v != nil {
					v[newKey] = vals
				} else {
					vv[i] = url.Values{newKey: vals}
				}
			}
		}
	}

	for _, v := range vv {
		if !yield(&Values{binder: b.binder, values: v}) {
			break
		}
	}

	return b
}

func (b *Values) String(key string, ptr *string) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	*ptr = b.values.Get(key)
	return b
}

func (b *Values) StringSlice(key string, ptr *[]string) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		*ptr = slices.Clone(vals)
	}
	return b
}

func (b *Values) Bool(key string, ptr *bool) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseBool(b.values.Get(key)); err == nil {
		*ptr = val
	} else {
		b.binder.err = err
	}
	return b
}

func (b *Values) BoolSlice(key string, ptr *[]bool) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]bool, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseBool(v); err == nil {
				slice[i] = val
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *Values) Int(key string, ptr *int) *Values {
	return bindInt(b, key, ptr, 0)
}

func (b *Values) IntSlice(key string, ptr *[]int) *Values {
	return bindIntSlice(b, key, ptr, 0)
}

func (b *Values) Int8(key string, ptr *int8) *Values {
	return bindInt(b, key, ptr, 8)
}

func (b *Values) Int8Slice(key string, ptr *[]int8) *Values {
	return bindIntSlice(b, key, ptr, 8)
}

func (b *Values) Int16(key string, ptr *int16) *Values {
	return bindInt(b, key, ptr, 16)
}

func (b *Values) Int16Slice(key string, ptr *[]int16) *Values {
	return bindIntSlice(b, key, ptr, 16)
}

func (b *Values) Int32(key string, ptr *int32) *Values {
	return bindInt(b, key, ptr, 32)
}

func (b *Values) Int32Slice(key string, ptr *[]int32) *Values {
	return bindIntSlice(b, key, ptr, 32)
}

func (b *Values) Int64(key string, ptr *int64) *Values {
	return bindInt(b, key, ptr, 64)
}

func (b *Values) Int64Slice(key string, ptr *[]int64) *Values {
	return bindIntSlice(b, key, ptr, 64)
}

func (b *Values) Uint(key string, ptr *uint) *Values {
	return bindUint(b, key, ptr, 0)
}

func (b *Values) UintSlice(key string, ptr *[]uint) *Values {
	return bindUintSlice(b, key, ptr, 0)
}

func (b *Values) Uint8(key string, ptr *uint8) *Values {
	return bindUint(b, key, ptr, 8)
}

func (b *Values) Uint8Slice(key string, ptr *[]uint8) *Values {
	return bindUintSlice(b, key, ptr, 8)
}

func (b *Values) Uint16(key string, ptr *uint16) *Values {
	return bindUint(b, key, ptr, 16)
}

func (b *Values) Uint16Slice(key string, ptr *[]uint16) *Values {
	return bindUintSlice(b, key, ptr, 16)
}

func (b *Values) Uint32(key string, ptr *uint32) *Values {
	return bindUint(b, key, ptr, 32)
}

func (b *Values) Uint32Slice(key string, ptr *[]uint32) *Values {
	return bindUintSlice(b, key, ptr, 32)
}

func (b *Values) Uint64(key string, ptr *uint64) *Values {
	return bindUint(b, key, ptr, 64)
}

func (b *Values) Uint64Slice(key string, ptr *[]uint64) *Values {
	return bindUintSlice(b, key, ptr, 64)
}

func (b *Values) Float32(key string, ptr *float32) *Values {
	return bindFloat(b, key, ptr, 32)
}

func (b *Values) Float32Slice(key string, ptr *[]float32) *Values {
	return bindFloatSlice(b, key, ptr, 32)
}

func (b *Values) Float64(key string, ptr *float64) *Values {
	return bindFloat(b, key, ptr, 64)
}

func (b *Values) Float64Slice(key string, ptr *[]float64) *Values {
	return bindFloatSlice(b, key, ptr, 64)
}

func (b *Values) Time(key string, layout string, ptr *time.Time) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := time.Parse(layout, b.values.Get(key)); err == nil {
		*ptr = val
	} else {
		b.binder.err = err
	}
	return b
}

func (b *Values) TimeSlice(key string, layout string, ptr *[]time.Time) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]time.Time, len(vals))
		for i, v := range vals {
			if val, err := time.Parse(layout, v); err == nil {
				slice[i] = val
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func bindInt[T constraints.Signed](b *Values, key string, ptr *T, bitSize int) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, bitSize); err == nil {
		*ptr = T(val)
	} else {
		b.binder.err = err
	}
	return b
}

func bindIntSlice[T constraints.Signed](b *Values, key string, ptr *[]T, bitSize int) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]T, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, bitSize); err == nil {
				slice[i] = T(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func bindUint[T constraints.Unsigned](b *Values, key string, ptr *T, bitSize int) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, bitSize); err == nil {
		*ptr = T(val)
	} else {
		b.binder.err = err
	}
	return b
}

func bindUintSlice[T constraints.Unsigned](b *Values, key string, ptr *[]T, bitSize int) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]T, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, bitSize); err == nil {
				slice[i] = T(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func bindFloat[T constraints.Float](b *Values, key string, ptr *T, bitSize int) *Values {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseFloat(b.values.Get(key), bitSize); err == nil {
		*ptr = T(val)
	} else {
		b.binder.err = err
	}
	return b
}

func bindFloatSlice[T constraints.Float](b *Values, key string, ptr *[]T, bitSize int) *Values {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]T, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseFloat(v, bitSize); err == nil {
				slice[i] = T(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}
