package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/grokify/gotilla/fmt/fmtutil"
	"github.com/joho/godotenv"

	au "github.com/grokify/go-aha/ahautil"
)

func main() {
	err := godotenv.Load(os.Getenv("ENV_PATH"))
	if err != nil {
		panic(err)
	}

	apis := au.NewClientAPIs(os.Getenv("AHA_ACCOUNT"), os.Getenv("AHA_API_KEY"))

	api := apis.APIClient.ProductsApi
	ctx := context.Background()

	params := map[string]interface{}{
		"page":    int32(1),
		"perPage": int32(500),
	}

	info, resp, err := api.GetProducts(ctx, params)

	if err != nil {
		log.Fatal("Error retrieving features")
	}

	fmt.Println(resp.StatusCode)
	fmtutil.PrintJSON(info)

	fmt.Printf("Found %v products\n", len(info.Products))

	fmt.Println("===")

	for _, prod := range info.Products {
		fmtutil.PrintJSON(prod)

		prod, resp, err := api.GetProduct(ctx, prod.Id)
		if err != nil {
			log.Fatal("Error retrieving product")
		} else if resp.StatusCode >= 300 {
			log.Fatal(fmt.Sprintf("Error calling API: %v", resp.StatusCode))
		}

		fmtutil.PrintJSON(prod)

		break
	}

	fmt.Println("DONE")
}