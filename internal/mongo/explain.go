package mongo

import (
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ExplainQuery runs explain("executionStats") for the given filter + sort
// and returns key execution stats plus the raw output for detail rendering.
func (c *Client) ExplainQuery(dbName, colName string, filter bson.M, sort bson.D) (msg.ExplainStats, error) {
	ctx, cancel := opCtx()
	defer cancel()

	if filter == nil {
		filter = bson.M{}
	}

	findCmd := bson.D{
		{Key: "find", Value: colName},
		{Key: "filter", Value: filter},
	}
	if len(sort) > 0 {
		findCmd = append(findCmd, bson.E{Key: "sort", Value: sort})
	}

	cmd := bson.D{
		{Key: "explain", Value: findCmd},
		{Key: "verbosity", Value: "executionStats"},
	}

	var raw bson.M
	if err := c.inner.Database(dbName).RunCommand(ctx, cmd).Decode(&raw); err != nil {
		return msg.ExplainStats{DB: dbName, Col: colName}, err
	}

	stats := msg.ExplainStats{DB: dbName, Col: colName, Raw: raw}

	if es := extractBSONDoc(raw["executionStats"]); es != nil {
		stats.NReturned = toInt64(es["nReturned"])
		stats.DocsExamined = toInt64(es["totalDocsExamined"])
		stats.KeysExamined = toInt64(es["totalKeysExamined"])
		stats.ExecutionTimeMs = toInt64(es["executionTimeMillis"])
	}

	if qp := extractBSONDoc(raw["queryPlanner"]); qp != nil {
		if wp := extractBSONDoc(qp["winningPlan"]); wp != nil {
			stats.IndexUsed = winningPlanIndex(wp)
		}
	}

	return stats, nil
}

// winningPlanIndex walks the winning-plan tree looking for an IXSCAN stage.
func winningPlanIndex(plan map[string]interface{}) string {
	if stage, _ := plan["stage"].(string); stage == "IXSCAN" {
		if name, _ := plan["indexName"].(string); name != "" {
			return name
		}
	}
	if is := extractBSONDoc(plan["inputStage"]); is != nil {
		return winningPlanIndex(is)
	}
	return ""
}
