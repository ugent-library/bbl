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
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, 0); err == nil {
		*ptr = int(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) IntSlice(key string, ptr *[]int) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]int, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, 0); err == nil {
				slice[i] = int(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Int8(key string, ptr *int8) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, 8); err == nil {
		*ptr = int8(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Int8Slice(key string, ptr *[]int8) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]int8, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, 8); err == nil {
				slice[i] = int8(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Int16(key string, ptr *int16) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, 16); err == nil {
		*ptr = int16(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Int16Slice(key string, ptr *[]int16) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]int16, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, 16); err == nil {
				slice[i] = int16(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Int32(key string, ptr *int32) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, 32); err == nil {
		*ptr = int32(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Int32Slice(key string, ptr *[]int32) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]int32, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, 32); err == nil {
				slice[i] = int32(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Int64(key string, ptr *int64) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseInt(b.values.Get(key), 10, 64); err == nil {
		*ptr = int64(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Int64Slice(key string, ptr *[]int64) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]int64, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseInt(v, 10, 64); err == nil {
				slice[i] = int64(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Uint(key string, ptr *uint) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, 0); err == nil {
		*ptr = uint(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) UintSlice(key string, ptr *[]uint) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]uint, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, 0); err == nil {
				slice[i] = uint(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Uint8(key string, ptr *uint8) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, 8); err == nil {
		*ptr = uint8(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Uint8Slice(key string, ptr *[]uint8) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]uint8, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, 8); err == nil {
				slice[i] = uint8(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Uint16(key string, ptr *uint16) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, 16); err == nil {
		*ptr = uint16(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Uint16Slice(key string, ptr *[]uint16) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]uint16, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, 16); err == nil {
				slice[i] = uint16(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Uint32(key string, ptr *uint32) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, 32); err == nil {
		*ptr = uint32(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Uint32Slice(key string, ptr *[]uint32) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]uint32, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, 32); err == nil {
				slice[i] = uint32(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Uint64(key string, ptr *uint64) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseUint(b.values.Get(key), 10, 64); err == nil {
		*ptr = uint64(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Uint64Slice(key string, ptr *[]uint64) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]uint64, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseUint(v, 10, 64); err == nil {
				slice[i] = uint64(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Float32(key string, ptr *float32) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseFloat(b.values.Get(key), 32); err == nil {
		*ptr = float32(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Float32Slice(key string, ptr *[]float32) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]float32, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseFloat(v, 32); err == nil {
				slice[i] = float32(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
}

func (b *valuesBinder) Float64(key string, ptr *float64) *valuesBinder {
	if b.binder.err != nil || !b.values.Has(key) {
		return b
	}
	if val, err := strconv.ParseFloat(b.values.Get(key), 64); err == nil {
		*ptr = float64(val)
	} else {
		b.binder.err = err
	}
	return b
}

func (b *valuesBinder) Float64Slice(key string, ptr *[]float64) *valuesBinder {
	if b.binder.err != nil {
		return b
	}
	if vals := b.values[key]; len(vals) > 0 {
		slice := make([]float64, len(vals))
		for i, v := range vals {
			if val, err := strconv.ParseFloat(v, 64); err == nil {
				slice[i] = float64(val)
			} else {
				b.binder.err = err
				return b
			}
		}
		*ptr = slice
	}
	return b
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
