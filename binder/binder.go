package binder

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

func New(r *http.Request) *Binder {
	return &Binder{r: r}
}

type Binder struct {
	r           *http.Request
	err         error
	queryBinder *valuesBinder
	formBinder  *valuesBinder
}

type valuesBinder struct {
	binder *Binder
	values url.Values
}

func (b *Binder) Query() *valuesBinder {
	if b.queryBinder == nil {
		b.queryBinder = &valuesBinder{binder: b, values: b.r.URL.Query()}
	}
	return b.queryBinder
}

func (b *Binder) Form() *valuesBinder {
	if b.formBinder == nil {
		if b.r.Form == nil {
			b.err = b.r.ParseMultipartForm(32 << 20)
		}
		b.formBinder = &valuesBinder{binder: b, values: b.r.Form}
	}
	return b.formBinder
}

func (b *Binder) Err() error {
	return b.err
}

func (b *valuesBinder) Query() *valuesBinder {
	return b.binder.Query()
}

func (b *valuesBinder) Form() *valuesBinder {
	return b.binder.Form()
}

func (b *valuesBinder) Err() error {
	return b.binder.err
}

func (b *valuesBinder) Vacuum() *valuesBinder {
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

func (b *valuesBinder) String(key string, ptr *string) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	*ptr = b.values.Get(key)
	return b
}

func (b *valuesBinder) StringSlice(key string, ptr *[]string) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		*ptr = slices.Clone(vals)
	}
	return b
}

func (b *valuesBinder) Bool(key string, ptr *bool) *valuesBinder {
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

func (b *valuesBinder) BoolSlice(key string, ptr *[]bool) *valuesBinder {
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

func (b *valuesBinder) Int(key string, ptr *int) *valuesBinder {
	return bindInt(b, key, ptr, 0)
}

func (b *valuesBinder) IntSlice(key string, ptr *[]int) *valuesBinder {
	return bindIntSlice(b, key, ptr, 0)
}

func (b *valuesBinder) Int8(key string, ptr *int8) *valuesBinder {
	return bindInt(b, key, ptr, 8)
}

func (b *valuesBinder) Int8Slice(key string, ptr *[]int8) *valuesBinder {
	return bindIntSlice(b, key, ptr, 8)
}

func (b *valuesBinder) Int16(key string, ptr *int16) *valuesBinder {
	return bindInt(b, key, ptr, 16)
}

func (b *valuesBinder) Int16Slice(key string, ptr *[]int16) *valuesBinder {
	return bindIntSlice(b, key, ptr, 16)
}

func (b *valuesBinder) Int32(key string, ptr *int32) *valuesBinder {
	return bindInt(b, key, ptr, 32)
}

func (b *valuesBinder) Int32Slice(key string, ptr *[]int32) *valuesBinder {
	return bindIntSlice(b, key, ptr, 32)
}

func (b *valuesBinder) Int64(key string, ptr *int64) *valuesBinder {
	return bindInt(b, key, ptr, 64)
}

func (b *valuesBinder) Int64Slice(key string, ptr *[]int64) *valuesBinder {
	return bindIntSlice(b, key, ptr, 64)
}

func (b *valuesBinder) Uint(key string, ptr *uint) *valuesBinder {
	return bindUint(b, key, ptr, 0)
}

func (b *valuesBinder) UintSlice(key string, ptr *[]uint) *valuesBinder {
	return bindUintSlice(b, key, ptr, 0)
}

func (b *valuesBinder) Uint8(key string, ptr *uint8) *valuesBinder {
	return bindUint(b, key, ptr, 8)
}

func (b *valuesBinder) Uint8Slice(key string, ptr *[]uint8) *valuesBinder {
	return bindUintSlice(b, key, ptr, 8)
}

func (b *valuesBinder) Uint16(key string, ptr *uint16) *valuesBinder {
	return bindUint(b, key, ptr, 16)
}

func (b *valuesBinder) Uint16Slice(key string, ptr *[]uint16) *valuesBinder {
	return bindUintSlice(b, key, ptr, 16)
}

func (b *valuesBinder) Uint32(key string, ptr *uint32) *valuesBinder {
	return bindUint(b, key, ptr, 32)
}

func (b *valuesBinder) Uint32Slice(key string, ptr *[]uint32) *valuesBinder {
	return bindUintSlice(b, key, ptr, 32)
}

func (b *valuesBinder) Uint64(key string, ptr *uint64) *valuesBinder {
	return bindUint(b, key, ptr, 64)
}

func (b *valuesBinder) Uint64Slice(key string, ptr *[]uint64) *valuesBinder {
	return bindUintSlice(b, key, ptr, 64)
}

func (b *valuesBinder) Float32(key string, ptr *float32) *valuesBinder {
	return bindFloat(b, key, ptr, 32)
}

func (b *valuesBinder) Float32Slice(key string, ptr *[]float32) *valuesBinder {
	return bindFloatSlice(b, key, ptr, 32)
}

func (b *valuesBinder) Float64(key string, ptr *float64) *valuesBinder {
	return bindFloat(b, key, ptr, 64)
}

func (b *valuesBinder) Float64Slice(key string, ptr *[]float64) *valuesBinder {
	return bindFloatSlice(b, key, ptr, 64)
}

func (b *valuesBinder) Time(key string, layout string, ptr *time.Time) *valuesBinder {
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

func (b *valuesBinder) TimeSlice(key string, layout string, ptr *[]time.Time) *valuesBinder {
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

func bindInt[T int | int8 | int16 | int32 | int64](b *valuesBinder, key string, ptr *T, bitSize int) *valuesBinder {
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

func bindIntSlice[T int | int8 | int16 | int32 | int64](b *valuesBinder, key string, ptr *[]T, bitSize int) *valuesBinder {
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

func bindUint[T uint | uint8 | uint16 | uint32 | uint64](b *valuesBinder, key string, ptr *T, bitSize int) *valuesBinder {
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

func bindUintSlice[T uint | uint8 | uint16 | uint32 | uint64](b *valuesBinder, key string, ptr *[]T, bitSize int) *valuesBinder {
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

func bindFloat[T float32 | float64](b *valuesBinder, key string, ptr *T, bitSize int) *valuesBinder {
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

func bindFloatSlice[T float32 | float64](b *valuesBinder, key string, ptr *[]T, bitSize int) *valuesBinder {
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
