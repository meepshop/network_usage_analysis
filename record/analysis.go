package record

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/meepshop/network_usage_analysis/connection"
	"google.golang.org/api/iterator"
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

	var lastSuccHour *time.Time
	err = conn.Pg.QueryRow("SELECT MAX(file_datetime) FROM throughput_record WHERE success = true").Scan(&lastSuccHour)
	if err != nil {
		log.Fatalf("Analysis.Pg.QueryRow() err: %+v", err)
	} else if lastSuccHour == nil {
		// storage的資料最早從2018-05-25 00:00:00開始
		t, _ := time.Parse("2006-01-02 15:04:05", "2018-05-24 23:00:00")
		lastSuccHour = &t
	}

	curExecHour := *lastSuccHour
	for {
		execTime := time.Now()
		curExecHour = curExecHour.Add(time.Hour * time.Duration(1))

		log.Println("curExecHour:", curExecHour.Format("2006-01-02 15:04:05"))

		_, err := conn.Pg.Exec("INSERT INTO throughput_record(file_datetime, exec_time) VALUES ($1, $2)", curExecHour, execTime)
		if err != nil {
			log.Fatalf("Analysis.Pg.INSERT() err: %+v", err)
		}

		if err := procHourlyData(execTime, curExecHour); err != nil {
			break
		}
	}
}

func procHourlyData(execTime, curExecHour time.Time) error {

	tx, err := conn.Pg.Begin()
	if err != nil {
		log.Printf("procHourlyData.Pg.Begin() err: %+v", err)
		return err
	}
	defer tx.Rollback()

	// 執行前刪除原有資料
	_, err = tx.Exec("DELETE FROM throughput WHERE utc = $1", curExecHour)
	if err != nil {
		log.Printf("procHourlyData.Pg.Delete() err: %+v", err)
		return err
	}
	_, err = tx.Exec("DELETE FROM throughput_old WHERE utc = $1", curExecHour)
	if err != nil {
		log.Printf("procHourlyData.Pg.Delete() err: %+v", err)
		return err
	}

	total := map[string]map[string]ThroughputItem{
		"new": map[string]ThroughputItem{},
		"old": map[string]ThroughputItem{},
	}

	// 拿該小時的所有檔案
	objs := bucket.Objects(conn.Ctx, &storage.Query{
		Prefix: curExecHour.Format("requests/2006/01/02/15:00:00_15:"),
	})

	for {
		attrs, err := objs.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("procHourlyData.objs.Next() err: %+v", err)
			return err
		}

		total, err = analysisLogFile(attrs.Name, total)
		if err != nil {
			return err
		}
	}

	if len(total["new"]) == 0 && len(total["old"]) == 0 {
		log.Println(curExecHour, "no any record")
		return errors.New("no any record")
	}

	// 寫入DB
	if err := batchInsert(tx, curExecHour, total); err != nil {
		return err
	}

	// 更新記錄為成功
	_, err = tx.Exec("UPDATE throughput_record SET success = true WHERE file_datetime = $1 AND exec_time = $2", curExecHour, execTime)
	if err != nil {
		log.Printf("procHourlyData.tx.Update() err: %+v", err)
		return err
	}

	tx.Commit()
	return nil
}

func analysisLogFile(fileName string, total map[string]map[string]ThroughputItem) (map[string]map[string]ThroughputItem, error) {

	obj := bucket.Object(fileName)
	_, err := obj.Attrs(conn.Ctx)
	if err != nil {
		log.Println("no file:", fileName)
		return total, err
	}

	r, err := obj.NewReader(conn.Ctx)
	if err != nil {
		log.Printf("analysisLogFile.obj.NewReader() err: %+v", err)
		return total, err
	}
	defer r.Close()

	// 逐行分析檔案
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {

		var rc *Record
		if err := json.Unmarshal([]byte(scanner.Text()), &rc); err != nil {
			log.Printf("analysisLogFile.json.Unmarshal() err: %+v", err)
			return total, err
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
		log.Printf("analysisLogFile.scanner err: %+v", err)
		return total, err
	}

	return total, nil
}

func batchInsert(tx *sql.Tx, utc time.Time, total map[string]map[string]ThroughputItem) error {

	query := ""
	values := []string{}
	for _, ti := range total["new"] {
		value := fmt.Sprintf("('%s', '%s', %d, %d)", ti.storeid, utc.Format("2006-01-02 15:04:05"), ti.requestsize, ti.responsesize)
		values = append(values, value)
	}
	query = fmt.Sprintf(`INSERT INTO throughput(storeid, utc, requestsize, responsesize) VALUES %s`, strings.Join(values, ","))

	_, err := tx.Exec(query)
	if err != nil {
		log.Printf("batchInsert.INSERT INTO throughput() err: %+v query: %s", err, query)
		return err
	}

	values = []string{}
	for _, ti := range total["old"] {
		value := fmt.Sprintf("('%s', '%s', %d, %d)", ti.cname, utc.Format("2006-01-02 15:04:05"), ti.requestsize, ti.responsesize)
		values = append(values, value)
	}
	query = fmt.Sprintf(`INSERT INTO throughput_old(cname, utc, requestsize, responsesize) VALUES %s`, strings.Join(values, ","))

	// conn.PgStage.Exec(query)
	_, err = tx.Exec(query)
	if err != nil {
		log.Printf("batchInsert.INSERT INTO throughput_old() err: %+v query: %s", err, query)
		return err
	}

	return nil
}
