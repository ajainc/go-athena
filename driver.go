package athena

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/athena"
)

var (
	openFromSessionMutex sync.Mutex
	openFromSessionCount int
)

// Driver is a sql.Driver. It's intended for db/sql.Open().
type Driver struct {
	cfg *Config
}

func init() {
	var drv driver.Driver = &Driver{}
	sql.Register("athena", drv)
}

// Open should be used via `db/sql.Open("athena", "<params>")`.
// The following parameters are supported in URI query format (k=v&k2=v2&...)
//
// - `db` (required)
// This is the Athena database name. In the UI, this defaults to "default",
// but the driver requires it regardless.
//
// - `output_location` (required)
// This is the S3 location Athena will dump query results in the format
// "s3://bucket/and/so/forth". In the AWS UI, this defaults to
// "s3://aws-athena-query-results-<ACCOUNTID>-<REGION>", but the driver requires it.
//
// - `poll_frequency` (optional)
// Athena's API requires polling to retrieve query results. This is the frequency at
// which the driver will poll for results. It should be a time/Duration.String().
// A completely arbitrary default of "5s" was chosen.
//
// Credentials must be accessible via the SDK's Default Credential Provider Chain.
// For more advanced AWS credentials/session/config management, please supply
// a custom AWS session directly via `athena.Open()`.
func (d *Driver) Open(connStr string) (driver.Conn, error) {
	cfg := d.cfg
	if cfg == nil {
		var err error
		cfg, err = configFromConnectionString(connStr)
		if err != nil {
			return nil, err
		}
	}

	if cfg.PollFrequency == 0 {
		cfg.PollFrequency = 5 * time.Second
	}

	return &conn{
		athena:         athena.New(cfg.Session),
		db:             cfg.Database,
		OutputLocation: cfg.OutputLocation,
		pollFrequency:  cfg.PollFrequency,
	}, nil
}

// Open is a more robust version of `db.Open`, as it accepts a raw aws.Session.
// This is useful if you have a complex AWS session since the driver doesn't
// currently attempt to serialize all options into a string.
func Open(cfg Config) (*sql.DB, error) {
	if cfg.Database == "" {
		return nil, errors.New("db is required")
	}

	if cfg.OutputLocation == "" {
		return nil, errors.New("s3_staging_url is required")
	}

	if cfg.Session == nil {
		return nil, errors.New("session is required")
	}

	// This hack was copied from jackc/pgx. Sorry :(
	// https://github.com/jackc/pgx/blob/70a284f4f33a9cc28fd1223f6b83fb00deecfe33/stdlib/sql.go#L130-L136
	openFromSessionMutex.Lock()
	openFromSessionCount++
	name := fmt.Sprintf("athena-%d", openFromSessionCount)
	openFromSessionMutex.Unlock()

	sql.Register(name, &Driver{&cfg})
	return sql.Open(name, "")
}