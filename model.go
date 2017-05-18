package main

import (
    "database/sql"
)

type product struct {
    ID    int     `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
    Description string `json:"description"`
    Quantity int `json:"quantity"`
}

func (p *product) getProduct(db *sql.DB) error {
    return db.QueryRow("SELECT name, price, description, quantity FROM products WHERE id=$1",
        p.ID).Scan(&p.Name, &p.Price, &p.Description, &p.Quantity)
}

func (p *product) updateProduct(db *sql.DB) error {
    _, err :=
        db.Exec("UPDATE products SET name=$1, price=$2, description=$3, quantity=$4 WHERE id=$5",
            p.Name, p.Price, &p.Description, p.Quantity, p.ID)

    return err
}

func (p *product) deleteProduct(db *sql.DB) error {
    _, err := db.Exec("DELETE FROM products WHERE id=$1", p.ID)

    return err
}

func (p *product) createProduct(db *sql.DB) error {
    err := db.QueryRow(
        "INSERT INTO products(name, price, description, quantity) VALUES($1, $2, $3, $4) RETURNING id",
        p.Name, p.Price, p.Description, p.Quantity).Scan(&p.ID)

    if err != nil {
        return err
    }

    return nil
}

func getProducts(db *sql.DB, start, count int) ([]product, error) {
    rows, err := db.Query(
        "SELECT id, name, price, description, quantity FROM products LIMIT $1 OFFSET $2",
        count, start)

    if err != nil {
        return nil, err
    }

    defer rows.Close()

    products := []product{}

    for rows.Next() {
        var p product
        if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description, &p.Quantity); err != nil {
            return nil, err
        }
        products = append(products, p)
    }

    return products, nil
}
