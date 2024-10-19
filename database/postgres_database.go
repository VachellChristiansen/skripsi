package database

import (
	"context"
	"skripsi/helper"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDatabase interface {
	OpenSingle()
	OpenPool()
	GetConn() *pgx.Conn
	GetPool() *pgxpool.Pool
	CloseSingle()
	ClosePool()
}

type PostgresDatabaseImpl struct {
	log  helper.LoggerHelper
	conn *pgx.Conn
	pool *pgxpool.Pool
}

func NewPostgresDatabase() PostgresDatabase {
	log := helper.NewLoggerHelper()
	return &PostgresDatabaseImpl{
		log: log,
	}
}

func (db *PostgresDatabaseImpl) OpenSingle() {
	conn, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		db.log.LogErrAndExit(2, err, "Failed opening postgres database connection")
	}

	db.conn = conn
}

func (db *PostgresDatabaseImpl) OpenPool() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		db.log.LogErrAndExit(2, err, "Failed opening postgres database pool")
	}

	db.pool = pool
}

func (db *PostgresDatabaseImpl) GetConn() *pgx.Conn {
	return db.conn
}

func (db *PostgresDatabaseImpl) GetPool() *pgxpool.Pool {
	return db.pool
}

func (db *PostgresDatabaseImpl) CloseSingle() {
	if db.conn != nil {
		db.conn.Close(context.Background())
	}
}

func (db *PostgresDatabaseImpl) ClosePool() {
	if db.pool != nil {
		db.pool.Close()
	}
}
