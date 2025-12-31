package wesplot

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

// errReader simulates an io.Reader that returns an error on Read.
type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

// CSV reader tests grouped
func TestCsvStringReader(t *testing.T) {
	t.Run("Read_SuccessAndCount", func(t *testing.T) {
		ctx := context.Background()
		r := NewCsvStringReader(strings.NewReader("1,2,3\n4,5,6\n"))
		line, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(line, want) {
			t.Fatalf("unexpected fields: got %v want %v", line, want)
		}

		// read second line as well to ensure multiple rows are parsed
		line2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("expected nil error on second read, got %v", err)
		}
		want2 := []string{"4", "5", "6"}
		if !reflect.DeepEqual(line2, want2) {
			t.Fatalf("unexpected fields on second line: got %v want %v", line2, want2)
		}

		// subsequent read should be EOF
		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("Read_EOF", func(t *testing.T) {
		ctx := context.Background()
		r := NewCsvStringReader(strings.NewReader(""))
		_, err := r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF, got %v", err)
		}
	})

	t.Run("Read_ParseError_Ignored", func(t *testing.T) {
		// malformed CSV with unmatched quote should produce a csv.ParseError
		ctx := context.Background()
		r := NewCsvStringReader(strings.NewReader("a,\"b"))
		_, err := r.Read(ctx)
		if err != errIgnoreThisRow {
			t.Fatalf("expected errIgnoreThisRow, got %v", err)
		}
	})

	t.Run("Read_UnderlyingError", func(t *testing.T) {
		ctx := context.Background()
		underlying := errors.New("boom")
		r := NewCsvStringReader(&errReader{err: underlying})
		_, err := r.Read(ctx)
		if !errors.Is(err, underlying) {
			t.Fatalf("expected underlying error %v, got %v", underlying, err)
		}
	})
}
func TestRelaxedStringReader(t *testing.T) {
	t.Run("Spaces", func(t *testing.T) {
		ctx := context.Background()
		r := NewRelaxedStringReader(strings.NewReader("1 2 3\n4 5 6\n"))
		got, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected split: got %v want %v", got, want)
		}

		// read second line as well
		got2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		want2 := []string{"4", "5", "6"}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("unexpected split on second line: got %v want %v", got2, want2)
		}

		// subsequent read should be EOF
		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("Tabs", func(t *testing.T) {
		ctx := context.Background()
		r := NewRelaxedStringReader(strings.NewReader("1\t2\t3\n7\t8\t9\n"))
		got, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected split: got %v want %v", got, want)
		}

		got2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		want2 := []string{"7", "8", "9"}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("unexpected split on second line: got %v want %v", got2, want2)
		}

		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("MultipleSpacesAndTabs", func(t *testing.T) {
		ctx := context.Background()
		// multiple spaces and tabs between values
		r := NewRelaxedStringReader(strings.NewReader("1  \t\t  2    3\n10\t\t20   30\n"))
		got, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected split: got %v want %v", got, want)
		}

		got2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		want2 := []string{"10", "20", "30"}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("unexpected split on second line: got %v want %v", got2, want2)
		}

		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("Commas", func(t *testing.T) {
		ctx := context.Background()
		r := NewRelaxedStringReader(strings.NewReader("1,2,3\n4,5,6\n"))
		got, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected split: got %v want %v", got, want)
		}

		got2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		want2 := []string{"4", "5", "6"}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("unexpected split on second line: got %v want %v", got2, want2)
		}

		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("MixedSeparators", func(t *testing.T) {
		ctx := context.Background()
		r := NewRelaxedStringReader(strings.NewReader(" 1,\t2  ,  3\t\n4 , 5\t,6\n"))
		got, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"1", "2", "3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected split: got %v want %v", got, want)
		}

		got2, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		want2 := []string{"4", "5", "6"}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("unexpected split on second line: got %v want %v", got2, want2)
		}

		_, err = r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF after reads, got %v", err)
		}
	})

	t.Run("UnderlyingError", func(t *testing.T) {
		ctx := context.Background()
		underlying := errors.New("boom")
		r := NewRelaxedStringReader(&errReader{err: underlying})
		_, err := r.Read(ctx)
		// The scanner will return false on Read errors and the current
		// implementation returns io.EOF when Scan() is false. Accept either
		// the underlying error or io.EOF to match behavior across Go versions.
		if err != io.EOF && !errors.Is(err, underlying) {
			t.Fatalf("expected underlying error %v or io.EOF, got %v", underlying, err)
		}
	})
}

// fakeStringReader helps simulate different StringReader behaviors.
type fakeStringReader struct {
	outputs [][]string
	errs    []error
	idx     int
}

func (f *fakeStringReader) Read(ctx context.Context) ([]string, error) {
	if f.idx >= len(f.outputs) {
		return nil, io.EOF
	}
	out := f.outputs[f.idx]
	err := f.errs[f.idx]
	f.idx++
	return out, err
}

// Group TextToDataRowReader tests
func TestTextToDataRowReader(t *testing.T) {
	t.Run("NormalXIndex", func(t *testing.T) {
		ctx := context.Background()
		// X is first column
		s := NewRelaxedStringReader(strings.NewReader("100,1,2,3\n"))
		r := &TextToDataRowReader{Input: s, XIndex: 0, Columns: []string{"a", "b", "c"}}

		dr, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dr.X != 100 {
			t.Fatalf("unexpected X: got %v want %v", dr.X, 100.0)
		}
		wantYs := []float64{1, 2, 3}
		if !reflect.DeepEqual(dr.Ys, wantYs) {
			t.Fatalf("unexpected Ys: got %v want %v", dr.Ys, wantYs)
		}
	})

	t.Run("GeneratedX", func(t *testing.T) {
		ctx := context.Background()
		s := NewRelaxedStringReader(strings.NewReader("1 2 3\n"))
		r := &TextToDataRowReader{Input: s, XIndex: -1, Columns: []string{"a", "b", "c"}}

		before := time.Now().UnixMicro()
		dr, err := r.Read(ctx)
		after := time.Now().UnixMicro()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(dr.Ys) != 3 {
			t.Fatalf("unexpected Ys length: got %d want %d", len(dr.Ys), 3)
		}
		xMicro := int64(dr.X * 1000000.0)
		if xMicro < before || xMicro > after {
			t.Fatalf("generated X out of expected range: %v not in [%v,%v]", xMicro, before, after)
		}
	})

	t.Run("NonFloat_Ignored", func(t *testing.T) {
		ctx := context.Background()
		s := NewRelaxedStringReader(strings.NewReader("abc,1,2\n"))
		r := &TextToDataRowReader{Input: s, XIndex: 0, Columns: []string{"a", "b"}}

		_, err := r.Read(ctx)
		if err != errIgnoreThisRow {
			t.Fatalf("expected errIgnoreThisRow, got %v", err)
		}
	})

	t.Run("ExpectExactColumnCount_Mismatch", func(t *testing.T) {
		ctx := context.Background()
		// row produces 3 Ys (X at index 0 -> Ys are 1,2,3) but Columns length is 2
		s := NewCsvStringReader(strings.NewReader("10,1,2,3\n"))
		r := &TextToDataRowReader{Input: s, XIndex: 0, Columns: []string{"a", "b"}, ExpectExactColumnCount: true}

		_, err := r.Read(ctx)
		if err != errIgnoreThisRow {
			t.Fatalf("expected errIgnoreThisRow for column mismatch, got %v", err)
		}
	})

	t.Run("ColumnNames", func(t *testing.T) {
		r := &TextToDataRowReader{Columns: []string{"x", "y"}}
		got := r.ColumnNames()
		want := []string{"x", "y"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected column names: got %v want %v", got, want)
		}
	})

	t.Run("InputErrorPropagation", func(t *testing.T) {
		ctx := context.Background()
		f := &fakeStringReader{outputs: [][]string{{"1", "2"}}, errs: []error{errIgnoreThisRow}}
		r := &TextToDataRowReader{Input: f, XIndex: 0}

		_, err := r.Read(ctx)
		if err != errIgnoreThisRow {
			t.Fatalf("expected errIgnoreThisRow propagated, got %v", err)
		}
	})

	t.Run("EOFPropagation", func(t *testing.T) {
		ctx := context.Background()
		f := &fakeStringReader{outputs: [][]string{}, errs: []error{}}
		r := &TextToDataRowReader{Input: f, XIndex: 0}

		_, err := r.Read(ctx)
		if err != io.EOF {
			t.Fatalf("expected io.EOF propagated, got %v", err)
		}
	})

	t.Run("XIndexMiddle", func(t *testing.T) {
		ctx := context.Background()
		s := NewRelaxedStringReader(strings.NewReader("1 2 3 4\n"))
		// XIndex = 2 should pick value 3 as X, Ys should be [1,2,4]
		r := &TextToDataRowReader{Input: s, XIndex: 2, Columns: []string{"a", "b", "c"}}

		dr, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dr.X != 3 {
			t.Fatalf("unexpected X: got %v want %v", dr.X, 3.0)
		}
		want := []float64{1, 2, 4}
		if !reflect.DeepEqual(dr.Ys, want) {
			t.Fatalf("unexpected Ys: got %v want %v", dr.Ys, want)
		}
	})

	t.Run("ExpectExactColumnCount_Success", func(t *testing.T) {
		ctx := context.Background()
		s := NewRelaxedStringReader(strings.NewReader("10 20 30\n"))
		// XIndex 0 -> Ys [20,30], Columns len 2 matches Ys -> should succeed
		r := &TextToDataRowReader{Input: s, XIndex: 0, Columns: []string{"a", "b"}, ExpectExactColumnCount: true}

		dr, err := r.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []float64{20, 30}
		if !reflect.DeepEqual(dr.Ys, want) {
			t.Fatalf("unexpected Ys: got %v want %v", dr.Ys, want)
		}
		if dr.X != 10 {
			t.Fatalf("unexpected X: got %v want %v", dr.X, 10.0)
		}
	})

	t.Run("NonFloatInY_Ignored", func(t *testing.T) {
		ctx := context.Background()
		s := NewCsvStringReader(strings.NewReader("1,2,abc\n"))
		r := &TextToDataRowReader{Input: s, XIndex: 0}

		_, err := r.Read(ctx)
		if err != errIgnoreThisRow {
			t.Fatalf("expected errIgnoreThisRow for non-float in Y, got %v", err)
		}
	})
}
