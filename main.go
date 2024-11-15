package main

import (
	"context"
	"skripsi/module"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Main Code Here
	core := module.NewCoreModule()
	// core.Helper.LoggerHelper.LogAndContinue("Henlo %s", os.Getenv("POSTGRES_URL"))
	core.WebModule.Init()
	core.WebModule.Serve()
}

func init() {
	godotenv.Load()
	// core := module.NewCoreModule()
	// test(core)
}

func test(core module.CoreModule) {
	type placeholder struct {
		id   int
		text string
	}

	core.Helper.LoggerHelper.SetDebugPrefix()
	core.Database.PostgresDatabase.OpenSingle()
	core.Database.PostgresDatabase.OpenPool()
	defer core.Database.PostgresDatabase.CloseSingle()
	defer core.Database.PostgresDatabase.ClosePool()

	// Single Connection
	testTime := time.Now()
	ps := []placeholder{}
	core.Database.PostgresDatabase.GetConn().Exec(context.Background(), "CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, t text)")
	core.Database.PostgresDatabase.GetConn().Exec(context.Background(), "INSERT INTO test (t) VALUES ('opus'),('untes'),('reimleigh'),('oxys')")
	rows, err := core.Database.PostgresDatabase.GetConn().Query(context.Background(), "SELECT * FROM test")
	if err != nil {
		core.Helper.LoggerHelper.LogErrAndExit(1, err, "Failed Query")
	}

	for rows.Next() {
		var p placeholder
		err := rows.Scan(&p.id, &p.text)
		if err != nil {
			core.Helper.LoggerHelper.LogErrAndExit(1, err, "Failed Scanning")
		}
		ps = append(ps, p)
	}
	core.Helper.LoggerHelper.LogAndContinue("Data: %v", ps)
	core.Helper.LoggerHelper.LogAndContinue("Test Time Elapsed: %dms", time.Since(testTime).Milliseconds())
	core.Helper.LoggerHelper.LogAndContinue("Postgres Database Single Connection OK")
	core.Database.PostgresDatabase.GetPool().Exec(context.Background(), "DROP TABLE IF EXISTS test")

	// Pool Connection
	testTime = time.Now()
	ps = []placeholder{}
	core.Database.PostgresDatabase.GetPool().Exec(context.Background(), "CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, t text)")
	core.Database.PostgresDatabase.GetPool().Exec(context.Background(), "INSERT INTO test (t) VALUES ('kaliper'),('rixev'),('oelp'),('neia'),('itop'),('iemei')")
	rows, err = core.Database.PostgresDatabase.GetPool().Query(context.Background(), "SELECT * FROM test")
	if err != nil {
		core.Helper.LoggerHelper.LogErrAndExit(1, err, "Failed Query")
	}

	for rows.Next() {
		var p placeholder
		err := rows.Scan(&p.id, &p.text)
		if err != nil {
			core.Helper.LoggerHelper.LogErrAndExit(1, err, "Failed Scanning")
		}
		ps = append(ps, p)
	}
	core.Helper.LoggerHelper.LogAndContinue("Data: %v", ps)
	core.Helper.LoggerHelper.LogAndContinue("Test Time Elapsed: %dms", time.Since(testTime).Milliseconds())
	core.Helper.LoggerHelper.LogAndContinue("Postgres Database Pool Connection OK")
	core.Database.PostgresDatabase.GetPool().Exec(context.Background(), "DROP TABLE IF EXISTS test")
}
