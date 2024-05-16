package postgres

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/Permify/permify/internal/storage"
	"github.com/Permify/permify/internal/storage/yugabyte/utils"
	db "github.com/Permify/permify/pkg/database/yugabyte"
	base "github.com/Permify/permify/pkg/pb/base/v1"
)

// SchemaWriter - Structure for SchemaWriter
type SchemaWriter struct {
	database *db.Yugabyte
	// options
	txOptions pgx.TxOptions
}

// NewSchemaWriter creates a new SchemaWriter
func NewSchemaWriter(database *db.Yugabyte) *SchemaWriter {
	return &SchemaWriter{
		database:  database,
		txOptions: pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite},
	}
}

// WriteSchema writes a schema to the database
func (w *SchemaWriter) WriteSchema(ctx context.Context, schemas []storage.SchemaDefinition) (err error) {
	ctx, span := tracer.Start(ctx, "schema-writer.write-schema")
	defer span.End()

	slog.Debug("writing schemas to the database", slog.Any("number_of_schemas", len(schemas)))

	insertBuilder := w.database.Builder.Insert(SchemaDefinitionTable).Columns("name, serialized_definition, version, tenant_id")

	for _, schema := range schemas {
		insertBuilder = insertBuilder.Values(schema.Name, schema.SerializedDefinition, schema.Version, schema.TenantID)
	}

	var query string
	var args []interface{}

	query, args, err = insertBuilder.ToSql()
	if err != nil {
		return utils.HandleError(ctx, span, err, base.ErrorCode_ERROR_CODE_SQL_BUILDER)
	}

	slog.Debug("executing sql insert query", slog.Any("query", query), slog.Any("arguments", args))

	_, err = w.database.WritePool.Exec(ctx, query, args...)
	if err != nil {
		return utils.HandleError(ctx, span, err, base.ErrorCode_ERROR_CODE_EXECUTION)
	}

	slog.Debug("successfully wrote schemas to the database", slog.Any("number_of_schemas", len(schemas)))

	return nil
}
