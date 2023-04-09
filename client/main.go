package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Quotation struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Fatalf("Erro ao criar a requisição. %v", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Erro ao fazer requisição. %v", err)
	}
	defer res.Body.Close()

	var quotation Quotation
	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Erro ao converter para bytes. %v", err)
	}

	err = json.Unmarshal(b, &quotation)
	if err != nil {
		log.Fatalf("Erro ao fazer o Unmarshal. %v", err)
	}

	file, err := os.Create("cotacoes.txt")
	if err != nil {
		log.Fatalf("Erro para criar o arquivo. %v", err)
	}

	_, err = file.Write([]byte(fmt.Sprintf("Dólar: %s", quotation.Bid)))
	if err != nil {
		log.Fatalf("Erro para escrever no arquivo. %v", err)
	}
}
