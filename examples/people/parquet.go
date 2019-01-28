package main

// This code is generated by github.com/parsyl/parquet.

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/golang/snappy"
	"github.com/parsyl/parquet"
)

// ParquetWriter reprents a row group
type ParquetWriter struct {
	fields []Field

	len int

	// child points to the next page
	child *ParquetWriter

	// max is the number of Record items that can get written before
	// a new set of column chunks is written
	max int

	meta *parquet.Metadata
	w    *WriteCounter
}

func Fields() []Field {
	return []Field{
		NewInt32Field(func(x Person) int32 { return x.ID }, func(x *Person, v int32) { x.ID = v }, "id"),
		NewInt32OptionalField(func(x Person) *int32 { return x.Age }, func(x *Person, v *int32) { x.Age = v }, "age"),
		NewInt64Field(func(x Person) int64 { return x.Happiness }, func(x *Person, v int64) { x.Happiness = v }, "happiness"),
		NewInt64OptionalField(func(x Person) *int64 { return x.Sadness }, func(x *Person, v *int64) { x.Sadness = v }, "sadness"),
		NewStringField(func(x Person) string { return x.Code }, func(x *Person, v string) { x.Code = v }, "code"),
		NewFloat32Field(func(x Person) float32 { return x.Funkiness }, func(x *Person, v float32) { x.Funkiness = v }, "funkiness"),
		NewFloat32OptionalField(func(x Person) *float32 { return x.Lameness }, func(x *Person, v *float32) { x.Lameness = v }, "lameness"),
		NewBoolOptionalField(func(x Person) *bool { return x.Keen }, func(x *Person, v *bool) { x.Keen = v }, "keen"),
		NewUint32Field(func(x Person) uint32 { return x.Birthday }, func(x *Person, v uint32) { x.Birthday = v }, "birthday"),
		NewUint64OptionalField(func(x Person) *uint64 { return x.Anniversary }, func(x *Person, v *uint64) { x.Anniversary = v }, "anniversary"),
	}
}

func NewParquetWriter(w io.Writer, opts ...func(*ParquetWriter)) *ParquetWriter {
	p := &ParquetWriter{
		max:    1000,
		w:      &WriteCounter{w: w},
		fields: Fields(),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.meta == nil {
		ff := Fields()
		schema := make([]parquet.Field, len(ff))
		for i, f := range ff {
			schema[i] = f.Schema()
		}
		p.meta = parquet.New(schema...)
	}

	return p
}

func withMeta(m *parquet.Metadata) func(*ParquetWriter) {
	return func(p *ParquetWriter) {
		p.meta = m
	}
}

// MaxPageSize is the maximum number of rows in each row groups' page.
func MaxPageSize(m int) func(*ParquetWriter) {
	return func(p *ParquetWriter) {
		p.max = m
	}
}

func (p *ParquetWriter) Write() error {
	if _, err := p.w.Write([]byte("PAR1")); err != nil {
		return err
	}

	for i, f := range p.fields {
		pos := p.w.n
		f.Write(p.w, p.meta, pos)

		for child := p.child; child != nil; child = child.child {
			pos := p.w.n
			child.fields[i].Write(p.w, p.meta, pos)
		}
	}

	if err := p.meta.Footer(p.w); err != nil {
		return err
	}

	_, err := p.w.Write([]byte("PAR1"))
	return err
}

func (p *ParquetWriter) Add(rec Person) {
	if p.len == p.max {
		if p.child == nil {
			p.child = NewParquetWriter(p.w, MaxPageSize(p.max), withMeta(p.meta))
		}

		p.child.Add(rec)
		return
	}

	for _, f := range p.fields {
		f.Add(rec)
	}

	p.len++
}

type Field interface {
	Add(r Person)
	Scan(r *Person)
	Read(r io.Reader, meta *parquet.Metadata, pos int) error
	Write(w io.Writer, meta *parquet.Metadata, pos int) error
	Schema() parquet.Field
}

type RequiredNumField struct {
	vals []interface{}
	col  string
}

func (i *RequiredNumField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	for _, i := range i.vals {
		if err := binary.Write(wc, binary.LittleEndian, i); err != nil {
			return err
		}
	}

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, i.col, pos, wc.n, len(compressed), len(i.vals)); err != nil {
		return err
	}

	_, err := io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

func (i *RequiredNumField) Read(r io.Reader, meta *parquet.Metadata, pos int) error {
	return nil
}

type OptionalNumField struct {
	vals []interface{}
	defs []int64
	col  string
}

func (i *OptionalNumField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	err := WriteLevels(wc, i.defs)
	if err != nil {
		return err
	}

	for _, i := range i.vals {
		if err := binary.Write(wc, binary.LittleEndian, i); err != nil {
			return err
		}
	}

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, i.col, pos, wc.n, len(compressed), len(i.defs)); err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

func (i *OptionalNumField) Read(r io.Reader, meta *parquet.Metadata, pos int) error {
	return nil
}

type Uint32Field struct {
	RequiredNumField
	val  func(r Person) uint32
	read func(r *Person, v uint32)
}

func NewUint32Field(val func(r Person) uint32, read func(r *Person, v uint32), col string) *Uint32Field {
	return &Uint32Field{
		val:              val,
		read:             read,
		RequiredNumField: RequiredNumField{col: col},
	}
}

func (i *Uint32Field) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Uint32Type, RepetitionType: parquet.RepetitionRequired}
}

func (i *Uint32Field) Scan(r *Person) {
	v := i.vals[0].(uint32)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Uint32Field) Add(r Person) {
	i.vals = append(i.vals, i.val(r))
}

type Uint32OptionalField struct {
	OptionalNumField
	val func(r Person) *uint32
}

func NewUint32OptionalField(val func(r Person) *uint32, col string) *Uint32OptionalField {
	return &Uint32OptionalField{
		val:              val,
		OptionalNumField: OptionalNumField{col: col},
	}
}

func (i *Uint32OptionalField) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Uint32Type, RepetitionType: parquet.RepetitionOptional}
}

func (i *Uint32OptionalField) Add(r Person) {
	v := i.val(r)
	if v != nil {
		i.vals = append(i.vals, *v)
		i.defs = append(i.defs, 1)
	} else {
		i.defs = append(i.defs, 0)
	}
}

type Int32Field struct {
	RequiredNumField
	val  func(r Person) int32
	read func(r *Person, v int32)
}

func NewInt32Field(val func(r Person) int32, read func(r *Person, v int32), col string) *Int32Field {
	return &Int32Field{
		val:              val,
		read:             read,
		RequiredNumField: RequiredNumField{col: col},
	}
}

func (i *Int32Field) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Int32Type, RepetitionType: parquet.RepetitionRequired}
}

func (i *Int32Field) Add(r Person) {
	i.vals = append(i.vals, i.val(r))
}

func (i *Int32Field) Scan(r *Person) {
	v := i.vals[0].(int32)
	i.vals = i.vals[1:]
	i.read(r, v)
}

type Int32OptionalField struct {
	OptionalNumField
	val  func(r Person) *int32
	read func(r *Person, v *int32)
}

func NewInt32OptionalField(val func(r Person) *int32, read func(r *Person, v *int32), col string) *Int32OptionalField {
	return &Int32OptionalField{
		val:              val,
		read:             read,
		OptionalNumField: OptionalNumField{col: col},
	}
}

func (i *Int32OptionalField) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Int32Type, RepetitionType: parquet.RepetitionOptional}
}

func (i *Int32OptionalField) Scan(r *Person) {
	v := i.vals[0].(*int32)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Int32OptionalField) Add(r Person) {
	v := i.val(r)
	if v != nil {
		i.vals = append(i.vals, *v)
		i.defs = append(i.defs, 1)
	} else {
		i.defs = append(i.defs, 0)
	}
}

type Int64Field struct {
	RequiredNumField
	val  func(r Person) int64
	read func(r *Person, v int64)
}

func NewInt64Field(val func(r Person) int64, read func(r *Person, v int64), col string) *Int64Field {
	return &Int64Field{
		val:              val,
		read:             read,
		RequiredNumField: RequiredNumField{col: col},
	}
}

func (i *Int64Field) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Int64Type, RepetitionType: parquet.RepetitionRequired}
}

func (i *Int64Field) Scan(r *Person) {
	v := i.vals[0].(int64)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Int64Field) Add(r Person) {
	i.vals = append(i.vals, i.val(r))
}

type Int64OptionalField struct {
	OptionalNumField
	val  func(r Person) *int64
	read func(r *Person, v *int64)
}

func NewInt64OptionalField(val func(r Person) *int64, read func(r *Person, v *int64), col string) *Int64OptionalField {
	return &Int64OptionalField{
		val:              val,
		read:             read,
		OptionalNumField: OptionalNumField{col: col},
	}
}

func (i *Int64OptionalField) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Int64Type, RepetitionType: parquet.RepetitionOptional}
}

func (i *Int64OptionalField) Scan(r *Person) {
	v := i.vals[0].(*int64)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Int64OptionalField) Add(r Person) {
	v := i.val(r)
	if v != nil {
		i.vals = append(i.vals, *v)
		i.defs = append(i.defs, 1)
	} else {
		i.defs = append(i.defs, 0)
	}
}

type Uint64Field struct {
	RequiredNumField
	val  func(r Person) uint64
	read func(r *Person, v uint64)
}

func NewUint64Field(val func(r Person) uint64, read func(r *Person, v uint64), col string) *Uint64Field {
	return &Uint64Field{
		val:              val,
		read:             read,
		RequiredNumField: RequiredNumField{col: col},
	}
}

func (i *Uint64Field) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Uint64Type, RepetitionType: parquet.RepetitionRequired}
}

func (i *Uint64Field) Scan(r *Person) {
	v := i.vals[0].(uint64)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Uint64Field) Add(r Person) {
	i.vals = append(i.vals, i.val(r))
}

type Uint64OptionalField struct {
	OptionalNumField
	val  func(r Person) *uint64
	read func(r *Person, v *uint64)
}

func NewUint64OptionalField(val func(r Person) *uint64, read func(r *Person, v *uint64), col string) *Uint64OptionalField {
	return &Uint64OptionalField{
		val:              val,
		read:             read,
		OptionalNumField: OptionalNumField{col: col},
	}
}

func (i *Uint64OptionalField) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Uint64Type, RepetitionType: parquet.RepetitionOptional}
}

func (i *Uint64OptionalField) Scan(r *Person) {
	v := i.vals[0].(*uint64)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Uint64OptionalField) Add(r Person) {
	v := i.val(r)
	if v != nil {
		i.vals = append(i.vals, *v)
		i.defs = append(i.defs, 1)
	} else {
		i.defs = append(i.defs, 0)
	}
}

type Float32Field struct {
	RequiredNumField
	val  func(r Person) float32
	read func(r *Person, v float32)
}

func NewFloat32Field(val func(r Person) float32, read func(r *Person, v float32), col string) *Float32Field {
	return &Float32Field{
		val:              val,
		read:             read,
		RequiredNumField: RequiredNumField{col: col},
	}
}

func (i *Float32Field) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Float32Type, RepetitionType: parquet.RepetitionRequired}
}

func (i *Float32Field) Scan(r *Person) {
	v := i.vals[0].(float32)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Float32Field) Add(r Person) {
	i.vals = append(i.vals, i.val(r))
}

type Float32OptionalField struct {
	OptionalNumField
	val  func(r Person) *float32
	read func(r *Person, v *float32)
}

func NewFloat32OptionalField(val func(r Person) *float32, read func(r *Person, v *float32), col string) *Float32OptionalField {
	return &Float32OptionalField{
		val:              val,
		read:             read,
		OptionalNumField: OptionalNumField{col: col},
	}
}

func (i *Float32OptionalField) Schema() parquet.Field {
	return parquet.Field{Name: i.col, Type: parquet.Float32Type, RepetitionType: parquet.RepetitionOptional}
}

func (i *Float32OptionalField) Scan(r *Person) {
	v := i.vals[0].(*float32)
	i.vals = i.vals[1:]
	i.read(r, v)
}

func (i *Float32OptionalField) Add(r Person) {
	v := i.val(r)
	if v != nil {
		i.vals = append(i.vals, *v)

		i.defs = append(i.defs, 1)
	} else {
		i.defs = append(i.defs, 0)
	}
}

type BoolOptionalField struct {
	vals []bool
	defs []int64
	col  string
	val  func(r Person) *bool
	read func(r *Person, v *bool)
}

func NewBoolOptionalField(val func(r Person) *bool, read func(r *Person, v *bool), col string) *BoolOptionalField {
	return &BoolOptionalField{
		val:  val,
		read: read,
		col:  col,
	}
}

func (f *BoolOptionalField) Schema() parquet.Field {
	return parquet.Field{Name: f.col, Type: parquet.BoolType, RepetitionType: parquet.RepetitionOptional}
}

func (f *BoolOptionalField) Scan(r *Person) {
	v := f.vals[0]
	f.vals = f.vals[1:]
	f.read(r, &v)
}

func (f *BoolOptionalField) Add(r Person) {
	v := f.val(r)
	if v != nil {
		f.vals = append(f.vals, *v)
		f.defs = append(f.defs, 1)
	} else {
		f.defs = append(f.defs, 0)
	}
}

func (f *BoolOptionalField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	err := WriteLevels(wc, f.defs)
	if err != nil {
		return err
	}

	ln := len(f.vals)
	byteNum := (ln + 7) / 8
	rawBuf := make([]byte, byteNum)

	for i := 0; i < ln; i++ {
		if f.vals[i] {
			rawBuf[i/8] = rawBuf[i/8] | (1 << uint32(i%8))
		}
	}

	wc.Write(rawBuf)

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, f.col, pos, wc.n, len(compressed), len(f.defs)); err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

func (f *BoolOptionalField) Read(r io.Reader, meta *parquet.Metadata, pos int) error {
	return nil
}

type BoolField struct {
	vals []bool
	col  string
	val  func(r Person) bool
	read func(r *Person, v bool)
}

func NewBoolField(val func(r Person) bool, read func(r *Person, v bool), col string) *BoolField {
	return &BoolField{
		val:  val,
		read: read,
		col:  col,
	}
}

func (f *BoolField) Schema() parquet.Field {
	return parquet.Field{Name: f.col, Type: parquet.BoolType, RepetitionType: parquet.RepetitionRequired}
}

func (f *BoolField) Scan(r *Person) {
	v := f.vals[0]
	f.vals = f.vals[1:]
	f.read(r, v)
}

func (f *BoolField) Add(r Person) {
	f.vals = append(f.vals, f.val(r))
}

func (f *BoolField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	ln := len(f.vals)
	byteNum := (ln + 7) / 8
	rawBuf := make([]byte, byteNum)

	for i := 0; i < ln; i++ {
		if f.vals[i] {
			rawBuf[i/8] = rawBuf[i/8] | (1 << uint32(i%8))
		}
	}

	wc.Write(rawBuf)

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, f.col, pos, wc.n, len(compressed), len(f.vals)); err != nil {
		return err
	}

	_, err := io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

type StringField struct {
	vals []string
	col  string
	val  func(r Person) string
	read func(r *Person, v string)
}

func NewStringField(val func(r Person) string, read func(r *Person, v string), col string) *StringField {
	return &StringField{
		val:  val,
		read: read,
		col:  col,
	}
}

func (f *StringField) Schema() parquet.Field {
	return parquet.Field{Name: f.col, Type: parquet.StringType, RepetitionType: parquet.RepetitionRequired}
}

func (f *StringField) Scan(r *Person) {
	v := f.vals[0]
	f.vals = f.vals[1:]
	f.read(r, v)
}

func (f *StringField) Add(r Person) {
	f.vals = append(f.vals, f.val(r))
}

func (f *StringField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	for _, s := range f.vals {
		if err := binary.Write(wc, binary.LittleEndian, int32(len(s))); err != nil {
			return err
		}
		wc.Write([]byte(s))
	}

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, f.col, pos, wc.n, len(compressed), len(f.vals)); err != nil {
		return err
	}

	_, err := io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

func (f *StringField) Read(r io.Reader, meta *parquet.Metadata, pos int) error {
	return nil
}

type StringOptionalField struct {
	vals []string
	defs []int64
	col  string
	val  func(r Person) *string
	read func(r *Person, v *string)
}

func NewStringOptionalField(val func(r Person) *string, read func(r *Person, v *string), col string) *StringOptionalField {
	return &StringOptionalField{
		val:  val,
		read: read,
		col:  col,
	}
}

func (f *StringOptionalField) Schema() parquet.Field {
	return parquet.Field{Name: f.col, Type: parquet.StringType, RepetitionType: parquet.RepetitionOptional}
}

func (f *StringOptionalField) Scan(r *Person) {
	v := f.vals[0]
	f.vals = f.vals[1:]
	f.read(r, &v)
}

func (f *StringOptionalField) Add(r Person) {
	v := f.val(r)
	if v != nil {
		f.vals = append(f.vals, *v)
		f.defs = append(f.defs, 1)
	} else {
		f.defs = append(f.defs, 0)
	}
}

func (f *StringOptionalField) Write(w io.Writer, meta *parquet.Metadata, pos int) error {
	buf := bytes.Buffer{}
	wc := &WriteCounter{w: &buf}

	err := WriteLevels(wc, f.defs)
	if err != nil {
		return err
	}

	for _, s := range f.vals {
		if err := binary.Write(wc, binary.LittleEndian, int32(len(s))); err != nil {
			return err
		}
		wc.Write([]byte(s))
	}

	compressed := snappy.Encode(nil, buf.Bytes())
	if err := meta.WritePageHeader(w, f.col, pos, wc.n, len(compressed), len(f.defs)); err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewBuffer(compressed))
	return err
}

func (f *StringOptionalField) Read(r io.Reader, meta *parquet.Metadata, pos int) error {
	return nil
}

type WriteCounter struct {
	n int
	w io.Writer
}

func (w *WriteCounter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.n += n
	return n, err
}

// WriteLevels writes vals to w as RLE encoded data
func WriteLevels(w io.Writer, vals []int64) error {
	var max uint64
	if len(vals) > 0 {
		max = 1
	}

	rleBuf := writeRLE(vals, int32(bitNum(max)))
	res := make([]byte, 0)
	var lenBuf bytes.Buffer
	binary.Write(&lenBuf, binary.LittleEndian, int32(len(rleBuf)))
	res = append(res, lenBuf.Bytes()...)
	res = append(res, rleBuf...)
	_, err := io.Copy(w, bytes.NewBuffer(res))
	return err
}

func writeRLE(vals []int64, bitWidth int32) []byte {
	ln := len(vals)
	i := 0
	res := make([]byte, 0)
	for i < ln {
		j := i + 1
		for j < ln && vals[j] == vals[i] {
			j++
		}
		num := j - i
		header := num << 1
		byteNum := (bitWidth + 7) / 8

		headerBuf := writeUnsignedVarInt(uint64(header))

		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, vals[i])
		valBuf := buf.Bytes()
		rleBuf := make([]byte, int64(len(headerBuf))+int64(byteNum))
		copy(rleBuf[0:], headerBuf)
		copy(rleBuf[len(headerBuf):], valBuf[0:byteNum])
		res = append(res, rleBuf...)
		i = j
	}
	return res
}

func writeUnsignedVarInt(num uint64) []byte {
	byteNum := (bitNum(uint64(num)) + 6) / 7
	if byteNum == 0 {
		return make([]byte, 1)
	}
	res := make([]byte, byteNum)

	numTmp := num
	for i := 0; i < int(byteNum); i++ {
		res[i] = byte(numTmp & uint64(0x7F))
		res[i] = res[i] | byte(0x80)
		numTmp = numTmp >> 7
	}
	res[byteNum-1] &= byte(0x7F)
	return res
}

func bitNum(num uint64) uint64 {
	var bitn uint64
	for ; num != 0; num >>= 1 {
		bitn++
	}
	return bitn
}
