package dremel

import (
	"github.com/parsyl/parquet/internal/fields"
)

// Package dremel generates code that parquetgen
// uses to encode/decode a struct for writing and
// reading parquet files.

// Write generates the code for initializing a struct
// with data from a parquet file.
func Write(f fields.Field) string {
	if f.Repeated() {
		return writeRepeated(f)
	}

	if f.Optional() {
		return writeOptional(f)
	}

	return writeRequired(f)
}

// Read generates the code for reading a struct
// and using the data to write to a parquet file.
func Read(f fields.Field) string {
	if f.Repeated() {
		return readRepeated(f)
	}

	if f.Optional() {
		return readOptional(f)
	}

	return readRequired(f)
}
