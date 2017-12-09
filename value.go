package athena

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/athena"
)

const (
	// TimestampFormat is the Go time layout string for an Athena `timestamp`.
	TimestampFormat = "2006-01-02 03:04:05.999"
)

func convertRow(columns []*athena.ColumnInfo, in []*athena.Datum, ret []driver.Value) error {
	for i, val := range in {
		coerced, err := convertValue(*columns[i].Type, val.VarCharValue)
		if err != nil {
			return err
		}

		ret[i] = coerced
	}

	return nil
}

func convertValue(athenaType string, rawValue *string) (interface{}, error) {
	if rawValue == nil {
		return nil, nil
	}

	val := *rawValue
	switch athenaType {
	case "smallint":
		return strconv.ParseInt(val, 10, 16)
	case "integer":
		return strconv.ParseInt(val, 10, 32)
	case "bigint":
		return strconv.ParseInt(val, 10, 64)
	case "boolean":
		switch val {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		return nil, fmt.Errorf("cannot parse '%s' as boolean", val)
	case "double":
		return strconv.ParseFloat(val, 64)
	case "varchar", "string":
		return val, nil
	case "timestamp":
		return time.Parse(TimestampFormat, val)
	default:
		panic(fmt.Errorf("unknown type `%s` with value %s", athenaType, val))
	}
}