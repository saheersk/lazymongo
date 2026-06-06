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

// ---- status / error ----

// StatusUpdate carries a transient message for the status bar.
type StatusUpdate struct {
	Text  string
	IsErr bool
}
