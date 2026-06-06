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
	Docs     []bson.M
	Total    int64
	Page     int
	PageSize int
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

// DatabaseDropped is dispatched after a drop-database attempt completes.
type DatabaseDropped struct {
	DB  string
	Err error
}

// ---- status / error ----

// StatusUpdate carries a transient message for the status bar.
type StatusUpdate struct {
	Text  string
	IsErr bool
}
