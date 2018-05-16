package record

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/meepshop/network_usage_analysis/connection"
)

var conn *connection.Connection
var bucket *storage.BucketHandle

type ThroughputItem struct {
	storeid      string
	cname        string
	system       string
	requestsize  int64
	responsesize int64
	utc          time.Time
}

func Analysis() {

	conn = connection.NewConnection()
	defer conn.Close()

	client, err := storage.NewClient(conn.Ctx)
	if err != nil {
		log.Fatalf("Analysis.storage.NewClient() err: %+v", err)
	}
	defer client.Close()
	bucket = client.Bucket("network_usage")

	var lastSuccFile *time.Time
	err = conn.Pg.QueryRow("SELECT MAX(file_datetime) FROM throughput_record WHERE success = true").Scan(&lastSuccFile)
	if err != nil {
		log.Fatalf("Analysis.Pg.QueryRow() err: %+v", err)
	} else if lastSuccFile == nil {
		// storage的資料最早從2018-01-22 05:00:00開始
		t, _ := time.Parse("2006-01-02 15:04:05", "2018-01-22 04:00:00")
		lastSuccFile = &t
	}

	curExecFile := *lastSuccFile
	for {
		execTime := time.Now()
		curExecFile = curExecFile.Add(time.Hour * time.Duration(1))

		log.Println("curExecFile:", curExecFile.Format("2006-01-02 15:04:05"))

		_, err := conn.Pg.Exec("INSERT INTO throughput_record(file_datetime, exec_time) VALUES ($1, $2)", curExecFile, execTime)
		if err != nil {
			log.Fatalf("Analysis.Pg.INSERT() err: %+v", err)
		}

		if err := procHourlyData(execTime, curExecFile); err != nil {
			break
		}
	}
}

func procHourlyData(execTime, curExecFile time.Time) error {

	obj := bucket.Object(curExecFile.Format("requests/2006/01/02/15:00:00_15") + ":59:59_S0.json")
	_, err := obj.Attrs(conn.Ctx)
	if err != nil {
		log.Println("no file:", curExecFile.Format("requests/2006/01/02/15:00:00_15")+":59:59_S0.json")
		return err
	}

	r, err := obj.NewReader(conn.Ctx)
	if err != nil {
		log.Printf("procHourlyData.obj.NewReader() err: %+v", err)
		return err
	}
	defer r.Close()

	tx, err := conn.Pg.Begin()
	if err != nil {
		log.Printf("procHourlyData.Pg.Begin() err: %+v", err)
		return err
	}
	defer tx.Rollback()

	// 執行前刪除原有資料
	conn.PgStage.Exec("DELETE FROM throughput WHERE utc = $1", curExecFile)
	_, err = tx.Exec("DELETE FROM throughput WHERE utc = $1", curExecFile)
	if err != nil {
		log.Printf("procHourlyData.Pg.Delete() err: %+v", err)
		return err
	}
	_, err = tx.Exec("DELETE FROM throughput_old WHERE utc = $1", curExecFile)
	if err != nil {
		log.Printf("procHourlyData.Pg.Delete() err: %+v", err)
		return err
	}

	total := map[string]map[string]ThroughputItem{
		"new": map[string]ThroughputItem{},
		"old": map[string]ThroughputItem{},
	}

	// 逐行分析檔案
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var rc *Record
		if err := json.Unmarshal([]byte(scanner.Text()), &rc); err != nil {
			log.Printf("procHourlyData.json.Unmarshal() err: %+v", err)
			return err
		}

		ti, err := rc.ParseToThroughputItem()
		if err != nil {
			// 遇到不符合規則的網址 跳過不處理
			// log.Println(err)
			continue
		}

		if ti.system == "new" {
			if _, exist := total["new"][ti.storeid]; exist {
				ti.requestsize += total["new"][ti.storeid].requestsize
				ti.responsesize += total["new"][ti.storeid].responsesize
			}
			total["new"][ti.storeid] = ti
		} else {
			if _, exist := total["old"][ti.cname]; exist {
				ti.requestsize += total["old"][ti.cname].requestsize
				ti.responsesize += total["old"][ti.cname].responsesize
			}
			total["old"][ti.cname] = ti
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("procHourlyData.scanner err: %+v", err)
		return err
	}

	// 寫入DB
	if err := batchInsert(tx, curExecFile, total); err != nil {
		log.Fatalf("Analysis.Pg.UPSERT() err: %+v", err)
		return err
	}

	// 更新記錄為成功
	_, err = tx.Exec("UPDATE throughput_record SET success = true WHERE file_datetime = $1 AND exec_time = $2", curExecFile, execTime)
	if err != nil {
		log.Printf("procHourlyData.tx.Update() err: %+v", err)
		return err
	}

	tx.Commit()
	return nil
}

func batchInsert(tx *sql.Tx, utc time.Time, total map[string]map[string]ThroughputItem) error {

	query := ""
	values := []string{}
	for _, ti := range total["new"] {
		value := fmt.Sprintf("('%s', '%s', %d, %d)", ti.storeid, utc.Format("2006-01-02 15:04:05"), ti.requestsize, ti.responsesize)
		values = append(values, value)
	}
	query = fmt.Sprintf(
		`INSERT INTO throughput(storeid, utc, requestsize, responsesize) VALUES %s
			ON CONFLICT (storeid, utc) DO UPDATE SET requestsize = throughput.requestsize + EXCLUDED.requestsize, responsesize = throughput.responsesize + EXCLUDED.responsesize`,
		strings.Join(values, ","))

	conn.PgStage.Exec(query)
	_, err := tx.Exec(query)
	if err != nil {
		return err
	}

	values = []string{}
	for _, ti := range total["old"] {
		value := fmt.Sprintf("('%s', '%s', %d, %d)", ti.cname, utc.Format("2006-01-02 15:04:05"), ti.requestsize, ti.responsesize)
		values = append(values, value)
	}
	query = fmt.Sprintf(
		`INSERT INTO throughput_old(cname, utc, requestsize, responsesize) VALUES %s
			ON CONFLICT (cname, utc) DO UPDATE SET requestsize = throughput_old.requestsize + EXCLUDED.requestsize, responsesize = throughput_old.responsesize + EXCLUDED.responsesize`,
		strings.Join(values, ","))

	// conn.PgStage.Exec(query)
	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
