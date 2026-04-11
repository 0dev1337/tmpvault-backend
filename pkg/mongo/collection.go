package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	driver "go.mongodb.org/mongo-driver/mongo"
)

// Collection wraps a mongo collection with common CRUD helpers.
type Collection struct {
	coll *driver.Collection
}

// Raw returns the underlying driver collection (nil if receiver is nil).
func (col *Collection) Raw() *driver.Collection {
	if col == nil {
		return nil
	}
	return col.coll
}

// InsertOne inserts a single document.
func (col *Collection) InsertOne(ctx context.Context, doc interface{}) (*driver.InsertOneResult, error) {
	if col == nil || col.coll == nil {
		return nil, ErrUninitialized
	}
	return col.coll.InsertOne(ctx, doc)
}

// InsertMany inserts multiple documents.
func (col *Collection) InsertMany(ctx context.Context, docs []interface{}) (*driver.InsertManyResult, error) {
	if col == nil || col.coll == nil {
		return nil, ErrUninitialized
	}
	return col.coll.InsertMany(ctx, docs)
}

// FindOne decodes the first matching document into result.
func (col *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
	if col == nil || col.coll == nil {
		return ErrUninitialized
	}
	return col.coll.FindOne(ctx, filter).Decode(result)
}

// Find returns all documents matching filter as bson.M maps.
func (col *Collection) Find(ctx context.Context, filter interface{}) ([]bson.M, error) {
	if col == nil || col.coll == nil {
		return nil, ErrUninitialized
	}
	cur, err := col.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []bson.M
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateOne updates a single document.
func (col *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*driver.UpdateResult, error) {
	if col == nil || col.coll == nil {
		return nil, ErrUninitialized
	}
	return col.coll.UpdateOne(ctx, filter, update)
}

// DeleteOne removes a single document.
func (col *Collection) DeleteOne(ctx context.Context, filter interface{}) (*driver.DeleteResult, error) {
	if col == nil || col.coll == nil {
		return nil, ErrUninitialized
	}
	return col.coll.DeleteOne(ctx, filter)
}
