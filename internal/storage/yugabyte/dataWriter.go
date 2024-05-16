package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/Masterminds/squirrel"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Permify/permify/internal/storage/yugabyte/snapshot"
	"github.com/Permify/permify/internal/storage/yugabyte/types"
	"github.com/Permify/permify/internal/storage/yugabyte/utils"
	"github.com/Permify/permify/internal/validation"
	"github.com/Permify/permify/pkg/bundle"
	"github.com/Permify/permify/pkg/database"
	db "github.com/Permify/permify/pkg/database/yugabyte"
	base "github.com/Permify/permify/pkg/pb/base/v1"
	"github.com/Permify/permify/pkg/token"
	"github.com/Permify/permify/pkg/tuple"
)

// DataWriter - Structure for Data Writer
type DataWriter struct {
	database *db.Yugabyte
	// options
	txOptions pgx.TxOptions
}

func NewDataWriter(database *db.Yugabyte) *DataWriter {
	return &DataWriter{
		database:  database,
		txOptions: pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite},
	}
}

// Write method writes a collection of tuples and attributes to the database for a specific tenant.
// It returns an EncodedSnapToken upon successful write or an error if the write fails.
func (w *DataWriter) Write(
	ctx context.Context,
	tenantID string,
	tupleCollection *database.TupleCollection,
	attributeCollection *database.AttributeCollection,
) (token token.EncodedSnapToken, err error) {
	// Start a new tracing span for this operation.
	ctx, span := tracer.Start(ctx, "data-writer.write")
	defer span.End() // Ensure that the span is ended when the function returns.

	// Log the start of a data write operation.
	slog.Debug("writing data for tenant_id", slog.String("tenant_id", tenantID), "max retries", slog.Any("max_retries", w.database.GetMaxRetries()))

	// Check if the total number of tuples and attributes exceeds the maximum allowed per write.
	if len(tupleCollection.GetTuples())+len(attributeCollection.GetAttributes()) > w.database.GetMaxDataPerWrite() {
		return nil, errors.New(base.ErrorCode_ERROR_CODE_MAX_DATA_PER_WRITE_EXCEEDED.String())
	}

	// Retry loop for handling transient errors like serialization issues.
	for i := 0; i <= w.database.GetMaxRetries(); i++ {
		// Attempt to write the data to the database.
		tkn, err := w.write(ctx, tenantID, tupleCollection, attributeCollection)
		if err != nil {
			// Check if the error is due to serialization, and if so, retry.
			if utils.IsSerializationRelatedError(err) || pgconn.SafeToRetry(err) {
				slog.Warn("serialization error occurred", slog.String("tenant_id", tenantID), slog.Int("retry", i))
				utils.WaitWithBackoff(ctx, tenantID, i)
				continue // Retry the operation.
			}
			// If the error is not serialization-related, handle it and return.
			return nil, utils.HandleError(ctx, span, err, base.ErrorCode_ERROR_CODE_DATASTORE)
		}
		// If to write is successful, return the token.
		return tkn, nil
	}

	// Log an error if the operation failed after reaching the maximum number of retries.
	slog.Error("max retries reached", slog.Any("error", errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())))

	// Return an error indicating that the maximum number of retries has been reached.
	return nil, errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())
}

// Delete method removes data from the database based on the provided tuple and attribute filters.
// It returns an EncodedSnapToken upon successful deletion or an error if the deletion fails.
func (w *DataWriter) Delete(
	ctx context.Context,
	tenantID string,
	tupleFilter *base.TupleFilter,
	attributeFilter *base.AttributeFilter,
) (token.EncodedSnapToken, error) {
	// Start a new tracing span for this delete operation.
	ctx, span := tracer.Start(ctx, "data-writer.delete")
	defer span.End() // Ensure that the span is ended when the function returns.

	// Log the start of a data deletion operation.
	slog.Debug("deleting data for tenant_id", slog.String("tenant_id", tenantID), "max retries", slog.Any("max_retries", w.database.GetMaxRetries()))

	// Retry loop for handling transient errors like serialization issues.
	for i := 0; i <= w.database.GetMaxRetries(); i++ {
		// Attempt to delete the data from the database.
		tkn, err := w.delete(ctx, tenantID, tupleFilter, attributeFilter)
		if err != nil {
			// Check if the error is due to serialization, and if so, retry.
			if utils.IsSerializationRelatedError(err) || pgconn.SafeToRetry(err) {
				slog.Warn("serialization error occurred", slog.String("tenant_id", tenantID), slog.Int("retry", i))
				utils.WaitWithBackoff(ctx, tenantID, i)
				continue // Retry the operation.
			}
			// If the error is not serialization-related, handle it and return.
			return nil, utils.HandleError(ctx, span, err, base.ErrorCode_ERROR_CODE_DATASTORE)
		}
		// If the delete operation is successful, return the token.
		return tkn, nil
	}

	// Log an error if the operation failed after reaching the maximum number of retries.
	slog.Debug("max retries reached", slog.Any("error", errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())))

	// Return an error indicating that the maximum number of retries has been reached.
	return nil, errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())
}

// RunBundle executes a bundle of operations in the context of a given tenant.
// It returns an EncodedSnapToken upon successful completion or an error if the operation fails.
func (w *DataWriter) RunBundle(
	ctx context.Context,
	tenantID string,
	arguments map[string]string,
	b *base.DataBundle,
) (token.EncodedSnapToken, error) {
	// Start a new tracing span for this operation.
	ctx, span := tracer.Start(ctx, "data-writer.run-bundle")
	defer span.End() // Ensure that the span is ended when the function returns.

	// Log the start of running a bundle operation.
	slog.Debug("running bundle for tenant_id", slog.String("tenant_id", tenantID), "max retries", slog.Any("max_retries", w.database.GetMaxRetries()))

	// Retry loop for handling transient errors like serialization issues.
	for i := 0; i <= w.database.GetMaxRetries(); i++ {
		// Attempt to run the bundle operation.
		tkn, err := w.runBundle(ctx, tenantID, arguments, b)
		if err != nil {
			// Check if the error is due to serialization, and if so, retry.
			if utils.IsSerializationRelatedError(err) || pgconn.SafeToRetry(err) {
				slog.Warn("serialization error occurred", slog.String("tenant_id", tenantID), slog.Int("retry", i))
				utils.WaitWithBackoff(ctx, tenantID, i)
				continue // Retry the operation.
			}
			// If the error is not serialization-related, handle it and return.
			return nil, utils.HandleError(ctx, span, err, base.ErrorCode_ERROR_CODE_DATASTORE)
		}
		// If the operation is successful, return the token.
		return tkn, nil
	}

	// Log an error if the operation failed after reaching the maximum number of retries.
	slog.Error("max retries reached", slog.Any("error", errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())))

	// Return an error indicating that the maximum number of retries has been reached.
	return nil, errors.New(base.ErrorCode_ERROR_CODE_ERROR_MAX_RETRIES.String())
}

// write handles the database writing of tuple and attribute collections for a given tenant.
// It returns an EncodedSnapToken upon successful write or an error if the write fails.
func (w *DataWriter) write(
	ctx context.Context,
	tenantID string,
	tupleCollection *database.TupleCollection,
	attributeCollection *database.AttributeCollection,
) (token token.EncodedSnapToken, err error) {
	var tx pgx.Tx
	tx, err = w.database.WritePool.BeginTx(ctx, w.txOptions)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var xid types.XID8
	err = tx.QueryRow(ctx, utils.TransactionTemplate, tenantID).Scan(&xid)
	if err != nil {
		return nil, err
	}

	slog.Debug("retrieved transaction", slog.Any("xid", xid), "for tenant", slog.Any("tenant_id", tenantID))

	slog.Debug("processing tuples and executing insert query")

	if len(tupleCollection.GetTuples()) > 0 {

		tuplesInsertBuilder := w.database.Builder.Insert(RelationTuplesTable).Columns("entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_tx_id, tenant_id")

		deleteClauses := squirrel.Or{}

		titer := tupleCollection.CreateTupleIterator()
		for titer.HasNext() {
			t := titer.GetNext()
			srelation := t.GetSubject().GetRelation()
			if srelation == tuple.ELLIPSIS {
				srelation = ""
			}

			// Build the condition for this tuple.
			condition := squirrel.Eq{
				"entity_type":      t.GetEntity().GetType(),
				"entity_id":        t.GetEntity().GetId(),
				"relation":         t.GetRelation(),
				"subject_type":     t.GetSubject().GetType(),
				"subject_id":       t.GetSubject().GetId(),
				"subject_relation": srelation,
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)

			tuplesInsertBuilder = tuplesInsertBuilder.Values(t.GetEntity().GetType(), t.GetEntity().GetId(), t.GetRelation(), t.GetSubject().GetType(), t.GetSubject().GetId(), srelation, xid, tenantID)
		}

		tDeleteBuilder := w.database.Builder.Update(RelationTuplesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var tdquery string
		var tdargs []interface{}

		tdquery, tdargs, err = tDeleteBuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, tdquery, tdargs...)
		if err != nil {
			return nil, err
		}

		var tiquery string
		var tiargs []interface{}

		tiquery, tiargs, err = tuplesInsertBuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, tiquery, tiargs...)
		if err != nil {
			return nil, err
		}
	}

	if len(attributeCollection.GetAttributes()) > 0 {

		attributesInsertBuilder := w.database.Builder.Insert(AttributesTable).Columns("entity_type, entity_id, attribute, value, created_tx_id, tenant_id")

		deleteClauses := squirrel.Or{}

		aiter := attributeCollection.CreateAttributeIterator()
		for aiter.HasNext() {
			a := aiter.GetNext()

			m := jsonpb.Marshaler{}
			jsonStr, err := m.MarshalToString(a.GetValue())
			if err != nil {
				return nil, err
			}

			// Build the condition for this attribute.
			condition := squirrel.Eq{
				"entity_type": a.GetEntity().GetType(),
				"entity_id":   a.GetEntity().GetId(),
				"attribute":   a.GetAttribute(),
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)

			attributesInsertBuilder = attributesInsertBuilder.Values(a.GetEntity().GetType(), a.GetEntity().GetId(), a.GetAttribute(), jsonStr, xid, tenantID)
		}

		aDeleteBuilder := w.database.Builder.Update(AttributesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var adquery string
		var adargs []interface{}

		adquery, adargs, err = aDeleteBuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, adquery, adargs...)
		if err != nil {
			return nil, err
		}

		var aquery string
		var aargs []interface{}

		aquery, aargs, err = attributesInsertBuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, aquery, aargs...)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	slog.Debug("data successfully written to the database")

	return snapshot.NewToken(xid).Encode(), nil
}

// delete handles the deletion of tuples and attributes from the database based on provided filters.
// It returns an EncodedSnapToken upon successful deletion or an error if the deletion fails.
func (w *DataWriter) delete(
	ctx context.Context,
	tenantID string,
	tupleFilter *base.TupleFilter,
	attributeFilter *base.AttributeFilter,
) (token token.EncodedSnapToken, err error) {
	var tx pgx.Tx
	tx, err = w.database.WritePool.BeginTx(ctx, w.txOptions)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var xid types.XID8
	err = tx.QueryRow(ctx, utils.TransactionTemplate, tenantID).Scan(&xid)
	if err != nil {
		return nil, err
	}

	slog.Debug("retrieved transaction", slog.Any("xid", xid), "for tenant", slog.Any("tenant_id", tenantID))

	slog.Debug("processing tuple and executing update query")

	if !validation.IsTupleFilterEmpty(tupleFilter) {
		tbuilder := w.database.Builder.Update(RelationTuplesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{"expired_tx_id": "0", "tenant_id": tenantID})
		tbuilder = utils.TuplesFilterQueryForUpdateBuilder(tbuilder, tupleFilter)

		var tquery string
		var targs []interface{}

		tquery, targs, err = tbuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, tquery, targs...)
		if err != nil {
			return nil, err
		}
	}

	slog.Debug("processing attribute and executing update query")

	if !validation.IsAttributeFilterEmpty(attributeFilter) {
		abuilder := w.database.Builder.Update(AttributesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{"expired_tx_id": "0", "tenant_id": tenantID})
		abuilder = utils.AttributesFilterQueryForUpdateBuilder(abuilder, attributeFilter)

		var aquery string
		var aargs []interface{}

		aquery, aargs, err = abuilder.ToSql()
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx, aquery, aargs...)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	slog.Debug("data successfully deleted from the database")

	return snapshot.NewToken(xid).Encode(), nil
}

// runBundle executes a series of operations defined in a DataBundle within a single database transaction.
// It returns an EncodedSnapToken upon successful execution of all operations or an error if any operation fails.
func (w *DataWriter) runBundle(
	ctx context.Context,
	tenantID string,
	arguments map[string]string,
	b *base.DataBundle,
) (token token.EncodedSnapToken, err error) {
	var tx pgx.Tx
	tx, err = w.database.WritePool.BeginTx(ctx, w.txOptions)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var xid types.XID8
	err = tx.QueryRow(ctx, utils.TransactionTemplate, tenantID).Scan(&xid)
	if err != nil {
		return nil, err
	}

	slog.Debug("retrieved transaction", slog.Any("xid", xid), "for tenant", slog.Any("tenant_id", tenantID))

	for _, op := range b.GetOperations() {
		tb, ab, err := bundle.Operation(arguments, op)
		if err != nil {
			return nil, err
		}

		err = w.runOperation(ctx, tx, xid, tenantID, tb, ab)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return snapshot.NewToken(xid).Encode(), nil
}

// runOperation processes and executes database operations defined in TupleBundle and AttributeBundle within a given transaction.
func (w *DataWriter) runOperation(
	ctx context.Context,
	tx pgx.Tx,
	xid types.XID8,
	tenantID string,
	tb database.TupleBundle,
	ab database.AttributeBundle,
) (err error) {
	slog.Debug("processing bundles queries")

	if len(tb.Write.GetTuples()) > 0 {

		tuplesInsertBuilder := w.database.Builder.Insert(RelationTuplesTable).Columns("entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_tx_id, tenant_id")

		deleteClauses := squirrel.Or{}

		titer := tb.Write.CreateTupleIterator()
		for titer.HasNext() {
			t := titer.GetNext()
			srelation := t.GetSubject().GetRelation()
			if srelation == tuple.ELLIPSIS {
				srelation = ""
			}

			// Build the condition for this tuple.
			condition := squirrel.Eq{
				"entity_type":      t.GetEntity().GetType(),
				"entity_id":        t.GetEntity().GetId(),
				"relation":         t.GetRelation(),
				"subject_type":     t.GetSubject().GetType(),
				"subject_id":       t.GetSubject().GetId(),
				"subject_relation": srelation,
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)

			tuplesInsertBuilder = tuplesInsertBuilder.Values(t.GetEntity().GetType(), t.GetEntity().GetId(), t.GetRelation(), t.GetSubject().GetType(), t.GetSubject().GetId(), srelation, xid, tenantID)
		}

		tDeleteBuilder := w.database.Builder.Update(RelationTuplesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var tdquery string
		var tdargs []interface{}

		tdquery, tdargs, err = tDeleteBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, tdquery, tdargs...)
		if err != nil {
			return err
		}

		var tiquery string
		var tiargs []interface{}

		tiquery, tiargs, err = tuplesInsertBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, tiquery, tiargs...)
		if err != nil {
			return err
		}
	}

	if len(ab.Write.GetAttributes()) > 0 {

		attributesInsertBuilder := w.database.Builder.Insert(AttributesTable).Columns("entity_type, entity_id, attribute, value, created_tx_id, tenant_id")

		deleteClauses := squirrel.Or{}

		aiter := ab.Write.CreateAttributeIterator()
		for aiter.HasNext() {
			a := aiter.GetNext()

			m := jsonpb.Marshaler{}
			jsonStr, err := m.MarshalToString(a.GetValue())
			if err != nil {
				return err
			}

			// Build the condition for this tuple.
			condition := squirrel.Eq{
				"entity_type": a.GetEntity().GetType(),
				"entity_id":   a.GetEntity().GetId(),
				"attribute":   a.GetAttribute(),
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)

			attributesInsertBuilder = attributesInsertBuilder.Values(a.GetEntity().GetType(), a.GetEntity().GetId(), a.GetAttribute(), jsonStr, xid, tenantID)
		}

		tDeleteBuilder := w.database.Builder.Update(AttributesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var adquery string
		var adargs []interface{}

		adquery, adargs, err = tDeleteBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, adquery, adargs...)
		if err != nil {
			return err
		}

		var aquery string
		var aargs []interface{}

		aquery, aargs, err = attributesInsertBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, aquery, aargs...)
		if err != nil {
			return err
		}
	}

	if len(tb.Delete.GetTuples()) > 0 {

		deleteClauses := squirrel.Or{}

		titer := tb.Delete.CreateTupleIterator()
		for titer.HasNext() {
			t := titer.GetNext()
			srelation := t.GetSubject().GetRelation()
			if srelation == tuple.ELLIPSIS {
				srelation = ""
			}

			// Build the condition for this tuple.
			condition := squirrel.Eq{
				"entity_type":      t.GetEntity().GetType(),
				"entity_id":        t.GetEntity().GetId(),
				"relation":         t.GetRelation(),
				"subject_type":     t.GetSubject().GetType(),
				"subject_id":       t.GetSubject().GetId(),
				"subject_relation": srelation,
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)
		}

		tDeleteBuilder := w.database.Builder.Update(RelationTuplesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var tquery string
		var targs []interface{}

		tquery, targs, err = tDeleteBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, tquery, targs...)
		if err != nil {
			return err
		}
	}

	if len(ab.Delete.GetAttributes()) > 0 {

		deleteClauses := squirrel.Or{}

		aiter := ab.Delete.CreateAttributeIterator()
		for aiter.HasNext() {
			a := aiter.GetNext()

			// Build the condition for this tuple.
			condition := squirrel.Eq{
				"entity_type": a.GetEntity().GetType(),
				"entity_id":   a.GetEntity().GetId(),
				"attribute":   a.GetAttribute(),
			}

			// Add the condition to the OR slice.
			deleteClauses = append(deleteClauses, condition)

		}

		aDeleteBuilder := w.database.Builder.Update(AttributesTable).Set("expired_tx_id", xid).Where(squirrel.Eq{
			"expired_tx_id": "0",
			"tenant_id":     tenantID,
		}).Where(deleteClauses)

		var tquery string
		var targs []interface{}

		tquery, targs, err = aDeleteBuilder.ToSql()
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, tquery, targs...)
		if err != nil {
			return err
		}
	}

	return nil
}
