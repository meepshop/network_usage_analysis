package connection

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Connection struct {
	Ctx     context.Context
	Pg      *sql.DB
	PgStage *sql.DB
}

func NewConnection() *Connection {

	ctx := context.Background()

	pgHost := mustGetenv("POSTGRES_HOST")
	pgUser := mustGetenv("POSTGRES_USER")
	pgPwd := mustGetenv("POSTGRES_PASSWORD")
	dataSource := fmt.Sprintf("postgres://%s:%s@%s:5432/meepshop?sslmode=disable", pgUser, pgPwd, pgHost)
	pg, err := sql.Open("postgres", dataSource)
	if err != nil {
		log.Println(dataSource)
		log.Fatal(err)
	}

	pgsHost := mustGetenv("POSTGRES_STAGE_HOST")
	pgsUser := mustGetenv("POSTGRES_STAGE_USER")
	pgsPwd := mustGetenv("POSTGRES_STAGE_PASSWORD")
	sDataSource := fmt.Sprintf("postgres://%s:%s@%s:5432/meepshop?sslmode=disable", pgsUser, pgsPwd, pgsHost)
	pgs, err := sql.Open("postgres", sDataSource)
	if err != nil {
		log.Println(sDataSource)
		log.Fatal(err)
	}

	return &Connection{ctx, pg, pgs}
}

func (c *Connection) Close() {
	c.Pg.Close()
	c.PgStage.Close()
}

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s environment variable not set.", k)
	}
	return v
}
