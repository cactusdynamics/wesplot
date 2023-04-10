package wesplot

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// The pipeline is supposed to start with an io.Reader (likely reading stdin),
// then one or more StringDataReaders will be used to process the strings, This
// is passed to the TextToDataRowReader, which will convert it to a data row.
// This can then be passed to one or more DataRowReaders, which will finally
// pass it to the DataBroadcaster which will emit it to the websocket.

var errIgnoreThisRow = errors.New("ignore this row")

// When Read is called, return an array of strings which are the columns.
type StringReader interface {
	Read(context.Context) ([]string, error)
}

type DataRow struct {
	X  float64
	Ys []float64

	streamEnded bool
	streamErr   error
}

// When Read is called, return the DataRow.
type DataRowReader interface {
	Read(context.Context) (DataRow, error)
	ColumnNames() []string
}

// This implements a StringDataReader and reads an io.Reader using the Golang
// csv module.  This means the input data must strictly conform to CSV data. If
// the input data is not exactly CSV (for example separated by one or more
// spaces), use the RelaxedStringReader.
type CsvStringReader struct {
	input     io.Reader
	csvReader *csv.Reader

	lineCount int
}

func NewCsvStringReader(input io.Reader) *CsvStringReader {
	return &CsvStringReader{
		input:     input,
		csvReader: csv.NewReader(input),
		lineCount: 0,
	}
}

func (r *CsvStringReader) Read(ctx context.Context) ([]string, error) {
	// TODO: context.Cancel
	line, err := r.csvReader.Read()
	if err == io.EOF {
		return nil, io.EOF
	}

	r.lineCount++

	if err != nil {
		logger := logrus.WithFields(logrus.Fields{
			"tag":     "CsvString",
			"line":    line,
			"lineNum": r.lineCount,
		})

		switch err.(type) {
		case *csv.ParseError:
			logger.WithError(err).Debug("unable to parse CSV, ignoring...")
			return nil, errIgnoreThisRow
		default:
			logger.WithError(err).Error("unable to read CSV")
			return nil, err
		}
	}

	return line, nil
}

// This is a more relaxed reader that can split on spaces or commas. However, it does not
// follow string CSV formatting. This is the default.
type RelaxedStringReader struct {
	input   io.Reader
	scanner *bufio.Scanner

	lineCount int
}

func NewRelaxedStringReader(input io.Reader) *RelaxedStringReader {
	return &RelaxedStringReader{
		input:   input,
		scanner: bufio.NewScanner(input),

		lineCount: 0,
	}
}

// Split on either comma or any number of spaces or tabs
var relaxedSplitter = regexp.MustCompile("[ \t]+|,")

func (r *RelaxedStringReader) Read(ctx context.Context) ([]string, error) {
	stillHasData := r.scanner.Scan()
	if !stillHasData {
		return nil, io.EOF
	}

	line := r.scanner.Text()
	err := r.scanner.Err()
	if err != nil {
		logrus.WithField("tag", "RelaxedString").WithError(err).Error("unable to read line")
		return nil, err
	}

	// Return only non-empty lines
	splittedLine := Filter(relaxedSplitter.Split(line, -1), func(value string) bool {
		return len(value) > 0
	})

	return splittedLine, nil
}

// Generates the current unix timestamp in seconds.
func NowXGenerator(line []float64) float64 {
	// Use Micro because we want to preserve the timestamp to at least millisecond
	// accuracy. using time.Now().Unix() will truncate.
	return float64(time.Now().UnixMicro()) / 1000000.0
}

// Creates a DataRowReader based on text input. Unrecognized/unparsable lines
// will be ignored and logged via warnings.
type TextToDataRowReader struct {
	// The input reader object (either CsvStringReader or RelaxedStringReader)
	Input StringReader

	// The x column index. If this is <0, X is generated via XGenerator, which
	// corresponds to the current time stamp in seconds. Note this column
	// will be put into DataRow.X while the rest of the row except this column
	// will be put into DataRow.Ys.
	XIndex int

	// The generator function. Defaults to NowXGenerator.
	XGenerator func([]float64) float64

	// The labels of the columns excluding the X column.
	Columns []string

	// If the input row has a different length than Columns, ignore the row.
	ExpectExactColumnCount bool
}

func (r *TextToDataRowReader) Read(ctx context.Context) (DataRow, error) {
	line, err := r.Input.Read(ctx)
	if err != nil {
		return DataRow{}, err
	}

	logger := logrus.WithFields(logrus.Fields{
		"tag":  "TextToData",
		"line": line,
	})

	dataRow := DataRow{}

	for i, value := range line {
		floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			logger.Warn("cannot parse float, ignoring...")
			return DataRow{}, errIgnoreThisRow
		}

		if i == r.XIndex {
			dataRow.X = floatValue
			continue
		}

		dataRow.Ys = append(dataRow.Ys, floatValue)
	}

	if r.ExpectExactColumnCount && (len(r.Columns) != len(dataRow.Ys)) {
		logger.Warnf("expected column count (%d) is not observed (%d). use `wesplot -n %d` to ensure this row is read", len(r.Columns), len(dataRow.Ys), len(dataRow.Ys))
		return DataRow{}, errIgnoreThisRow
	}

	if r.XIndex < 0 {
		xGenerator := r.XGenerator
		if xGenerator == nil {
			xGenerator = NowXGenerator
		}

		dataRow.X = xGenerator(dataRow.Ys)
	}

	return dataRow, nil
}

func (r *TextToDataRowReader) ColumnNames() []string {
	return r.Columns
}
