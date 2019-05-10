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
	Ctx context.Context
	Pg  *sql.DB
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

	return &Connection{ctx, pg}
}

func (c *Connection) Close() {
	c.Pg.Close()
}

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s environment variable not set.", k)
	}
	return v
}
