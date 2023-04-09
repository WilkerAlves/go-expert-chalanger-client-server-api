package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	file   string = "database.db"
	create string = `
	  CREATE TABLE IF NOT EXISTS quotations (
	  id varchar(255) PRIMARY KEY,
	  bid varchar(255)
	  );
	`
)

type BodyResponse struct {
	Usdbrl Quotation `json:"USDBRL"`
}

type Quotation struct {
	Bid string `json:"bid"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", file)
	if err != nil {
		log.Fatalf("erro ao abrir a conexão. %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(create); err != nil {
		log.Fatalf("erro para criar tabela. %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", HandlerQuotation)
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("erro ao subir o servidor. %v", err)
	}

}

func HandlerQuotation(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 200*time.Millisecond)
	defer cancel()
	quotation, err := GetQuotation(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			res.WriteHeader(http.StatusRequestTimeout)
			return
		}
		res.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("erro para obter a cotação. %v", err)
		return
	}

	ctx, cancel = context.WithTimeout(req.Context(), 10*time.Millisecond)
	defer cancel()
	err = InsertQuotation(req.Context(), quotation.Bid)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			res.WriteHeader(http.StatusRequestTimeout)
			return
		}
		res.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("erro ao inserir. %v", err)
		return
	}

	response := make(map[string]string)
	response["bid"] = quotation.Bid

	jsonResp, err := json.Marshal(response)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(jsonResp)
}

func InsertQuotation(ctx context.Context, quotation string) error {
	stmt, err := db.PrepareContext(ctx, "INSERT INTO quotations VALUES(?,?);")
	if err != nil {
		log.Fatalf("erro para preparar insert. %v", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, uuid.NewString(), quotation)
	if err != nil {
		log.Fatalf("erro para executar o insert. %v", err)
		return err
	}

	return nil
}

func GetQuotation(ctx context.Context) (*Quotation, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"https://economia.awesomeapi.com.br/json/last/USD-BRL",
		nil,
	)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var resBody BodyResponse
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &resBody)
	if err != nil {
		return nil, err
	}

	return &resBody.Usdbrl, nil
}
