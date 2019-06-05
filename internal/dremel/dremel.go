package dremel

import "github.com/parsyl/parquet/internal/parse"

func Write(f parse.Field) string {
	if !isOptional(f) {
		return writeRequired(f)
	}
	return writeOptional(f)
}

func Read(f parse.Field) string {
	if !isOptional(f) {
		return readRequired(f)
	}
	return readOptional(f)
}

func isOptional(f parse.Field) bool {
	for _, o := range f.Optionals {
		if o {
			return true
		}
	}
	return false
}
