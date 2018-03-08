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
	KEY_VISITOR_COUNT = "bigproject.lathif.vstr.cnt"
	KEY_USER_PREFIX   = "bigproject.lathif.usr."
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
	_, err = c.Do("SETEX", KEY_VISITOR_COUNT, 60*60, 1)
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
	c := redisConnect()
	defer c.Close()
	users := getUser(c, filter, 0, 10)
	encoded, _ := json.Marshal(users)
	w.Write(encoded)
}

func getUser(c redis.Conn, filterName *string, offset, limit int) []User {
	userList := getUserRedis(c, filterName, offset, limit)
	if len(userList) == 0 {
		userList = getUserDB(filterName, offset, limit)
		setUserRedis(c, userList, filterName, offset, limit)
	}
	return userList
}

func getUserRedis(c redis.Conn, filterName *string, offset, limit int) []User {
	dataKey := KEY_USER_PREFIX + strconv.Itoa(offset) + "." + strconv.Itoa(limit)
	if filterName != nil && *filterName != "" {
		dataKey = KEY_USER_PREFIX + *filterName + "." + strconv.Itoa(offset) + "." + strconv.Itoa(limit)
	}
	exist, err := redis.Int(c.Do("EXISTS", dataKey))
	checkErr(err, "Redis check user list failure", false)
	userList := []User{}
	if exist > 0 {
		res, err := redis.String(c.Do("GET", dataKey))
		checkErr(err, "Redis get user list failure", false)
		json.Unmarshal([]byte(res), userList)
	}
	return userList
}

func setUserRedis(c redis.Conn, userList []User, filterName *string, offset, limit int) {
	dataKey := KEY_USER_PREFIX + *filterName + "." + strconv.Itoa(offset) + "." + strconv.Itoa(limit)
	dataString, err := json.Marshal(userList)
	checkErr(err, "Json marshal user list failure", false)
	_, err = c.Do("SETEX", dataKey, 15*60, dataString)
	checkErr(err, "Redis set user list failure", false)
}

func getUserDB(filterName *string, offset, limit int) []User { // DB connection
	connectString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DB_USER, DB_PSWD, DB_HOST, DB_NAME)
	db, err = sql.Open("postgres", connectString)
	checkErr(err, "DB connection open failure", true)
	defer db.Close()

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
	http.HandleFunc("/bigproject", handleIndex)
	http.HandleFunc("/data", handleData)
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	log.Fatal(http.ListenAndServe(":12121", nil))
}
