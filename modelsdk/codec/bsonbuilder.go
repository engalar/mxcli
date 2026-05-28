package codec

import "go.mongodb.org/mongo-driver/v2/bson"

// BSONDocBuilder holds a raw bson.D document for interop with the
// BSONDocument helper methods in bsondoc.go (ToBuilder,
// AppendToVersionedArray, InsertWidgetsNear, ReplaceWidgetByName, …).
//
// Prefer BSONDocument for building new documents — it provides a
// richer, type-safe API. BSONDocBuilder exists only as an interop
// handle for functions that accept *BSONDocBuilder parameters.
type BSONDocBuilder struct {
	doc bson.D
}
