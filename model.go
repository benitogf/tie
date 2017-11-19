package main

import (
    "database/sql"
    "github.com/lib/pq"
)

type product struct {
    ID    int64     `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
    Description string `json:"description"`
    Quantity int64 `json:"quantity"`
    Files []int64 `json:"files"`
    Created string `json:"created"`
}

type file struct {
    ID    int64     `json:"id"`
    Name  string  `json:"name"`
    Type string `json:"type"`
    Description string `json:"description"`
    Data string `json:"data"`
    Created string `json:"created"`
}

type item struct {
    Key  string  `json:"key"`
    Value string `json:"value"`
}

// Products

func (p *product) createProduct(db *sql.DB) error {
    err := db.QueryRow(
        "INSERT INTO products(name, price, description, quantity, files) VALUES($1, $2, $3, $4, $5) RETURNING id",
        p.Name, p.Price, p.Description, p.Quantity, ToStrf(p.Files)).Scan(&p.ID)

    if err != nil {
        return err
    }

    return nil
}

func (p *product) getProduct(db *sql.DB) error {
    // https://stackoverflow.com/a/40906849
    arr := pq.Int64Array{}
    err := db.QueryRow("SELECT name, price, description, quantity, files, created FROM products WHERE id=$1",
        p.ID).Scan(&p.Name, &p.Price, &p.Description, &p.Quantity, &arr, &p.Created)
    p.Files = []int64(arr)
    return err
}

func (p *product) updateProduct(db *sql.DB) error {
    _, err :=
        db.Exec("UPDATE products SET name=$1, price=$2, description=$3, quantity=$4, files=$5 WHERE id=$6",
            p.Name, p.Price, &p.Description, p.Quantity, ToStrf(p.Files), p.ID)

    return err
}

func (p *product) deleteProduct(db *sql.DB) error {
    _, err := db.Exec("DELETE FROM products WHERE id=$1", p.ID)

    return err
}

func getProducts(db *sql.DB, start, count int) ([]product, error) {
    rows, err := db.Query(
        "SELECT id, name, price, description, quantity, files, created FROM products LIMIT $1 OFFSET $2",
        count, start)

    if err != nil {
        return nil, err
    }

    defer rows.Close()

    products := []product{}

    for rows.Next() {
        var p product
        if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description, &p.Quantity, &p.Files, &p.Created); err != nil {
            return nil, err
        }
        products = append(products, p)
    }

    return products, nil
}

// Files

func (f *file) createFile(db *sql.DB) error {
    err := db.QueryRow(
        "INSERT INTO files(name, type, description, data) VALUES($1, $2, $3, $4) RETURNING id",
        f.Name, f.Type, f.Description, f.Data).Scan(&f.ID)

    if err != nil {
        return err
    }

    return nil
}

func (f *file) getFile(db *sql.DB) error {
    return db.QueryRow("SELECT name, type, description, data, created FROM files WHERE id=$1",
        f.ID).Scan(&f.Name, &f.Type, &f.Description, &f.Data, &f.Created)
}

func (f *file) deleteFile(db *sql.DB) error {
    _, err := db.Exec("DELETE FROM files WHERE id=$1", f.ID)

    return err
}
