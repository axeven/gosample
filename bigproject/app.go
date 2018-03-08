package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/garyburd/redigo/redis"
	_ "github.com/lib/pq"
)

const (
	DB_HOST           = "devel-postgre.tkpd"
	DB_USER           = "ja161003"
	DB_PSWD           = "beteinga94"
	DB_NAME           = "tokopedia-user"
	KEY_VISITOR_COUNT = "visitor.count.lathif"
)

var db *sql.DB
var err error

func checkErr(err error, msg string, exitOnError bool) {
	if err != nil {
		if exitOnError {
			log.Fatalln(msg, err)
		} else {
			log.Println(msg, err)
		}
	}
}

type User struct {
	ID        int        `db:"user_id"`
	Name      *string    `db:"full_name"`
	MSISDN    *string    `db:"msisdn"`
	Email     *string    `db:"user_email"`
	BirthDate *time.Time `db:"birth_date"`
	Created   *time.Time `db:"create_time"`
	Updated   *time.Time `db:"update_time"`
}

func redisConnect() redis.Conn {
	c, err := redis.Dial("tcp", "devel-redis.tkpd:6379")
	checkErr(err, "Redis connection failure", true)
	return c
}

func incVisitor(c redis.Conn) int {
	exist, err := redis.Int(c.Do("EXISTS", KEY_VISITOR_COUNT))
	checkErr(err, "Redis check visitor failure", false)
	if exist > 0 {
		var res = 1
		res, err = redis.Int(c.Do("INCR", KEY_VISITOR_COUNT))
		checkErr(err, "Redis incr visitor failure", false)
		return res
	}
	_, err = c.Do("SET", KEY_VISITOR_COUNT, 1)
	checkErr(err, "Redis set visitor error", false)
	return 1

}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template.html")
	checkErr(err, "Reading template failure", false)
	redisConn := redisConnect()
	defer redisConn.Close()
	visitor := incVisitor(redisConn)
	tmpl.ExecuteTemplate(w, "template", visitor)

}

func handleData(w http.ResponseWriter, r *http.Request) {
	filter := new(string)
	if keys, ok := r.URL.Query()["filter"]; ok {
		*filter = keys[0]
	}
	users := getUser(filter, 0, 10)
	encoded, _ := json.Marshal(users)
	w.Write(encoded)
}

func getUser(filterName *string, offset, limit int) []User {
	query := "SELECT user_id, full_name, msisdn, user_email, birth_date, create_time, update_time FROM ws_user"
	if filterName != nil && *filterName != "" {
		query = query + " WHERE LOWER(full_name) LIKE '%" + strings.ToLower(*filterName) + "%' "
	}
	query = query + " LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
	rows, err := db.Query(query)
	checkErr(err, "Retrieving data failure", true)
	userList := []User{}
	for rows.Next() {
		u := &User{}
		err := rows.Scan(&u.ID, &u.Name, &u.MSISDN, &u.Email, &u.BirthDate, &u.Created, &u.Updated)
		checkErr(err, "Retrieving row failure", true)
		userList = append(userList, *u)
	}
	return userList
}

func main() {
	// DB connection
	connectString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DB_USER, DB_PSWD, DB_HOST, DB_NAME)
	db, err = sql.Open("postgres", connectString)
	checkErr(err, "DB connection open failure", true)
	defer db.Close()

	http.HandleFunc("/bigproject", handleIndex)
	http.HandleFunc("/data", handleData)
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	log.Fatal(http.ListenAndServe(":12121", nil))
}
