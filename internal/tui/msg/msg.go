// Package msg defines all inter-panel messages for the bubbletea event loop.
package msg

import "go.mongodb.org/mongo-driver/v2/bson"

// DatabaseInfo carries metadata about one database.
type DatabaseInfo struct {
	Name       string
	SizeOnDisk int64
	Empty      bool
}

// CollectionInfo carries metadata about one collection.
type CollectionInfo struct {
	Name string
	Type string // "collection" | "view" | "timeseries"
}

// PageResult is a single page of documents returned from MongoDB.
type PageResult struct {
	Docs       []bson.M
	Total      int64
	Page       int
	PageSize   int
	DurationMs int64
}

// ---- async result messages ----

// DatabasesLoaded is dispatched when the database list finishes loading.
type DatabasesLoaded struct {
	DBs []DatabaseInfo
	Err error
}

// CollectionsLoaded is dispatched when a database's collections finish loading.
type CollectionsLoaded struct {
	DB          string
	Collections []CollectionInfo
	Err         error
}

// DocumentsLoaded is dispatched when a page of documents finishes loading.
type DocumentsLoaded struct {
	Result PageResult
	Err    error
}

// ---- navigation / intent messages ----

// CollectionSelected is emitted by the sidebar when the user picks a collection.
type CollectionSelected struct {
	DB         string
	Collection string
}

// DocumentSelected is emitted by the documents panel when the user opens a doc.
type DocumentSelected struct {
	Doc bson.M
}

// FilterChanged is emitted by the documents panel when the active filter changes.
type FilterChanged struct {
	Filter bson.M
	Sort   bson.D
	Expr   string // raw filter expression for display in status bar
}

// ---- aggregation ----

// PipelineReady is dispatched by the ExecProcess callback after the user saves
// a pipeline file. It carries the parsed pipeline so the app can fire the
// actual aggregate command as a separate async cmd (the same split CRUD uses).
type PipelineReady struct {
	Pipeline     bson.A
	PipelineText string // raw JSON for re-run prefill
	Err          error  // file-read / JSON-parse error (before any DB call)
}

// AggregateResult is returned after running a pipeline.
type AggregateResult struct {
	Docs         []bson.M
	Err          error
	PipelineText string // echoed back from PipelineReady so Model can store it
}

// ---- indexes ----

// IndexInfo describes a single index on a collection.
type IndexInfo struct {
	Name       string
	Keys       bson.D
	Unique     bool
	Sparse     bool
	TTLSeconds int32 // -1 = not a TTL index
}

// CollectionStats holds lightweight collection metadata.
type CollectionStats struct {
	DocCount   int64
	IndexCount int
}

// IndexesLoaded is dispatched when the index list finishes loading.
type IndexesLoaded struct {
	DB         string
	Collection string
	Indexes    []IndexInfo
	Stats      CollectionStats
	Err        error
}

// IndexEditorDone is returned by the ExecProcess callback after the
// user edits an index definition file.
type IndexEditorDone struct {
	Keys   bson.D
	Unique bool
	Sparse bool
	Err    error
}

// IndexCreated confirms a successful index creation.
type IndexCreated struct {
	Name string
	Err  error
}

// IndexDropped confirms a successful index drop.
type IndexDropped struct {
	Err error
}

// ---- CRUD editor ----

// EditorDone is returned by the ExecProcess callback after the user's
// $EDITOR closes. Doc is nil when the user aborted without saving.
type EditorDone struct {
	Doc    bson.M
	IsNew  bool        // true → insert; false → replace
	OrigID interface{} // original _id for replace operations
	Err    error       // file/JSON parse error (shown in status bar)
}

// DocumentCreated confirms a successful insert.
type DocumentCreated struct {
	InsertedID interface{}
	Err        error
}

// DocumentUpdated confirms a successful replace.
type DocumentUpdated struct {
	Err error
}

// DocumentDeleted confirms a successful delete.
type DocumentDeleted struct {
	Err error
}

// BulkDeleted confirms a successful bulk delete.
type BulkDeleted struct {
	Count int64
	Err   error
}

// DatabaseDropped is dispatched after a drop-database attempt completes.
type DatabaseDropped struct {
	DB  string
	Err error
}

// ExportDone is dispatched after an export (JSON or CSV) completes.
type ExportDone struct {
	Path  string
	Count int
	Err   error
}

// CollectionStatsDetail holds detailed collection statistics.
type CollectionStatsDetail struct {
	DocCount    int64
	TotalSize   int64
	AvgDocSize  float64
	StorageSize int64
	IndexCount  int
	IndexSize   int64
}

// CollectionStatsLoaded is dispatched after collection stats are loaded.
type CollectionStatsLoaded struct {
	DB    string
	Col   string
	Stats CollectionStatsDetail
	Err   error
}

// CollectionCreated is dispatched after a collection creation attempt.
type CollectionCreated struct {
	DB  string
	Col string
	Err error
}

// CollectionDropped is dispatched after a collection drop attempt.
type CollectionDropped struct {
	DB  string
	Col string
	Err error
}

// CollectionRenamed is dispatched after a collection rename attempt.
type CollectionRenamed struct {
	DB     string
	OldCol string
	NewCol string
	Err    error
}

// ---- status / error ----

// StatusUpdate carries a transient message for the status bar.
type StatusUpdate struct {
	Text  string
	IsErr bool
}

// ClearFlash is sent by a timer command to dismiss a transient status message.
type ClearFlash struct{}

// ---- explain plan ----

// ExplainStats holds the key figures extracted from an explain("executionStats") result.
type ExplainStats struct {
	DB, Col         string
	IndexUsed       string // empty string means COLLSCAN
	NReturned       int64
	DocsExamined    int64
	KeysExamined    int64
	ExecutionTimeMs int64
	Raw             bson.M // full explain output for detail rendering
	Err             error
}

// ExplainLoaded is dispatched when the explain result arrives.
type ExplainLoaded struct {
	Stats ExplainStats
}

// ---- schema inference ----

// TypeFreq pairs a BSON type name with its occurrence count in sampled docs.
type TypeFreq struct {
	Type  string
	Count int
}

// SchemaField describes one field across the sampled documents.
type SchemaField struct {
	Name  string
	Types []TypeFreq // sorted by Count desc
	Count int        // number of sampled docs containing this field
}

// SchemaResult carries the output of SampleSchema.
type SchemaResult struct {
	DB, Col    string
	Fields     []SchemaField
	SampleSize int // actual number of docs sampled
	Err        error
}

// SchemaLoaded is dispatched when schema inference completes.
type SchemaLoaded struct {
	Result SchemaResult
}

// ---- import ----

// ImportDone is dispatched after a bulk import attempt completes.
type ImportDone struct {
	Inserted int
	Failed   int
	Err      error // first batch error, if any
}
