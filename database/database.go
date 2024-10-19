package database

type Database struct {
	PostgresDatabase PostgresDatabase
}

func NewDatabase() Database {
	return Database{
		PostgresDatabase: NewPostgresDatabase(),
	}
}