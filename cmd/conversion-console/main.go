package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/order-schema-console/internal/fixture"
	"example.com/order-schema-console/internal/httpapi"
	"example.com/order-schema-console/internal/service"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	command := "preview"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	svc := service.New()
	switch command {
	case "preview":
		flags := flag.NewFlagSet("preview", flag.ContinueOnError)
		flags.SetOutput(output)
		fixtureName := flags.String("fixture", fixture.DefaultCatalog, "fixture name")
		if err := flags.Parse(args); err != nil {
			return err
		}
		preview, err := svc.Preview(*fixtureName)
		if err != nil {
			return err
		}
		return writeJSON(output, preview)
	case "tables":
		flags := flag.NewFlagSet("tables", flag.ContinueOnError)
		flags.SetOutput(output)
		query := flags.String("query", "", "table name filter")
		cursor := flags.String("cursor", "", "result cursor")
		limit := flags.Int("limit", 50, "page size")
		if err := flags.Parse(args); err != nil {
			return err
		}
		page, err := svc.ListTables(*query, *cursor, *limit)
		if err != nil {
			return err
		}
		return writeJSON(output, page)
	case "execute":
		flags := flag.NewFlagSet("execute", flag.ContinueOnError)
		flags.SetOutput(output)
		fixtureName := flags.String("fixture", fixture.DefaultCatalog, "fixture name")
		skipExisting := flags.Bool("skip-existing", true, "skip tables already in the target")
		if err := flags.Parse(args); err != nil {
			return err
		}
		execution, err := svc.Execute(*fixtureName, *skipExisting)
		if err != nil {
			return err
		}
		return writeJSON(output, execution)
	case "serve":
		flags := flag.NewFlagSet("serve", flag.ContinueOnError)
		flags.SetOutput(output)
		address := flags.String("addr", ":8080", "listen address")
		if err := flags.Parse(args); err != nil {
			return err
		}
		server := &http.Server{Addr: *address, Handler: httpapi.New(svc).Router(), ReadHeaderTimeout: 5 * time.Second}
		log.Printf("order schema console listening on %s", *address)
		return server.ListenAndServe()
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	return nil
}
