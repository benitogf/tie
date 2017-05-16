package main_test

import (
    "os"
    "log"
    "fmt"
    "bytes"
    "strconv"
    "testing"
    "net/http"
    "database/sql"
    "encoding/json"
    "net/http/httptest"
    "github.com/benitogf/pasticho"
    _ "github.com/lib/pq"
)

var DB_USER = os.Getenv("PASTICHO_DB_USER")
var DB_NAME = os.Getenv("PASTICHO_DB_NAME")
var DB_PASSWORD = os.Getenv("PASTICHO_DB_PASSWORD")
var PGPW = os.Getenv("PG_PASSWORD")

const userCreateQuery = `DO
    $body$
    BEGIN
       IF NOT EXISTS (
          SELECT *
          FROM   pg_catalog.pg_user
          WHERE  usename = '%s') THEN

          CREATE ROLE %s LOGIN PASSWORD '%s';
       END IF;
    END
    $body$;`

const databaseCreateQuery = `DO
    $do$
    DECLARE
      _db TEXT := '%s';
      _user TEXT := '%s';
      _password TEXT := '%s';
    BEGIN
      CREATE EXTENSION IF NOT EXISTS dblink; -- enable extension
      IF EXISTS (SELECT 1 FROM pg_database WHERE datname = _db) THEN
        RAISE NOTICE 'Database already exists';
      ELSE
        PERFORM dblink_connect('host=localhost user=' || _user || ' password=' || _password || ' dbname=' || current_database());
        PERFORM dblink_exec('CREATE DATABASE ' || _db);
      END IF;
    END
    $do$`

const grantUserDatabaseQuery = `GRANT ALL PRIVILEGES ON DATABASE %s TO %s;`

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS products
    (
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
    )`

var app main.App
var token string

func initialize() {
  connectionString := fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=disable", "postgres", PGPW, "postgres")
  var PGDB, err = sql.Open("postgres", connectionString)
  if err != nil {
      log.Fatal(err)
  }
  userCreateString := fmt.Sprintf(userCreateQuery, DB_USER, DB_USER, DB_PASSWORD)
  if _, err := PGDB.Exec(userCreateString); err != nil {
    log.Fatal(err)
  }
  databaseCreateString := fmt.Sprintf(databaseCreateQuery, DB_NAME, "postgres", PGPW)
  if _, err := PGDB.Exec(databaseCreateString); err != nil {
    log.Fatal(err)
  }
  grantUserDatabaseString := fmt.Sprintf(grantUserDatabaseQuery, DB_NAME, DB_USER)
  if _, err := PGDB.Exec(grantUserDatabaseString); err != nil {
    log.Fatal(err)
  }
  if err := PGDB.Close(); err != nil {
    log.Fatal(err)
  }
  if _, err := app.DB.Exec(tableCreationQuery); err != nil {
    log.Fatal(err)
  }
}

func clearTable() {
  app.DB.Exec("DELETE FROM products")
  app.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
  reqRecorder := httptest.NewRecorder()
  app.Router.ServeHTTP(reqRecorder, req)

  return reqRecorder
}

func checkResponseCode(t *testing.T, expected, actual int) {
  if expected != actual {
    t.Errorf("Expected response code %d. Got %d\n", expected, actual)
  }
}

func addProducts(count int) {
  if count < 1 {
    count = 1
  }

  for i := 0; i < count; i++ {
    app.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product "+strconv.Itoa(i), (i+1.0)*10)
  }
}

func TestAuthorize(t *testing.T) {
  payload := []byte(`{"account":"pasticho","password":"202cb962ac59075b964b07152d234b70"}`)
  req, _ := http.NewRequest("POST", "/authorize", bytes.NewBuffer(payload))
  response := executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)

  dec := json.NewDecoder(response.Body)
	var credentials map[string]interface{}
	if err := dec.Decode(&credentials); err != nil {
		t.Error("error decoding authorize response")
	}
  if credentials["token"] == nil {
    t.Errorf("Expected a token in the credentials response %s", credentials["token"])
  } else {
    token = credentials["token"].(string)
  }
}

func TestEmptyTable(t *testing.T) {
  clearTable()

  req, _ := http.NewRequest("GET", "/products", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)

  if body := response.Body.String(); body != "[]" {
    t.Errorf("Expected an empty array. Got %s", body)
  }
}

func TestCreateProduct(t *testing.T) {
  clearTable()

  payload := []byte(`{"name":"test product","price":11.22}`)

  req, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(payload))
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusCreated, response.Code)

  var m map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &m)

  if m["name"] != "test product" {
      t.Errorf("Expected product name to be 'test product'. Got '%v'", m["name"])
  }

  if m["price"] != 11.22 {
      t.Errorf("Expected product price to be '11.22'. Got '%v'", m["price"])
  }

  // the id is compared to 1.0 because JSON unmarshaling converts numbers to
  // floats, when the target is a map[string]interface{}
  if m["id"] != 1.0 {
      t.Errorf("Expected product ID to be '1'. Got '%v'", m["id"])
  }
}

func TestUpdateProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)
  var originalProduct map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &originalProduct)

  payload := []byte(`{"name":"test product - updated name","price":11.22}`)

  req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(payload))
  req.Header.Set("Authorization", "Bearer " + token)
  response = executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)

  var m map[string]interface{}
  json.Unmarshal(response.Body.Bytes(), &m)

  if m["id"] != originalProduct["id"] {
    t.Errorf("Expected the id to remain the same (%v). Got %v", originalProduct["id"], m["id"])
  }

  if m["name"] == originalProduct["name"] {
    t.Errorf("Expected the name to change from '%v' to '%v'. Got '%v'", originalProduct["name"], m["name"], m["name"])
  }

  if m["price"] == originalProduct["price"] {
    t.Errorf("Expected the price to change from '%v' to '%v'. Got '%v'", originalProduct["price"], m["price"], m["price"])
  }
}

func TestDeleteProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)
  checkResponseCode(t, http.StatusOK, response.Code)

  req, _ = http.NewRequest("DELETE", "/product/1", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response = executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)

  req, _ = http.NewRequest("GET", "/product/1", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response = executeRequest(req)
  checkResponseCode(t, http.StatusNotFound, response.Code)
}

func TestGetProduct(t *testing.T) {
  clearTable()
  addProducts(1)

  req, _ := http.NewRequest("GET", "/product/1", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusOK, response.Code)
}

func TestGetNonExistentProduct(t *testing.T) {
  clearTable()

  req, _ := http.NewRequest("GET", "/product/11", nil)
  req.Header.Set("Authorization", "Bearer " + token)
  response := executeRequest(req)

  checkResponseCode(t, http.StatusNotFound, response.Code)

  var m map[string]string
  json.Unmarshal(response.Body.Bytes(), &m)
  if m["message"] != "Product not found" {
    t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
  }
}

func TestMain(m *testing.M) {
  app = main.App{}
  app.Initialize(
    DB_USER,
    DB_PASSWORD,
    DB_NAME)

  initialize()

  code := m.Run()

  clearTable()

  os.Exit(code)
}
