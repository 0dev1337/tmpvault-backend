package mongo

import driver "go.mongodb.org/mongo-driver/mongo"

// Database wraps a mongo database handle.
type Database struct {
	db *driver.Database
}

// Collection returns a typed collection wrapper.
func (d *Database) Collection(name string) *Collection {
	if d == nil || d.db == nil {
		return nil
	}
	return &Collection{coll: d.db.Collection(name)}
}
