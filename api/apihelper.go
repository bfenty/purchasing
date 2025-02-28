package api

import (
	"fmt"
	"purchasing/config"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Field struct {
	Column string
	Alias  string
}

// boolToInt converts a boolean value to an integer (true -> 1, false -> 0)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// derefString safely dereferences a string pointer, returning an empty string if nil
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// derefInt safely dereferences an int pointer, returning 0 if nil
func derefInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

// derefFloat safely dereferences a float64 pointer, returning 0.0 if nil
func derefFloat(f *float64) float64 {
	if f != nil {
		return *f
	}
	return 0.0
}

// derefBool safely dereferences a bool pointer, returning false if nil
func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

// buildQuery dynamically constructs either a SELECT or REPLACE SQL query based on parameters.
// If no queryType is provided, it defaults to SELECT.
// For SELECT queries, it also builds a separate COUNT query for pagination.
func buildQuery(table string, selectedFields []Field, filterConditions map[string]string, queryType ...string) ([]interface{}, strings.Builder, string) {
	var queryArgs []interface{}
	var queryBuilder strings.Builder
	qType := "SELECT"
	if len(queryType) > 0 {
		qType = queryType[0]
	}

	switch qType {

	case "DELETE":
		// DELETE query - assumes filterConditions will contain the WHERE clause conditions
		queryBuilder.WriteString(fmt.Sprintf("DELETE FROM %s WHERE 1", table))
		for column, value := range filterConditions {
			if value != "" {
				queryArgs = append(queryArgs, value)
				queryBuilder.WriteString(fmt.Sprintf(" AND %s = ?", column))
			}
		}
		return queryArgs, queryBuilder, ""

	case "UPDATE":
		// UPDATE query - assumes filterConditions contains both SET values and WHERE conditions
		queryBuilder.WriteString(fmt.Sprintf("UPDATE %s SET ", table))
		setClauses := []string{}
		whereClauses := []string{}
		for _, field := range selectedFields {
			if value, exists := filterConditions[field.Column]; exists {
				setClauses = append(setClauses, fmt.Sprintf("%s = ?", field.Column))
				queryArgs = append(queryArgs, value)
			}
		}
		queryBuilder.WriteString(strings.Join(setClauses, ", "))
		queryBuilder.WriteString(" WHERE 1")
		for column, value := range filterConditions {
			if value != "" {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", column))
				queryArgs = append(queryArgs, value)
			}
		}
		if len(whereClauses) > 0 {
			queryBuilder.WriteString(" AND ")
			queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
		}
		return queryArgs, queryBuilder, ""
	case "SELECT":
		queryBuilder.WriteString("SELECT ")
		fields := []string{}
		for _, field := range selectedFields {
			fields = append(fields, fmt.Sprintf("%s AS %s", field.Column, field.Alias))
		}
		queryBuilder.WriteString(strings.Join(fields, ", "))
		queryBuilder.WriteString(" FROM ")
		queryBuilder.WriteString(table)
		queryBuilder.WriteString(" WHERE 1")

		for column, value := range filterConditions {
			if value != "" {
				if isStringColumn(table, column) {
					// For string columns, use LIKE with wildcards
					queryArgs = append(queryArgs, "%"+value+"%")
					queryBuilder.WriteString(fmt.Sprintf(" AND %s LIKE ?", column))
				} else {
					// For non-string columns, use exact match
					queryArgs = append(queryArgs, value)
					queryBuilder.WriteString(fmt.Sprintf(" AND %s = ?", column))
				}
			}
		}

		queryBuilder.WriteString(" ORDER BY modified DESC")

		countQuery := strings.Replace(queryBuilder.String(), "SELECT "+strings.Join(fields, ", "), "SELECT COUNT(*)", 1)
		return queryArgs, queryBuilder, countQuery

	case "REPLACE":
		queryBuilder.WriteString(fmt.Sprintf("REPLACE INTO %s (", table))
		columns := []string{}
		placeholders := []string{}
		for _, field := range selectedFields {
			columns = append(columns, field.Column)
			placeholders = append(placeholders, "?")
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
		queryBuilder.WriteString(") VALUES (")
		queryBuilder.WriteString(strings.Join(placeholders, ", "))
		queryBuilder.WriteString(")")

		for _, field := range selectedFields {
			if val, exists := filterConditions[field.Column]; exists {
				queryArgs = append(queryArgs, val)
			} else {
				queryArgs = append(queryArgs, nil)
			}
		}
		return queryArgs, queryBuilder, ""

	default:
		log.WithFields(log.Fields{"queryType": qType}).Error("Unsupported query type")
		return nil, strings.Builder{}, ""
	}
}

// columnTypeCache stores table column metadata in memory to avoid repeated schema lookups
var columnTypeCache = make(map[string]map[string]string)

// getColumnTypes queries INFORMATION_SCHEMA to retrieve the column types for a given table
// Results are cached to improve performance.
func getColumnTypes(table string) (map[string]string, error) {
	// Skip schema lookup if the table string contains a JOIN
	if strings.Contains(strings.ToUpper(table), "JOIN") {
		log.WithFields(log.Fields{"table": table}).Warn("Skipping column type lookup for joined tables")
		return make(map[string]string), nil
	}
	if cached, ok := columnTypeCache[table]; ok {
		log.WithFields(log.Fields{"table": table}).Debug("Using cached column types")
		return cached, nil
	}

	log.WithFields(log.Fields{"table": table}).Debug("Fetching column types from database")

	query := `
		SELECT COLUMN_NAME, DATA_TYPE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()
	`

	rows, err := config.DB.Query(query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes := make(map[string]string)
	for rows.Next() {
		var column, dataType string
		if err := rows.Scan(&column, &dataType); err != nil {
			return nil, err
		}
		columnTypes[column] = dataType
	}

	columnTypeCache[table] = columnTypes
	return columnTypes, nil
}

// isStringColumn determines if a given column in a table is a string type
// This uses cached column metadata fetched from INFORMATION_SCHEMA
func isStringColumn(table, column string) bool {
	columnTypes, err := getColumnTypes(table)
	if err != nil {
		log.WithFields(log.Fields{"table": table, "error": err}).Error("Failed to fetch column types")
		return false
	}

	dataType, exists := columnTypes[column]
	if !exists {
		log.WithFields(log.Fields{"table": table, "column": column}).Warn("Column not found in metadata")
		return false
	}

	stringTypes := map[string]bool{
		"char": true, "varchar": true, "text": true, "tinytext": true, "mediumtext": true, "longtext": true,
	}

	return stringTypes[strings.ToLower(dataType)]
}
