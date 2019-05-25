package bq

import (
    "cloud.google.com/go/bigquery"
    "context"
    "log"
)

type BigQueryWriter struct {
    Client *bigquery.Client
    Ctx    context.Context
}

func NewBigQueryWriter(project string) *BigQueryWriter {

    ctx := context.Background()
    bqClient, err := bigquery.NewClient(ctx, project)
    if err != nil {
        log.Fatal(err)
    }

    return &BigQueryWriter{
        Client: bqClient,
        Ctx:    ctx,
    }
}
