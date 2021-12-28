package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/konstantinfoerster/card-importer-go/internal/config"
	"runtime"
	"strings"
	"sync"
	"time"
)

var once sync.Once
var dbConn *DBConnection

type DBConnection struct {
	Ctx    context.Context
	Conn   DBConn
	pgxCon *pgxpool.Pool
}

func Connect(ctx context.Context, config *config.Database) (*DBConnection, error) {
	var err error

	once.Do(func() {
		c, pErr := pgxpool.ParseConfig(config.ConnectionUrl())
		if pErr != nil {
			err = pErr
			return
		}
		c.MaxConnLifetime = time.Second * 5
		c.MaxConnIdleTime = time.Millisecond * 500
		c.HealthCheckPeriod = time.Millisecond * 500
		c.MaxConns = int32(runtime.NumCPU()) + 5 // + 5 just in case

		pool, cErr := pgxpool.ConnectConfig(ctx, c)
		if cErr != nil {
			err = cErr
			return
		}

		pErr = pool.Ping(ctx)
		if pErr != nil {
			err = pErr
			return
		}

		dbConn = &DBConnection{
			Ctx:    ctx,
			Conn:   pool,
			pgxCon: pool,
		}
	})

	return dbConn, err
}

func (d *DBConnection) Close() error {
	d.pgxCon.Close()
	once = sync.Once{}
	return nil
}

func (d *DBConnection) WithTransaction(f func(conn *DBConnection) error) error {
	switch v := d.Conn.(type) {
	case pgx.Tx:
		// start transaction inside transaction
		return v.BeginFunc(d.Ctx, func(t pgx.Tx) error {
			dbCon := &DBConnection{
				Ctx:    d.Ctx,
				Conn:   t,
				pgxCon: d.pgxCon,
			}
			return f(dbCon)
		})
	default:
		opts := pgx.TxOptions{AccessMode: pgx.ReadWrite, IsoLevel: pgx.ReadCommitted}
		return d.pgxCon.BeginTxFunc(d.Ctx, opts, func(t pgx.Tx) error {
			dbCon := &DBConnection{
				Ctx:    d.Ctx,
				Conn:   t,
				pgxCon: d.pgxCon,
			}
			return f(dbCon)
		})
	}
}

func (d *DBConnection) Cleanup() error {
	tables := []string{
		"card_block_translation",
		"card_block",

		"card_set_translation",
		"card_set",

		"card_super_type",
		"card_sub_type",
		"card_card_type",

		"card_translation",
		"card",

		"card_type_translation",
		"card_type",
		"super_type_translation",
		"super_type",
		"sub_type",
		"sub_type_translation",
	}
	_, err := d.Conn.Exec(d.Ctx, fmt.Sprintf("TRUNCATE %s RESTART IDENTITY", strings.Join(tables, ",")))
	return err
}

// DBConn implemented by pgx.Conn and pgx.Tx
type DBConn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}
