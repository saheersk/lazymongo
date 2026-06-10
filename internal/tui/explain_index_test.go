package tui

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTopLevelFilterFields(t *testing.T) {
	cases := []struct {
		name   string
		filter bson.M
		want   []string
	}{
		{"nil filter", nil, nil},
		{"simple fields sorted", bson.M{"status": "x", "age": bson.M{"$gt": 18}}, []string{"age", "status"}},
		{"skips operators and _id", bson.M{"$or": bson.A{}, "_id": 1, "name": "a"}, []string{"name"}},
		{"only operators", bson.M{"$and": bson.A{}}, nil},
	}
	for _, c := range cases {
		if got := topLevelFilterFields(c.filter); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
