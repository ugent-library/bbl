package bbl

import "encoding/json"

type jsonlExporter[T Rec] struct {
	enc *json.Encoder
}

func (e *jsonlExporter[T]) Add(rec T) error {
	return e.enc.Encode(rec)
}

func (e *jsonlExporter[T]) Done() error {
	return nil
}
