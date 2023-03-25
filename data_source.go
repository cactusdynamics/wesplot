package wesplot

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Arrow? haha it's too much work for this for now.
type DataRow struct {
	Timestamp float64
	Data      []float64
}

type Operator func(DataRow) (DataRow, error)

type DataSource interface {
	Read(context.Context) (DataRow, error)
	Columns() []string
}

var errIgnoreThisRow = errors.New("ignoreThisRow")

type CsvDataSource struct {
	io                  io.Reader
	pipeline            []Operator
	timestampColumn     int
	columns             []string
	dynamicColumnsCount bool

	csvReader *csv.Reader
	logger    logrus.FieldLogger
}

// Creates a new CSV data source.
//
// The data format is expected to be a CSV file that are all floating points.
// Unrecognized/unparsable lines will be ignored and logged via warnings.
//
//   - io: A reader object (usually STDIN)
//   - pipeline: a list of operator to apply to the DataRow to transform it.
//   - timestampColumn: the timestamp column index. If this is <0, the timestamp
//     is assumed to be when the data is read by this code, otherwise, the unix
//     timestamp is assumed to be at that column.
//   - columns: the label of the columns excluding the timestamp column (do not
//     change the order, just exclude the timestamp column whereever it is)
//   - dynamicColumnCount: Don't bother validating the number of columns in the
//     input data. Instead simply grow the column count to the biggest we've
//     seen.
func NewCsvDataSource(io io.Reader, pipeline []Operator, timestampColumn int, columns []string, dynamicColumnCount bool) *CsvDataSource {
	return &CsvDataSource{
		io:                  io,
		pipeline:            pipeline,
		timestampColumn:     timestampColumn,
		columns:             columns,
		dynamicColumnsCount: dynamicColumnCount,
		csvReader:           csv.NewReader(io),
		logger:              logrus.WithField("tag", "CsvDataSource"),
	}
}

func (s *CsvDataSource) Read(ctx context.Context) (DataRow, error) {
	record, err := s.csvReader.Read()
	if err == io.EOF {
		return DataRow{}, io.EOF
	}

	if err != nil {
		switch err.(type) {
		case *csv.ParseError:
			s.logger.WithError(err).Warn("unable to parse") // TODO: log line number and stuff if not already done?
			return DataRow{}, errIgnoreThisRow
		default:
			s.logger.WithError(err).Error("unable to read CSV")
			return DataRow{}, err
		}
	}

	dataRow, err := s.interpretRawData(record)
	if err == errIgnoreThisRow {
		s.logger.WithField("line", record).Warnf("ignoring line due to raw data interpret")
		return DataRow{}, err
	} else if err != nil {
		s.logger.WithField("line", record).WithError(err).Error("failed to interpret raw data") // Shouldn't really happen, right?
		return DataRow{}, err
	}

	if !s.dynamicColumnsCount {
		if len(dataRow.Data) != len(s.columns) {
			s.logger.WithFields(logrus.Fields{
				"line":          record,
				"expectedCount": len(s.columns),
			}).Warn("ignoring data as it doesn't match the expected column count")
			return DataRow{}, errIgnoreThisRow
		}
	}

	// TODO: pipeline is not tested
	for _, operator := range s.pipeline {
		dataRow, err = operator(dataRow)
		if err == errIgnoreThisRow {
			s.logger.Warnf("ignoring line due to operator: %v", record)
			return DataRow{}, err
		} else if err != nil {
			s.logger.WithField("line", record).WithField("operator", operator).WithError(err).Error("failed to apply operator raw data") // Shouldn't really happen, right? also operator name not logged?
			return DataRow{}, err
		}
	}

	return dataRow, nil
}

func (s *CsvDataSource) Columns() []string {
	return s.columns
}

func (s *CsvDataSource) interpretRawData(line []string) (DataRow, error) {
	var dataRow DataRow
	// TODO: timestamp feature is not tested.
	if s.timestampColumn < 0 {
		dataRow.Timestamp = float64(time.Now().UnixMilli())
	}

	for i, value := range line {
		if i == s.timestampColumn {
			continue
		}

		floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return DataRow{}, errIgnoreThisRow
		}

		dataRow.Data = append(dataRow.Data, floatValue)
	}

	return dataRow, nil
}
